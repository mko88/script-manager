package ui

import (
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
	a.EnableUpdates(update.NewClient(current, "linux", "amd64"), dir, onStartup)
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
	bar.SetUpdateHint("v1.6.0")
	bar.Size.Width = 120
	if !strings.Contains(bar.View(), "Update to v1.6.0") {
		t.Error("hint missing from the status bar")
	}
}
