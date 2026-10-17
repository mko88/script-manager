package ui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"script-manager/internal/config"
	"script-manager/internal/update"
	"script-manager/internal/update/testserver"
)

func updateServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	for _, b := range update.Binaries {
		os.WriteFile(filepath.Join(dir, b), []byte(b+" "+tag), 0o755)
	}
	rel, err := testserver.FromDir(dir, tag)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(testserver.Handler(rel, nil))
	t.Cleanup(srv.Close)
	return srv
}

func updateApp(t *testing.T, baseURL, current string, onStartup bool) (*App, string) {
	t.Helper()
	t.Setenv(update.EnvBaseURL, baseURL)
	dir := t.TempDir()
	for _, b := range update.Binaries {
		os.WriteFile(filepath.Join(dir, b), []byte(b+" current"), 0o755)
	}
	a := NewApp(&config.Config{}, nil, nil)
	a.EnableUpdates(update.NewClient(current, "linux", "amd64"), filepath.Join(dir, "script-manager"), onStartup)
	return a, dir
}

func run(a *App, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	_, next := a.Update(cmd())
	return next
}

func key(s string) tea.KeyMsg {
	if s == "ctrl+u" {
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestStartupCheckShowsHint(t *testing.T) {
	srv := updateServer(t, "v1.6.0")
	a, _ := updateApp(t, srv.URL, "v1.5.1", true)
	run(a, a.startupUpdateCheck())
	if a.status.update != "v1.6.0" {
		t.Errorf("hint = %q, want v1.6.0", a.status.update)
	}
	if a.updateConfirm {
		t.Error("startup check must not prompt")
	}
}

func TestStartupCheckOffOrDev(t *testing.T) {
	srv := updateServer(t, "v1.6.0")
	a, _ := updateApp(t, srv.URL, "v1.5.1", false)
	if a.startupUpdateCheck() != nil {
		t.Error("checked with the pref off")
	}
	a, _ = updateApp(t, srv.URL, "dev", true)
	if a.startupUpdateCheck() != nil {
		t.Error("a dev build checked")
	}
}

func TestStartupCheckErrorIsSilent(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	a, _ := updateApp(t, srv.URL, "v1.5.1", true)
	run(a, a.startupUpdateCheck())
	if a.status.message != "" {
		t.Errorf("startup error shown: %q", a.status.message)
	}
}

func TestManualCheckErrorIsShown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	a, _ := updateApp(t, srv.URL, "v1.5.1", false)
	_, cmd := a.Update(key("ctrl+u"))
	run(a, cmd)
	if !strings.Contains(a.status.message, "Update check failed") {
		t.Errorf("message = %q", a.status.message)
	}
}

func TestUpdateKeyConfirmCancel(t *testing.T) {
	srv := updateServer(t, "v1.6.0")
	a, dir := updateApp(t, srv.URL, "v1.5.1", false)
	_, cmd := a.Update(key("ctrl+u"))
	run(a, cmd)
	if !a.updateConfirm {
		t.Fatalf("no confirm prompt; message %q", a.status.message)
	}
	a.Update(key("n"))
	if a.updateConfirm || a.installing || a.restart {
		t.Error("cancel did not cancel")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "script-manager"))
	if string(data) != "script-manager current" {
		t.Error("binary changed after cancel")
	}
}

func TestUpdateKeyConfirmInstalls(t *testing.T) {
	srv := updateServer(t, "v1.6.0")
	a, dir := updateApp(t, srv.URL, "v1.5.1", true)
	run(a, a.startupUpdateCheck())
	a.Update(key("ctrl+u"))
	_, cmd := a.Update(key("y"))
	if !a.installing {
		t.Fatal("not installing after y")
	}
	if a.Update(key("q")); a.restart {
		t.Fatal("quit accepted mid-install")
	}
	next := run(a, cmd)
	if !a.RestartRequested() {
		t.Fatalf("restart not requested; message %q", a.status.message)
	}
	if next == nil {
		t.Fatal("no quit command")
	}
	if _, ok := next().(tea.QuitMsg); !ok {
		t.Error("install did not quit")
	}
	for _, b := range update.Binaries {
		data, _ := os.ReadFile(filepath.Join(dir, b))
		if string(data) != b+" v1.6.0" {
			t.Errorf("%s = %q", b, data)
		}
	}
}

func TestStatusBarShowsUpdateHint(t *testing.T) {
	bar := newStatusBarTile()
	bar.SetUpdateHint("v1.6.0", true)
	bar.Size.Width = 120
	if !strings.Contains(bar.View(), "Update to v1.6.0") {
		t.Error("hint missing from the status bar")
	}
}

func TestStartupCheckBlocksSecondCheck(t *testing.T) {
	srv := updateServer(t, "v1.6.0")
	a, _ := updateApp(t, srv.URL, "v1.5.1", true)
	pending := a.startupUpdateCheck()
	if !a.checkingUpdate {
		t.Fatal("startup check not marked as running")
	}
	if _, cmd := a.Update(key("ctrl+u")); cmd != nil {
		t.Error("Ctrl+U started a second check")
	}
	if !strings.Contains(a.status.message, "Checking") {
		t.Errorf("no feedback: %q", a.status.message)
	}
	run(a, pending)
	if a.checkingUpdate {
		t.Error("still marked as checking")
	}
	if !a.updateConfirm {
		t.Error("the startup result was not offered after Ctrl+U")
	}
}

func TestCheckResultUnderPINPromptOnlySetsHint(t *testing.T) {
	srv := updateServer(t, "v1.6.0")
	a, _ := updateApp(t, srv.URL, "v1.5.1", false)
	_, cmd := a.Update(key("ctrl+u"))
	a.pinPrompt = true
	a.status.SetMessage("PIN: ")
	run(a, cmd)
	if a.updateConfirm {
		t.Error("confirm opened over the PIN prompt")
	}
	if a.status.update != "v1.6.0" {
		t.Errorf("hint = %q", a.status.update)
	}
	if a.status.message != "PIN: " {
		t.Errorf("PIN prompt replaced by %q", a.status.message)
	}
}

func TestCtrlCCancelsInstall(t *testing.T) {
	srv := updateServer(t, "v1.6.0")
	a, dir := updateApp(t, srv.URL, "v1.5.1", true)
	run(a, a.startupUpdateCheck())
	a.Update(key("ctrl+u"))
	_, cmd := a.Update(key("y"))
	a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	run(a, cmd)
	if a.RestartRequested() || a.installing {
		t.Fatal("install not cancelled")
	}
	if !strings.Contains(a.status.message, "cancelled") {
		t.Errorf("message = %q", a.status.message)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "script-manager"))
	if string(data) != "script-manager current" {
		t.Error("binary changed after cancel")
	}
}

func TestNotInstallableReleasePointsToPage(t *testing.T) {
	dir := t.TempDir()
	for _, b := range update.Binaries {
		os.WriteFile(filepath.Join(dir, b), []byte(b), 0o755)
	}
	rel, err := testserver.FromDir(dir, "v1.6.0")
	if err != nil {
		t.Fatal(err)
	}
	delete(rel.Assets, update.ChecksumsAsset)
	srv := httptest.NewServer(testserver.Handler(rel, nil))
	t.Cleanup(srv.Close)

	a, _ := updateApp(t, srv.URL, "v1.5.1", true)
	run(a, a.startupUpdateCheck())
	a.status.Size.Width = 200
	if !strings.Contains(a.status.View(), "v1.6.0 on the release page") {
		t.Errorf("hint missing: %q", a.status.View())
	}
	a.Update(key("ctrl+u"))
	if a.updateConfirm {
		t.Error("offered to install a release without checksums")
	}
	if !strings.Contains(a.status.message, "release") {
		t.Errorf("message = %q", a.status.message)
	}
}

func TestInstallOutcomeMessages(t *testing.T) {
	cases := map[string]struct {
		err  error
		want string
	}{
		"stall":  {fmt.Errorf("downloading update: %w", update.ErrStalled), "stalled"},
		"cancel": {fmt.Errorf("downloading update: %w", context.Canceled), "cancelled"},
		"other":  {errors.New("boom"), "Update failed: boom"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			a := NewApp(&config.Config{}, nil, nil)
			a.installing = true
			a.onUpdateInstalled(updateInstalledMsg{c.err})
			if !strings.Contains(a.status.message, c.want) || a.restart {
				t.Errorf("message = %q, restart = %v", a.status.message, a.restart)
			}
			if name == "stall" && strings.Contains(a.status.message, "cancelled") {
				t.Error("stall reported as a cancel")
			}
		})
	}
}

func TestCtrlCTooLateAfterSwapStarts(t *testing.T) {
	a := NewApp(&config.Config{}, nil, nil)
	cancelled := false
	a.installing = true
	a.cancelInstall = func() { cancelled = true }
	a.swapping.Store(true)
	a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cancelled {
		t.Error("cancelled during the swap")
	}
	if !strings.Contains(a.status.message, "Too late to cancel") {
		t.Errorf("message = %q", a.status.message)
	}
}
