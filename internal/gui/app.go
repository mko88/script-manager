// Package gui is the Wails-bound backend for the desktop frontend. Every
// exported method on App becomes a callable binding in the frontend, under
// the "gui" namespace (window.go.gui.App).
package gui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	stdhtml "html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"script-manager/internal/action"
	"script-manager/internal/config"
	"script-manager/internal/render"

	"github.com/atotto/clipboard"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// wtWindowName is the Windows Terminal window name actions are run in. Every
// invocation of `wt -w <name>` reuses the window still open under that name
// (creating it on first use), giving the app a single dedicated WT instance
// instead of spawning a new window per action.
const wtWindowName = "script-manager"

// tempScriptPattern matches the temp files RunAction writes; see
// cleanupTempScripts for their lifecycle.
const tempScriptPattern = "script-manager-action-*"

// inlineOutPattern matches the temp output files RunActionInline writes;
// see cleanupTempScripts for their lifecycle.
const inlineOutPattern = "script-manager-inline-*"

// App is the Wails-bound backend.
type App struct {
	ctx     context.Context
	cfg     *config.Config
	load    func() (*config.Config, error)
	md      goldmark.Markdown
	exeDir  string
	loadErr error // from the initial load; the frontend fetches it once via LoadError

	// inlineMu guards the one inline (captured-output) run allowed at a time
	// app-wide, since the Command pane has only one output area to show it
	// in. inlineCmd is non-nil only while a run is in progress, letting a
	// concurrent CancelInlineAction call find and kill it. inlineOutPath
	// names the temp file GetInlineStatus re-reads on every poll; it stays
	// set after the run finishes so a final poll can still read the
	// complete output, and is only cleaned up when the next run starts.
	inlineMu       sync.Mutex
	inlineCmd      *exec.Cmd
	inlineOutPath  string
	inlineExitCode int
	inlineErrMsg   string
}

// NewApp builds the backend around a config loader, so an explicit -config
// path and F5 reloads go through the same resolution.
func NewApp(load func() (*config.Config, error)) *App {
	cfg, err := load()
	go cleanupTempScripts()
	return &App{
		cfg:     cfg,
		load:    load,
		exeDir:  exeDir(),
		loadErr: err,
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

// LoadError returns the error from the initial config load, if any, so the
// frontend can surface a startup failure the same way ReloadConfig errors are
// surfaced. Returns "" when the initial load succeeded.
func (a *App) LoadError() string {
	if a.loadErr == nil {
		return ""
	}
	return a.loadErr.Error()
}

// Startup is wired as the Wails OnStartup callback.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// exeDir returns the directory containing the running executable, or "" if
// it can't be determined.
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// cleanupTempScriptMinAge is how old a matched file must be before
// cleanupTempScripts will remove it. NewApp spawns the sweep in the
// background (not to slow down startup waiting on a directory scan) — with
// no minimum age at all, that sweep can race a script this same instance is
// about to write moments later (e.g. the window opens and the user
// immediately clicks Run/Run here) and delete it out from under the shell
// that's about to read it. Something orphaned by a previous, crashed run is
// at least this old by the time a new instance starts; anything newer is
// left for a later sweep rather than risked.
const cleanupTempScriptMinAge = 2 * time.Second

// cleanupTempScripts removes every action script left behind by previous
// runs, regardless of how old that makes it — as long as it's older than
// cleanupTempScriptMinAge (see there for why that floor exists). Every
// script wrapScript produces deletes itself once the launched shell actually
// starts executing it (see wrapScript); this is only the fallback for
// whatever that missed — e.g. the terminal or shell never started at all —
// so nothing lingers across restarts, however old it is.
func cleanupTempScripts() {
	cutoff := time.Now().Add(-cleanupTempScriptMinAge)
	for _, pattern := range []string{tempScriptPattern, inlineOutPattern} {
		matches, err := filepath.Glob(filepath.Join(os.TempDir(), pattern))
		if err != nil {
			continue
		}
		for _, path := range matches {
			if info, err := os.Stat(path); err == nil && info.ModTime().After(cutoff) {
				continue
			}
			os.Remove(path)
		}
	}
}

// ReloadConfig re-reads the config from disk. On total failure — nothing at
// all could be loaded, e.g. a missing file — the previously loaded config is
// kept and an error is returned so the frontend can surface it without losing
// the current view. A preferred file (e.g. config-win.yaml) failing to parse
// while a fallback (config.yaml) still loads is not total failure: the
// fallback is applied and its parse error comes back as a non-fatal warning
// string instead, since a Go error return would otherwise reject the whole
// call on the frontend regardless of the fallback having succeeded.
func (a *App) ReloadConfig() (string, error) {
	cfg, err := a.load()
	if cfg.SourcePath == "" {
		return "", err
	}
	a.cfg = cfg
	if err != nil {
		return err.Error(), nil
	}
	return "", nil
}

// TitlesDTO mirrors config.TitlesConfig for the frontend pane headers.
type TitlesDTO struct {
	Items   string `json:"items"`
	Actions string `json:"actions"`
	Details string `json:"details"`
	Command string `json:"command"`
}

func (a *App) GetTitles() TitlesDTO {
	return TitlesDTO{
		Items:   orDefault(a.cfg.Titles.Items, "Items"),
		Actions: orDefault(a.cfg.Titles.Actions, "Actions"),
		Details: orDefault(a.cfg.Titles.Details, "Details"),
		Command: orDefault(a.cfg.Titles.Command, "Command"),
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ActionGroupDTO is one entry of the config's optional actionGroups catalog
// — the frontend uses Color to paint group chips instead of showing every
// group with the same flat color; a group with no catalog entry (or no
// Color set) just falls back to the default chip styling.
type ActionGroupDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
}

func (a *App) GetActionGroups() []ActionGroupDTO {
	out := make([]ActionGroupDTO, len(a.cfg.ActionGroups))
	for i, g := range a.cfg.ActionGroups {
		out[i] = ActionGroupDTO{ID: g.ID, Title: g.Title, Color: g.Color}
	}
	return out
}

// ItemDTO is a row in the item list.
type ItemDTO struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

func (a *App) GetItems() []ItemDTO {
	items := make([]ItemDTO, len(a.cfg.Items))
	for i, item := range a.cfg.Items {
		items[i] = ItemDTO{Index: i, Label: a.renderListLabel(item)}
	}
	return items
}

// renderListLabel expands the list template for the item, falling back to
// the item's name when the template failed to parse or execute.
func (a *App) renderListLabel(item map[string]any) string {
	d := config.FindDisplay(a.cfg.Display, item)
	out, err := action.Expand(d.List, item)
	if err != nil {
		return fmt.Sprint(item[config.KeyName])
	}
	return out
}

func (a *App) mergedItem(item map[string]any) map[string]any {
	return action.Merge(a.cfg.Env, item)
}

func (a *App) itemAt(index int) map[string]any {
	if index < 0 || index >= len(a.cfg.Items) {
		return nil
	}
	return a.cfg.Items[index]
}

// ActionDTO is a row in the actions list for the selected item.
type ActionDTO struct {
	Index  int      `json:"index"`
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Groups []string `json:"groups"`
}

// GetActions returns the actions available for the item, in the same order
// GetActionDetail expects to receive back via ActionDTO.Index. Action IDs are
// optional (customActions rarely set one) so the index, not the ID, is the
// only reliable way to address a specific action.
func (a *App) GetActions(itemIndex int) []ActionDTO {
	item := a.itemAt(itemIndex)
	actions := config.ActionsForItem(a.cfg.Actions, item)
	out := make([]ActionDTO, len(actions))
	for i, act := range actions {
		out[i] = ActionDTO{Index: i, ID: act.ID, Title: act.Title, Groups: act.Groups}
	}
	return out
}

// ActionDetailDTO carries the expanded (but not yet run) command preview for
// the selected item/action pair.
type ActionDetailDTO struct {
	Description string `json:"description"`
	Cmd         string `json:"cmd"`
	NoWait      bool   `json:"noWait"`
}

func (a *App) GetActionDetail(itemIndex, actionIndex int) ActionDetailDTO {
	item := a.itemAt(itemIndex)
	if item == nil {
		return ActionDetailDTO{}
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return ActionDetailDTO{}
	}
	act := actions[actionIndex]
	merged := a.mergedItem(item)
	return ActionDetailDTO{
		Description: action.Preview(act.Description, merged),
		Cmd:         action.Preview(act.Cmd, merged),
		NoWait:      act.NoWait,
	}
}

// RunAction launches the item/action pair in a terminal window. Which
// terminal is used is resolved by resolveTerminal: an explicit config.
// Terminal override takes precedence, otherwise it auto-detects the most
// common terminal for the current OS (see windowsAutoDetect/linuxAutoDetect
// in terminal.go). macOS and other platforms get a clear error instead of a
// silent no-op.
func (a *App) RunAction(itemIndex, actionIndex int) error {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return fmt.Errorf("running actions is not supported on %s", runtime.GOOS)
	}
	item := a.itemAt(itemIndex)
	if item == nil {
		return fmt.Errorf("invalid item")
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return fmt.Errorf("invalid action")
	}
	if len(a.cfg.Shell) == 0 {
		return fmt.Errorf("no shell configured")
	}

	act := actions[actionIndex]
	merged := a.mergedItem(item)

	title := act.Title
	if name, ok := item[config.KeyName].(string); ok && name != "" {
		title = act.Title + " · " + name
	}

	// Resolve the terminal before writing anything to disk, so a missing or
	// misconfigured terminal fails fast without leaving a temp script behind.
	term, err := resolveTerminal(a.cfg.Terminal, runtime.GOOS, title, a.exeDir)
	if err != nil {
		return err
	}

	expandedCmd, err := action.Expand(act.Cmd, merged)
	if err != nil {
		return fmt.Errorf("cmd template error: %w", err)
	}
	script := wrapScript(shellBasename(a.cfg.Shell[0]), expandedCmd, !act.NoWait)
	scriptPath, err := writeTempScript(a.cfg.Shell[0], script)
	if err != nil {
		return fmt.Errorf("failed to write temp script: %w", err)
	}
	shellArgv := buildShellArgv(a.cfg.Shell, scriptPath, !act.NoWait)

	cmd := exec.Command(term.path, term.args(title, a.exeDir, shellArgv)...)
	if a.exeDir != "" {
		cmd.Dir = a.exeDir
	}
	cmd.Env = action.Env(merged)
	if err := cmd.Start(); err != nil {
		os.Remove(scriptPath)
		return err
	}
	return nil
}

// wrapScript wraps the expanded command with a self-delete of its own temp
// file, so cleanup is synchronized to actual execution instead of guessed by
// an external timer: whichever line runs the delete, the interpreter must
// already have opened (and read up to) that point in the file, so it can
// never race a terminal/shell that is merely slow to start — the previous
// approach (an external goroutine deleting the file on a timer) could win
// that race on a slow wt.exe/pwsh cold start, deleting the script before
// PowerShell ever opened it and making -File fail with "term ... is not
// recognized".
//
//   - pwsh/powershell: self-delete is the first line. PowerShell parses the
//     whole file before executing any of it, so this is also the fastest
//     point to get a secret-bearing script off disk.
//   - POSIX shells (bash, sh, zsh, dash, ksh): self-delete is the first line
//     too, for the same reason (unlinking a file another process still has
//     open is always safe on POSIX). When stayOpen is set, an epilogue at the
//     end waits for Enter before the terminal window closes.
//   - cmd: self-delete is the *last* line. Deleting a batch file as its very
//     first line is a well-known source of quirky behavior in cmd.exe (its
//     line-by-line reads can get confused); appending it after the real
//     command, once cmd.exe has already consumed everything before it, is
//     the safe, commonly-recommended placement.
func wrapScript(shellBase, script string, stayOpen bool) string {
	switch shellBase {
	case "pwsh", "powershell":
		return "Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue\n" + script + "\n"
	case "cmd":
		return script + "\r\ndel \"%~f0\"\r\n"
	default:
		var b strings.Builder
		b.WriteString("rm -f -- \"$0\"\n")
		b.WriteString(script)
		b.WriteString("\n")
		if stayOpen {
			b.WriteString("__status=$?\n")
			b.WriteString("printf '\\n[exit status %s] Press Enter to close...' \"$__status\"\n")
			b.WriteString("read -r __line\n")
		}
		return b.String()
	}
}

// writeTempScript writes script to a new temp file with an extension the
// target shell recognizes, and returns its path. Running the script from a
// file — rather than inlining it as a single -Command/-c argument — avoids
// depending on the terminal launcher's reconstruction of the argv surviving
// embedded newlines and quotes, which is unreliable for anything beyond a
// trivial one-liner. script is expected to already be wrapped by wrapScript,
// so it deletes this very file once the shell starts executing it; note the
// expanded command (including any masked values) is on disk in plain text
// until then.
func writeTempScript(shellBin, script string) (string, error) {
	ext := ".txt"
	switch shellBasename(shellBin) {
	case "pwsh", "powershell":
		ext = ".ps1"
	case "cmd":
		ext = ".bat"
	case "bash", "sh", "zsh", "dash", "ksh":
		ext = ".sh"
	}
	f, err := os.CreateTemp("", tempScriptPattern+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(script); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func shellBasename(shellBin string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(shellBin)), ".exe")
}

// buildShellArgv returns the full argv (shell binary + args) that runs
// scriptPath, for the given shell. When stayOpen is true it uses a
// shell-specific flag so the tab remains open (and the output visible) after
// the script finishes, rather than closing immediately.
func buildShellArgv(shell []string, scriptPath string, stayOpen bool) []string {
	switch shellBasename(shell[0]) {
	case "pwsh", "powershell":
		argv := []string{shell[0]}
		for _, a := range shell[1:] {
			if strings.EqualFold(a, "-command") {
				continue
			}
			argv = append(argv, a)
		}
		if stayOpen {
			argv = append(argv, "-NoExit")
		}
		return append(argv, "-File", scriptPath)
	case "cmd":
		flag := "/c"
		if stayOpen {
			flag = "/k"
		}
		return []string{shell[0], flag, scriptPath}
	default:
		// POSIX shells: -c makes the next argument a command *string*, which
		// would try to execute the script path as a program (and fail without
		// an exec bit). Strip it so the path is read as a script file, the
		// same way the -Command strip works for pwsh above.
		argv := []string{shell[0]}
		for _, a := range shell[1:] {
			if a == "-c" {
				continue
			}
			argv = append(argv, a)
		}
		return append(argv, scriptPath)
	}
}

// DetailsDTO is the rendered details pane: HTML plus the copyable values
// found in the source template, in the order they appear. MissingFields
// lists template fields the item lacks (rendered as <nil> in the HTML); the
// frontend shows them in a pinned warning bar rather than inline markdown.
type DetailsDTO struct {
	Html          string   `json:"html"`
	CopyValues    []string `json:"copyValues"`
	CopyMasked    []bool   `json:"copyMasked"`
	MissingFields []string `json:"missingFields"`
}

// codeTagRe matches a single inline <code>...</code> element as emitted by
// goldmark for a backtick span. Code fences render as <pre><code>, which this
// intentionally does not match since fenced blocks aren't used as copy targets.
var codeTagRe = regexp.MustCompile(`<code>(.*?)</code>`)

func (a *App) GetItemDetails(itemIndex int) DetailsDTO {
	item := a.itemAt(itemIndex)
	if item == nil {
		return DetailsDTO{}
	}
	merged := a.mergedItem(item)
	d := config.FindDisplay(a.cfg.Display, merged)
	funcMap := template.FuncMap{"mask": render.MaskFunc}
	tmpl, err := template.New("detail").Funcs(funcMap).Parse(d.Details)
	if err != nil {
		return DetailsDTO{Html: "<pre>details template error: " + err.Error() + "</pre>"}
	}
	data, missing := render.FillMissingFields(tmpl, merged)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return DetailsDTO{Html: "<pre>details template error: " + err.Error() + "</pre>"}
	}
	expanded := render.ExpandAllEnv(buf.String(), merged)
	expanded = render.ExpandConfigFile(expanded, a.cfg.SourcePath)

	displayMd, copyValues, copyMasked := render.ProcessMaskSpans(expanded)

	var htmlBuf bytes.Buffer
	if err := a.md.Convert([]byte(displayMd), &htmlBuf); err != nil {
		return DetailsDTO{Html: "<pre>" + strings.TrimSpace(displayMd) + "</pre>", MissingFields: missing}
	}

	idx := -1
	htmlOut := codeTagRe.ReplaceAllStringFunc(htmlBuf.String(), func(match string) string {
		idx++
		sub := codeTagRe.FindStringSubmatch(match)
		inner := sub[1]
		masked := idx < len(copyMasked) && copyMasked[idx]
		cls := "copy-value"
		titleAttr := ""
		if masked {
			cls += " copy-value-masked"
			// A genuine secret must never surface on hover, but a value
			// masked only because it spans multiple lines isn't sensitive —
			// just too long to inline — so it's safe to preview in full via
			// the native title tooltip.
			if idx < len(copyValues) && strings.Contains(copyValues[idx], "\n") {
				titleAttr = ` title="` + stdhtml.EscapeString(copyValues[idx]) + `"`
			}
		}
		return `<code class="` + cls + `"` + titleAttr + ` data-copy-idx="` + strconv.Itoa(idx) + `">` + inner + `</code>`
	})

	return DetailsDTO{Html: htmlOut, CopyValues: copyValues, CopyMasked: copyMasked, MissingFields: missing}
}

// CopyToClipboard writes value to the system clipboard.
func (a *App) CopyToClipboard(value string) error {
	return clipboard.WriteAll(value)
}

// buildInlineCmd resolves itemIndex/actionIndex into a ready-to-Start
// *exec.Cmd for an inline (captured-output) run — shared by RunActionInline's
// blocking fallback and handleRunHTTP's live-streamed version, so the two
// never drift in how they build the command itself, only in how they capture
// its output. cleanup removes the temp script file and must be called once
// the command has actually been started (the script self-deletes as it
// runs — see wrapScript — so this only covers whatever that missed).
func (a *App) buildInlineCmd(itemIndex, actionIndex int) (cmd *exec.Cmd, cleanup func(), err error) {
	item := a.itemAt(itemIndex)
	if item == nil {
		return nil, nil, fmt.Errorf("invalid item")
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return nil, nil, fmt.Errorf("invalid action")
	}
	if len(a.cfg.Shell) == 0 {
		return nil, nil, fmt.Errorf("no shell configured")
	}

	act := actions[actionIndex]
	merged := a.mergedItem(item)

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
	if a.exeDir != "" {
		cmd.Dir = a.exeDir
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

// InlineStatusDTO is GetInlineStatus's snapshot of the one app-wide inline
// run. Output is whatever has been captured so far — the full thing once
// Running is false — and ExitCode/ErrMsg are only meaningful once Running is
// false and a run has actually completed.
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
// learn when the process finishes. Only one inline run is allowed at a time
// app-wide, since the Command pane has only one output area to show it in.
func (a *App) RunActionInline(itemIndex, actionIndex int) error {
	a.inlineMu.Lock()
	if a.inlineCmd != nil {
		a.inlineMu.Unlock()
		return fmt.Errorf("a command is already running")
	}
	// The previous run's output file (if any) is only cleaned up here, not
	// right when that run finished, so a final GetInlineStatus poll after
	// completion can still read it.
	if a.inlineOutPath != "" {
		os.Remove(a.inlineOutPath)
		a.inlineOutPath = ""
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
	a.inlineCmd = cmd
	a.inlineOutPath = outFile.Name()
	a.inlineExitCode = 0
	a.inlineErrMsg = ""
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
		a.inlineExitCode = exitCode
		a.inlineErrMsg = errMsg
		a.inlineCmd = nil
		a.inlineMu.Unlock()
	}()

	return nil
}

// GetInlineStatus reports the current state of the one app-wide inline run —
// the frontend polls this on a short timer after calling RunActionInline to
// get a live-updating view of the output, rather than being pushed a
// completion event. Output is read fresh from the run's temp file on every
// call, so it reflects whatever the process has written so far; once
// Running is false, it's the complete output and ExitCode/ErrMsg are final.
func (a *App) GetInlineStatus() InlineStatusDTO {
	a.inlineMu.Lock()
	running := a.inlineCmd != nil
	outPath := a.inlineOutPath
	exitCode := a.inlineExitCode
	errMsg := a.inlineErrMsg
	a.inlineMu.Unlock()

	output := ""
	if outPath != "" {
		if data, err := os.ReadFile(outPath); err == nil {
			output = string(data)
		}
	}
	return InlineStatusDTO{Running: running, Output: output, ExitCode: exitCode, ErrMsg: errMsg}
}

// CancelInlineAction terminates the currently running inline action, if any,
// killing its whole process tree — plain Process.Kill only signals the shell
// directly; SIGKILL can't be caught or forwarded, so the shell would die
// without ever killing whatever foreground command it was running, silently
// orphaning it instead.
func (a *App) CancelInlineAction() error {
	a.inlineMu.Lock()
	cmd := a.inlineCmd
	a.inlineMu.Unlock()
	if cmd == nil {
		return fmt.Errorf("no command is running")
	}
	return killProcessTree(cmd)
}
