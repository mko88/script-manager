package gui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"script-manager/internal/action"
	"script-manager/internal/config"
)

// inlineOutPattern matches the temp output files RunActionInline writes;
// see cleanupTempScripts for their lifecycle.
const inlineOutPattern = "script-manager-inline-*"

// inlineKey identifies one item/action pair's inline (captured-output) run.
type inlineKey struct {
	itemIndex   int
	actionIndex int
}

// inlineRun is one item/action pair's inline run state. cmd is non-nil only
// while the process is actually running, letting a concurrent
// CancelInlineAction call find and kill it; it goes nil once the process
// exits, but the entry itself stays in App.inlineRuns (outPath/exitCode/
// errMsg intact) so a later GetInlineStatus poll — e.g. after switching back
// to that action in the UI — can still read the finished result, right up
// until a new run for that same key replaces it.
type inlineRun struct {
	cmd      *exec.Cmd
	outPath  string
	exitCode int
	errMsg   string
}

// buildInlineCmd resolves itemIndex/actionIndex into a ready-to-Start
// *exec.Cmd for an inline (captured-output) run, the shared groundwork
// RunActionInline builds on before wiring up output capture and starting it.
// cleanup removes the temp script file and must be called once the command
// has actually been started (the script self-deletes as it runs — see
// wrapScript — so this only covers whatever that missed).
func (a *App) buildInlineCmd(itemIndex, actionIndex int) (cmd *exec.Cmd, cleanup func(), err error) {
	item := a.itemAt(itemIndex)
	if item == nil {
		return nil, nil, fmt.Errorf("invalid item")
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return nil, nil, fmt.Errorf("invalid action")
	}

	act := actions[actionIndex]
	if act.Interactive {
		return nil, nil, fmt.Errorf("this action is interactive and needs a real terminal")
	}
	merged := a.mergedItem(item)

	if act.Script != "" {
		expandedScript, err := action.Expand(act.Script, merged)
		if err != nil {
			return nil, nil, fmt.Errorf("script path template error: %w", err)
		}
		cmd = exec.Command(expandedScript)
		if a.appDataDir != "" {
			cmd.Dir = a.appDataDir
		}
		cmd.Env = action.Env(merged)
		setProcessGroup(cmd)
		return cmd, func() {}, nil
	}

	if len(a.cfg.Shell) == 0 {
		return nil, nil, fmt.Errorf("no shell configured")
	}
	expandedCmd, err := action.Expand(act.Cmd, merged)
	if err != nil {
		return nil, nil, fmt.Errorf("cmd template error: %w", err)
	}
	// No stayOpen epilogue: there's no interactive terminal for a "press
	// Enter to close" prompt to wait in, and NoWait's terminal-window
	// semantics don't apply to a run that never opens one.
	script := wrapScript(shellBasename(a.cfg.Shell[0]), expandedCmd, false)
	scriptPath, err := writeTempScript(a.cfg.Shell[0], script)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write temp script: %w", err)
	}

	shellArgv := buildShellArgv(a.cfg.Shell, scriptPath, false)
	cmd = exec.Command(shellArgv[0], shellArgv[1:]...)
	if a.appDataDir != "" {
		cmd.Dir = a.appDataDir
	}
	cmd.Env = action.Env(merged)
	// Stdin is deliberately left disconnected: an inline run is for a
	// command that isn't expected to need input. Go gives an unset Stdin an
	// immediate EOF rather than hanging, so a command that unexpectedly
	// prompts fails fast instead of blocking forever with no terminal for
	// anyone to type into.
	setProcessGroup(cmd)
	return cmd, func() { os.Remove(scriptPath) }, nil
}

// exitCodeOf turns a cmd.Wait() error into an (exitCode, errMsg) pair — nil
// means exit 0, a non-*exec.ExitError (e.g. the process was killed by a
// signal) reports -1 since there's no real exit code to extract.
func exitCodeOf(waitErr error) (exitCode int, errMsg string) {
	if waitErr == nil {
		return 0, ""
	}
	errMsg = waitErr.Error()
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode(), errMsg
	}
	return -1, errMsg
}

// InlineStatusDTO is GetInlineStatus's snapshot of one item/action pair's
// inline run. Output is whatever has been captured so far — the full thing
// once Running is false — and ExitCode/ErrMsg are only meaningful once
// Running is false and a run has actually completed. The zero value (all
// fields empty/zero) means this pair has never been run, or its last run's
// state was already overwritten by a newer one for the same pair.
type InlineStatusDTO struct {
	Running  bool   `json:"running"`
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode"`
	ErrMsg   string `json:"errMsg"`
}

// RunActionInline starts the item/action pair running with its output
// captured instead of handed off to an external terminal — meant for a
// command that isn't expected to need interactive input, so its result can
// be read right in the Command pane rather than needing a separate terminal
// window. It returns as soon as the process starts; the frontend polls
// GetInlineStatus on a short timer to read the output captured so far and
// learn when the process finishes. Different item/action pairs may run
// concurrently — switching to a different action in the UI doesn't stop one
// already running — but the same pair can't be started twice at once.
func (a *App) RunActionInline(itemIndex, actionIndex int) error {
	key := inlineKey{itemIndex, actionIndex}

	a.inlineMu.Lock()
	if run, ok := a.inlineRuns[key]; ok {
		if run.cmd != nil {
			a.inlineMu.Unlock()
			return fmt.Errorf("this action is already running")
		}
		// A previous, finished run's output file for this same key is only
		// cleaned up here, not right when that run finished, so a
		// GetInlineStatus poll after completion (e.g. switching back to
		// this action in the UI) can still read it up until now. The map
		// entry itself is removed too, not just the file: if buildInlineCmd
		// or cmd.Start below fails, this key must go back to "never run"
		// rather than keep reporting the previous run's now-deleted output
		// and stale exit code.
		os.Remove(run.outPath)
		delete(a.inlineRuns, key)
	}
	a.inlineMu.Unlock()

	cmd, cleanup, err := a.buildInlineCmd(itemIndex, actionIndex)
	if err != nil {
		return err
	}

	// Stdout and Stderr go to a real temp file, not an in-memory io.Writer:
	// both point at the very same file, so the child's writes to each still
	// interleave in true OS-level chronological order, same guarantee an
	// in-memory bytes.Buffer/io.Pipe would give — but backed by a real fd Go
	// just hands to the child directly, no internal copying goroutine
	// involved. GetInlineStatus re-reads this same file on every poll to
	// report the output captured so far, rather than the run accumulating
	// it in memory itself.
	outFile, err := os.CreateTemp("", inlineOutPattern+".log")
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to create output file: %w", err)
	}
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		os.Remove(outFile.Name())
		cleanup()
		return err
	}

	a.inlineMu.Lock()
	a.inlineRuns[key] = &inlineRun{cmd: cmd, outPath: outFile.Name()}
	a.inlineMu.Unlock()

	go func() {
		waitErr := cmd.Wait()
		// Only safe to close once Wait returns: os/exec's own copier isn't
		// involved here (outFile is a real fd, not a Go-managed io.Writer),
		// but the child itself may still have the fd open until this point.
		outFile.Close()
		cleanup()

		exitCode, errMsg := exitCodeOf(waitErr)

		a.inlineMu.Lock()
		a.inlineRuns[key] = &inlineRun{outPath: outFile.Name(), exitCode: exitCode, errMsg: errMsg}
		a.inlineMu.Unlock()
	}()

	return nil
}

// GetInlineStatus reports the current state of one item/action pair's
// inline run — the frontend polls this on a short timer after calling
// RunActionInline to get a live-updating view of the output, rather than
// being pushed a completion event. Output is read fresh from the run's temp
// file on every call, so it reflects whatever the process has written so
// far; once Running is false, it's the complete output and ExitCode/ErrMsg
// are final.
func (a *App) GetInlineStatus(itemIndex, actionIndex int) InlineStatusDTO {
	a.inlineMu.Lock()
	run, ok := a.inlineRuns[inlineKey{itemIndex, actionIndex}]
	a.inlineMu.Unlock()
	if !ok {
		return InlineStatusDTO{}
	}

	output := ""
	if run.outPath != "" {
		if data, err := os.ReadFile(run.outPath); err == nil {
			output = string(data)
		}
	}
	return InlineStatusDTO{Running: run.cmd != nil, Output: output, ExitCode: run.exitCode, ErrMsg: run.errMsg}
}

// CancelInlineAction terminates the given item/action pair's inline run, if
// it's currently running — killing its whole process tree, since plain
// Process.Kill only signals the shell directly and SIGKILL can't be caught
// or forwarded, silently orphaning whatever foreground command it was
// running otherwise.
func (a *App) CancelInlineAction(itemIndex, actionIndex int) error {
	a.inlineMu.Lock()
	run, ok := a.inlineRuns[inlineKey{itemIndex, actionIndex}]
	a.inlineMu.Unlock()
	if !ok || run.cmd == nil {
		return fmt.Errorf("no command is running")
	}
	return killProcessTree(run.cmd)
}
