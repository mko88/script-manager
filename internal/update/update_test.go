package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"script-manager/internal/update/testserver"
)

func TestVersionOrdering(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.6.0", "v1.5.1", true},
		{"v1.5.1", "v1.5.1", false},
		{"v1.5.0", "v1.5.1", false},
		{"v1.5.2", "v1.5.1-3-gabc1234", true},
		{"v1.5.1", "v1.5.1-3-gabc1234", false},
		{"v1.5.1", "v1.5.1-3-gabc1234-dirty", false},
		{"v1.5.1", "v1.5.0-dirty", true},
		{"v2.0.0", "v1.99.99", true},
		{"v1.10.0", "v1.9.0", true},
		{"v1.3.0.0", "v1.4.0", false},
		{"v1.4.0", "v1.3.0.0-2-gabc1234", true},
		{"v1.3.1.0", "v1.3.0.0", false},
	}
	for _, c := range cases {
		cur, ok := parseBuild(c.current)
		if !ok {
			t.Fatalf("parseBuild(%q) not ok", c.current)
		}
		latest, ok := parseTag(c.latest)
		if !ok {
			t.Fatalf("parseTag(%q) not ok", c.latest)
		}
		if got := latest.newerThan(cur); got != c.want {
			t.Errorf("%s newer than %s = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestNotCheckableBuilds(t *testing.T) {
	for _, v := range []string{"dev", "", "abc1234", "abc1234-dirty", "v1.5", "1.5.0-beta"} {
		if Checkable(v) {
			t.Errorf("Checkable(%q) = true", v)
		}
	}
	if !Checkable("v1.5.1-3-gabc1234-dirty") {
		t.Error("a git describe build should be checkable")
	}
}

func TestPrereleaseTagIsNotARelease(t *testing.T) {
	if _, ok := parseTag("v1.6.0-rc1"); ok {
		t.Error("parseTag accepted a prerelease suffix")
	}
}

// fakeRelease returns a release of tag for both platforms whose binaries
// contain "<name> <tag>".
func fakeRelease(t *testing.T, tag string) *testserver.Release {
	t.Helper()
	win, linux := testserver.ArchiveNames(tag)
	var winEntries, linuxEntries []testserver.Entry
	for _, b := range Binaries {
		winEntries = append(winEntries, testserver.Entry{Name: b + ".exe", Body: []byte(b + ".exe " + tag)})
		linuxEntries = append(linuxEntries, testserver.Entry{Name: "./" + b, Body: []byte(b + " " + tag)})
	}
	winEntries = append(winEntries, testserver.Entry{Name: "README.md", Body: []byte("readme")})
	linuxEntries = append(linuxEntries, testserver.Entry{Name: "./README.md", Body: []byte("readme")})
	rel := &testserver.Release{Tag: tag, Assets: map[string][]byte{}}
	var err error
	if rel.Assets[win], err = testserver.Zip(winEntries); err != nil {
		t.Fatal(err)
	}
	if rel.Assets[linux], err = testserver.TarGz(linuxEntries); err != nil {
		t.Fatal(err)
	}
	rel.Assets[ChecksumsAsset] = testserver.Checksums(rel.Assets, win, linux)
	return rel
}

type fixture struct {
	srv    *httptest.Server
	mu     sync.Mutex
	agents []string
}

func serve(t *testing.T, rel *testserver.Release) *fixture {
	t.Helper()
	f := &fixture{}
	f.srv = httptest.NewServer(testserver.Handler(rel, func(r *http.Request) {
		f.mu.Lock()
		f.agents = append(f.agents, r.UserAgent())
		f.mu.Unlock()
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fixture) client(current, goos string) *Client {
	t := os.Getenv(EnvBaseURL)
	os.Setenv(EnvBaseURL, f.srv.URL+"/")
	defer os.Setenv(EnvBaseURL, t)
	return NewClient(current, goos, "amd64")
}

func TestCheck(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	cases := []struct {
		current string
		want    bool
	}{
		{"v1.5.1", true},
		{"v1.5.1-4-g94b2835-dirty", true},
		{"v1.6.0", false},
		{"v1.6.0-2-gabc1234", false},
		{"v1.7.0", false},
	}
	for _, c := range cases {
		res, err := f.client(c.current, "windows").Check(context.Background())
		if err != nil {
			t.Fatalf("Check(%s): %v", c.current, err)
		}
		if res.Available != c.want {
			t.Errorf("Check(%s).Available = %v, want %v", c.current, res.Available, c.want)
		}
		if res.Latest != "v1.6.0" {
			t.Errorf("Latest = %q", res.Latest)
		}
	}
	for _, ua := range f.agents {
		if ua != userAgent {
			t.Errorf("User-Agent = %q", ua)
		}
	}
	if len(f.agents) == 0 {
		t.Error("no request reached the server")
	}
}

func TestDevBuildNeverChecks(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	_, err := f.client("dev", "windows").Check(context.Background())
	if !errors.Is(err, ErrNotCheckable) {
		t.Fatalf("err = %v, want ErrNotCheckable", err)
	}
	if len(f.agents) != 0 {
		t.Error("a dev build contacted the server")
	}
}

func TestPrereleaseIgnored(t *testing.T) {
	rel := fakeRelease(t, "v1.6.0")
	rel.Prerelease = true
	f := serve(t, rel)
	res, err := f.client("v1.5.1", "windows").Check(context.Background())
	if err != nil || res.Available {
		t.Fatalf("prerelease offered: %+v, %v", res, err)
	}
}

func TestLegacyLatestTagNotOffered(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.3.0.0"))
	res, err := f.client("v1.5.1", "windows").Check(context.Background())
	if err != nil || res.Available {
		t.Fatalf("legacy tag offered: %+v, %v", res, err)
	}
}

func TestDefaultBaseURL(t *testing.T) {
	old, had := os.LookupEnv(EnvBaseURL)
	os.Unsetenv(EnvBaseURL)
	defer func() {
		if had {
			os.Setenv(EnvBaseURL, old)
		}
	}()
	if c := NewClient("v1.0.0", "windows", "amd64"); c.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
}

// installDir creates the current binaries in a temp dir.
func installDir(t *testing.T, goos string) string {
	t.Helper()
	dir := t.TempDir()
	for _, b := range Binaries {
		n := binaryName(b, goos)
		if err := os.WriteFile(filepath.Join(dir, n), []byte(n+" current"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func readBinaries(t *testing.T, dir, goos string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, b := range Binaries {
		data, err := os.ReadFile(filepath.Join(dir, binaryName(b, goos)))
		if err != nil {
			t.Fatal(err)
		}
		out[b] = string(data)
	}
	return out
}

func assertUnchanged(t *testing.T, dir, goos string) {
	t.Helper()
	for b, got := range readBinaries(t, dir, goos) {
		if want := binaryName(b, goos) + " current"; got != want {
			t.Errorf("%s = %q, want it untouched", b, got)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != len(Binaries) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("dir has leftovers: %v", names)
	}
}

func checkAndInstall(t *testing.T, c *Client, dir string) error {
	t.Helper()
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !res.Available {
		t.Fatal("update not offered")
	}
	return c.Install(context.Background(), res, dir)
}

func TestInstallSwapsAllThree(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		t.Run(goos, func(t *testing.T) {
			f := serve(t, fakeRelease(t, "v1.6.0"))
			dir := installDir(t, goos)
			if err := checkAndInstall(t, f.client("v1.5.1", goos), dir); err != nil {
				t.Fatalf("Install: %v", err)
			}
			for b, got := range readBinaries(t, dir, goos) {
				if want := binaryName(b, goos) + " v1.6.0"; got != want {
					t.Errorf("%s = %q, want %q", b, got, want)
				}
				old, err := os.ReadFile(filepath.Join(dir, binaryName(b, goos)+".old"))
				if err != nil || string(old) != binaryName(b, goos)+" current" {
					t.Errorf("%s.old = %q, %v", b, old, err)
				}
				if runtime.GOOS != "windows" {
					st, _ := os.Stat(filepath.Join(dir, binaryName(b, goos)))
					if st.Mode().Perm()&0o111 == 0 {
						t.Errorf("%s lost its exec bit: %v", b, st.Mode())
					}
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
				t.Error("README.md was extracted")
			}

			CleanupOld(dir, goos)
			entries, _ := os.ReadDir(dir)
			if len(entries) != len(Binaries) {
				t.Errorf("after cleanup dir has %d entries", len(entries))
			}
		})
	}
}

func TestChecksumMismatchRefuses(t *testing.T) {
	rel := fakeRelease(t, "v1.6.0")
	win, _ := testserver.ArchiveNames("v1.6.0")
	rel.Assets[ChecksumsAsset] = testserver.Checksums(map[string][]byte{win: []byte("other")}, win)
	f := serve(t, rel)
	dir := installDir(t, "windows")
	err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("err = %v, want a SHA-256 failure", err)
	}
	assertUnchanged(t, dir, "windows")
}

func TestMissingChecksumsRefuses(t *testing.T) {
	rel := fakeRelease(t, "v1.6.0")
	delete(rel.Assets, ChecksumsAsset)
	f := serve(t, rel)
	dir := installDir(t, "windows")
	err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
	if err == nil || !strings.Contains(err.Error(), "can't be verified") {
		t.Fatalf("err = %v, want a missing-checksums refusal", err)
	}
	assertUnchanged(t, dir, "windows")
}

func TestChecksumsWithoutEntryRefuses(t *testing.T) {
	rel := fakeRelease(t, "v1.6.0")
	_, linux := testserver.ArchiveNames("v1.6.0")
	rel.Assets[ChecksumsAsset] = testserver.Checksums(rel.Assets, linux)
	f := serve(t, rel)
	dir := installDir(t, "windows")
	err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
	if err == nil || !strings.Contains(err.Error(), "no entry") {
		t.Fatalf("err = %v, want a no-entry refusal", err)
	}
	assertUnchanged(t, dir, "windows")
}

func releaseWith(t *testing.T, goos string, extra testserver.Entry) *testserver.Release {
	t.Helper()
	win, linux := testserver.ArchiveNames("v1.6.0")
	var entries []testserver.Entry
	for _, b := range Binaries {
		entries = append(entries, testserver.Entry{Name: binaryName(b, goos), Body: []byte("new")})
	}
	entries = append(entries, extra)
	rel := &testserver.Release{Tag: "v1.6.0", Assets: map[string][]byte{}}
	name := win
	pack := testserver.Zip
	if goos == "linux" {
		name, pack = linux, testserver.TarGz
	}
	data, err := pack(entries)
	if err != nil {
		t.Fatal(err)
	}
	rel.Assets[name] = data
	rel.Assets[ChecksumsAsset] = testserver.Checksums(rel.Assets, name)
	return rel
}

func TestUnsafeArchiveEntriesRefused(t *testing.T) {
	cases := []struct {
		name  string
		goos  string
		entry testserver.Entry
	}{
		{"zip dotdot", "windows", testserver.Entry{Name: "../evil.exe", Body: []byte("x")}},
		{"zip nested dotdot", "windows", testserver.Entry{Name: `sub\..\..\evil.exe`, Body: []byte("x")}},
		{"zip absolute", "windows", testserver.Entry{Name: "/etc/evil", Body: []byte("x")}},
		{"zip drive", "windows", testserver.Entry{Name: "C:/evil.exe", Body: []byte("x")}},
		{"zip symlink", "windows", testserver.Entry{Name: "link", Link: "/etc/passwd"}},
		{"tar dotdot", "linux", testserver.Entry{Name: "./../evil", Body: []byte("x")}},
		{"tar absolute", "linux", testserver.Entry{Name: "/tmp/evil", Body: []byte("x")}},
		{"tar symlink", "linux", testserver.Entry{Name: "./link", Link: "/etc/passwd"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := serve(t, releaseWith(t, c.goos, c.entry))
			dir := installDir(t, c.goos)
			if err := checkAndInstall(t, f.client("v1.5.1", c.goos), dir); err == nil {
				t.Fatal("unsafe archive was installed")
			}
			assertUnchanged(t, dir, c.goos)
		})
	}
}

func TestArchiveMissingBinaryRefused(t *testing.T) {
	win, _ := testserver.ArchiveNames("v1.6.0")
	data, _ := testserver.Zip([]testserver.Entry{{Name: "script-manager.exe", Body: []byte("new")}})
	rel := &testserver.Release{Tag: "v1.6.0", Assets: map[string][]byte{win: data}}
	rel.Assets[ChecksumsAsset] = testserver.Checksums(rel.Assets, win)
	f := serve(t, rel)
	dir := installDir(t, "windows")
	if err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir); err == nil {
		t.Fatal("partial archive was installed")
	}
	assertUnchanged(t, dir, "windows")
}

func TestRollbackWhenThirdSwapFails(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := installDir(t, "windows")
	third := filepath.Join(dir, binaryName(Binaries[2], "windows"))

	rename = func(from, to string) error {
		if to == third && strings.Contains(from, ".new-") {
			return errors.New("locked")
		}
		return os.Rename(from, to)
	}
	defer func() { rename = os.Rename }()

	err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
	if err == nil || !strings.Contains(err.Error(), "nothing was changed") {
		t.Fatalf("err = %v, want a rollback error", err)
	}
	assertUnchanged(t, dir, "windows")
}

func TestUnwritableDirRefused(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := filepath.Join(t.TempDir(), "missing")
	err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
	if err == nil || !strings.Contains(err.Error(), "no write access") {
		t.Fatalf("err = %v, want a write-access error", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.agents) > 1 {
		t.Fatal("downloaded despite an unwritable dir")
	}
}

func TestCleanupOldIgnoresMissing(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "notes.old")
	os.WriteFile(keep, nil, 0o644)
	os.WriteFile(filepath.Join(dir, "sm-config-edit.exe.old"), nil, 0o644)
	CleanupOld(dir, "windows")
	CleanupOld("", "windows")
	if _, err := os.Stat(filepath.Join(dir, "sm-config-edit.exe.old")); err == nil {
		t.Error("sm-config-edit.exe.old survived")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("an unrelated .old file was removed")
	}
}
