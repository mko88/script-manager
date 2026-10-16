package update

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
	"time"

	"script-manager/internal/update/testserver"
)

func TestCleanupOld(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "apps [x64]")
	os.Mkdir(dir, 0o755)
	write := func(name string, age time.Duration) {
		p := filepath.Join(dir, name)
		os.WriteFile(p, nil, 0o644)
		at := time.Now().Add(-age)
		os.Chtimes(p, at, at)
	}

	gone := []string{
		"sm-config-edit.exe.old", "script-manager.exe.old-1a2b3c4d",
		".script-manager-gui.exe.new-123", ".sm-update-probe-9",
	}
	kept := []string{
		"notes.old", "script-manager.exe", "script-manager-gui.exe",
		"script-manager.exe.old-backup", "script-manager.exe.old-1A2B3C4D", "script-manager.exe.old-1a2b3c4d5",
		".script-manager.exe.new-fresh", ".sm-update-probe-fresh",
	}
	for _, n := range gone {
		write(n, time.Hour)
	}
	for _, n := range kept {
		age := time.Hour
		if strings.HasSuffix(n, "fresh") {
			age = time.Minute
		}
		write(n, age)
	}

	CleanupOld(dir, "windows")
	CleanupOld("", "windows")
	for _, n := range gone {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			t.Errorf("%s survived", n)
		}
	}
	for _, n := range kept {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s was removed", n)
		}
	}
}

func TestReleasePage(t *testing.T) {
	rel := ghRelease{TagName: "v1.6.0", HTMLURL: "javascript:alert(1)"}

	strict := NewClient("v1.5.1", "windows", "amd64")
	strict.strict = true
	if got := strict.releasePage(rel); got != "https://github.com/mko88/script-manager/releases/tag/v1.6.0" {
		t.Errorf("strict page = %q", got)
	}
	if got := strict.releasePage(ghRelease{TagName: "v1.6.0;rm", HTMLURL: "https://x"}); got != "" {
		t.Errorf("strict page for a bad tag = %q", got)
	}

	t.Setenv(EnvBaseURL, "http://127.0.0.1:1")
	loose := NewClient("v1.5.1", "windows", "amd64")
	for in, want := range map[string]string{
		"javascript:alert(1)":            "",
		"file:///etc/passwd":             "",
		"https://example.com/r/v1.6.0":   "https://example.com/r/v1.6.0",
		"http://127.0.0.1:8765/r/v1.6.0": "http://127.0.0.1:8765/r/v1.6.0",
	} {
		rel.HTMLURL = in
		if got := loose.releasePage(rel); got != want {
			t.Errorf("releasePage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJavascriptReleaseURLDropped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v1.6.0","html_url":"javascript:alert(1)","assets":[]}`)
	}))
	defer srv.Close()
	t.Setenv(EnvBaseURL, srv.URL)
	res, err := NewClient("v1.6.0", "windows", "amd64").Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.ReleaseURL != "" {
		t.Errorf("ReleaseURL = %q", res.ReleaseURL)
	}
}

// stallingServer serves a release whose archive sends a few bytes and then
// stops until the client goes away.
func stallingServer(t *testing.T) *httptest.Server {
	t.Helper()
	rel := fakeRelease(t, "v1.6.0")
	win, _ := testserver.ArchiveNames("v1.6.0")
	inner := testserver.Handler(rel, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download/"+win {
			inner.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(rel.Assets[win])))
		w.Write(rel.Assets[win][:10])
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestStallIsReportedAsStall(t *testing.T) {
	old := idleTimeout
	idleTimeout = 100 * time.Millisecond
	defer func() { idleTimeout = old }()

	srv := stallingServer(t)
	t.Setenv(EnvBaseURL, srv.URL)
	dir := installDir(t, "windows")
	c := NewClient("v1.5.1", "windows", "amd64")
	err := checkAndInstall(t, c, dir)
	if !errors.Is(err, ErrStalled) || errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want ErrStalled only", err)
	}
	assertUnchanged(t, dir, "windows")
}

func TestUserCancelIsNotAStall(t *testing.T) {
	srv := stallingServer(t)
	t.Setenv(EnvBaseURL, srv.URL)
	dir := installDir(t, "windows")
	c := NewClient("v1.5.1", "windows", "amd64")
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	err = c.Install(ctx, res, filepath.Join(dir, "script-manager.exe"))
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrStalled) {
		t.Fatalf("err = %v, want context.Canceled only", err)
	}
	assertUnchanged(t, dir, "windows")
}

func TestBeforeSwapRuns(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := installDir(t, "windows")
	c := f.client("v1.5.1", "windows")
	called := false
	c.BeforeSwap = func() {
		called = true
		if data, _ := os.ReadFile(filepath.Join(dir, "script-manager.exe")); string(data) != "script-manager.exe current" {
			t.Error("BeforeSwap ran after the swap")
		}
	}
	if err := checkAndInstall(t, c, dir); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("BeforeSwap not called")
	}
}
