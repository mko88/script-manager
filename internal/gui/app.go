// Package gui is the Wails-bound backend for the desktop frontend. Every
// exported method on App becomes a callable binding in the frontend, under
// the "gui" namespace (window.go.gui.App).
package gui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"

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

// App is the Wails-bound backend.
type App struct {
	ctx    context.Context
	cfg    *config.Config
	load   func() (*config.Config, error)
	md     goldmark.Markdown
	exeDir string
}

// NewApp builds the backend around a config loader, so an explicit -config
// path and F5 reloads go through the same resolution.
func NewApp(load func() (*config.Config, error)) *App {
	cfg, _ := load()
	go cleanupTempScripts()
	return &App{
		cfg:    cfg,
		load:   load,
		exeDir: exeDir(),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
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

// cleanupTempScripts removes every action script left behind by previous
// runs, regardless of age. Every script wrapScript produces deletes itself
// once the launched shell actually starts executing it (see wrapScript); this
// is only the fallback for whatever that missed — e.g. the terminal or shell
// never started at all — so nothing lingers across restarts, however old it
// is.
func cleanupTempScripts() {
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), tempScriptPattern))
	if err != nil {
		return
	}
	for _, path := range matches {
		os.Remove(path)
	}
}

// ReloadConfig re-reads the config from disk. On failure — a missing file or
// a YAML syntax error — the previously loaded config is kept and an error is
// returned so the frontend can surface it without losing the current view.
func (a *App) ReloadConfig() error {
	cfg, err := a.load()
	if err != nil {
		return err
	}
	a.cfg = cfg
	return nil
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

// RunAction launches the item/action pair in a terminal window: on Windows
// as a new tab in the dedicated Windows Terminal window (reused across
// calls), on Linux in the first terminal emulator found on PATH (see
// linuxTerminals for the order). Other platforms get a clear error instead
// of a silent no-op.
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

	// Resolve the terminal before writing anything to disk, so a missing
	// terminal fails fast without leaving a temp script behind.
	var wtPath string
	var linTerm linuxTerminal
	var err error
	if runtime.GOOS == "windows" {
		if wtPath, err = exec.LookPath("wt.exe"); err != nil {
			return fmt.Errorf("Windows Terminal (wt.exe) not found in PATH")
		}
	} else {
		if linTerm, err = findLinuxTerminal(); err != nil {
			return err
		}
	}

	act := actions[actionIndex]
	merged := a.mergedItem(item)
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

	title := act.Title
	if name, ok := item[config.KeyName].(string); ok && name != "" {
		title = act.Title + " · " + name
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		wtArgs := []string{"-w", wtWindowName, "new-tab", "--title", title}
		if a.exeDir != "" {
			wtArgs = append(wtArgs, "-d", a.exeDir)
		}
		wtArgs = append(wtArgs, "--")
		wtArgs = append(wtArgs, shellArgv...)
		cmd = exec.Command(wtPath, wtArgs...)
	} else {
		cmd = exec.Command(linTerm.path, linTerm.args(title, a.exeDir, shellArgv)...)
		if a.exeDir != "" {
			cmd.Dir = a.exeDir
		}
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
		if masked {
			cls += " copy-value-masked"
		}
		return `<code class="` + cls + `" data-copy-idx="` + strconv.Itoa(idx) + `">` + inner + `</code>`
	})

	return DetailsDTO{Html: htmlOut, CopyValues: copyValues, CopyMasked: copyMasked, MissingFields: missing}
}

// CopyToClipboard writes value to the system clipboard.
func (a *App) CopyToClipboard(value string) error {
	return clipboard.WriteAll(value)
}
