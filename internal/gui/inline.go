package gui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"script-manager/internal/action"
	"script-manager/internal/config"
)

const inlineOutPattern = "script-manager-inline-*"

type inlineKey struct {
	itemIndex   int
	actionIndex int
}

type inlineRun struct {
	cmd      *exec.Cmd
	outPath  string
	exitCode int
	errMsg   string
}

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
	merged, err := a.mergedItemForRun(item, act)
	if err != nil {
		return nil, nil, err
	}

	if len(a.cfg.Shell) == 0 {
		return nil, nil, fmt.Errorf("no shell configured")
	}

	if act.Script != "" {
		expandedScript, err := action.Expand(act.Script, merged)
		if err != nil {
			return nil, nil, fmt.Errorf("script path template error: %w", err)
		}
		wrapped := action.WrapScriptFile(action.ShellBasename(a.cfg.Shell[0]), expandedScript, false)
		scriptPath, err := action.WriteTempScript(a.cfg.Shell[0], wrapped)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to write temp script: %w", err)
		}
		shellArgv := action.ScriptArgv(a.cfg.Shell, scriptPath, false)
		cmd = exec.Command(shellArgv[0], shellArgv[1:]...)
		if a.appDataDir != "" {
			cmd.Dir = a.appDataDir
		}
		cmd.Env = action.Env(merged)
		setProcessGroup(cmd)
		return cmd, func() { os.Remove(scriptPath) }, nil
	}

	expandedCmd, err := action.Expand(act.Cmd, merged)
	if err != nil {
		return nil, nil, fmt.Errorf("cmd template error: %w", err)
	}
	script := wrapScript(action.ShellBasename(a.cfg.Shell[0]), expandedCmd, false)
	scriptPath, err := action.WriteTempScript(a.cfg.Shell[0], script)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write temp script: %w", err)
	}

	shellArgv := action.ScriptArgv(a.cfg.Shell, scriptPath, false)
	cmd = exec.Command(shellArgv[0], shellArgv[1:]...)
	if a.appDataDir != "" {
		cmd.Dir = a.appDataDir
	}
	cmd.Env = action.Env(merged)
	setProcessGroup(cmd)
	return cmd, func() { os.Remove(scriptPath) }, nil
}

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

type InlineStatusDTO struct {
	Running  bool   `json:"running"`
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode"`
	ErrMsg   string `json:"errMsg"`
}

func (a *App) RunActionInline(itemIndex, actionIndex int) error {
	key := inlineKey{itemIndex, actionIndex}

	a.inlineMu.Lock()
	if run, ok := a.inlineRuns[key]; ok {
		if run.cmd != nil {
			a.inlineMu.Unlock()
			return fmt.Errorf("this action is already running")
		}
		os.Remove(run.outPath)
		delete(a.inlineRuns, key)
	}
	a.inlineMu.Unlock()

	cmd, cleanup, err := a.buildInlineCmd(itemIndex, actionIndex)
	if err != nil {
		return err
	}

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
		outFile.Close()
		cleanup()

		exitCode, errMsg := exitCodeOf(waitErr)

		a.inlineMu.Lock()
		a.inlineRuns[key] = &inlineRun{outPath: outFile.Name(), exitCode: exitCode, errMsg: errMsg}
		a.inlineMu.Unlock()
	}()

	return nil
}

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

func (a *App) CancelInlineAction(itemIndex, actionIndex int) error {
	a.inlineMu.Lock()
	run, ok := a.inlineRuns[inlineKey{itemIndex, actionIndex}]
	a.inlineMu.Unlock()
	if !ok || run.cmd == nil {
		return fmt.Errorf("no command is running")
	}
	return killProcessTree(run.cmd)
}
