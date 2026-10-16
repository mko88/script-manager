package update

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"script-manager/internal/update/testserver"
)

// failStagedRename makes the rename of a staged file onto the index'th
// binary fail.
func failStagedRename(t *testing.T, dir string, index int) {
	t.Helper()
	target := filepath.Join(dir, binaryName(Binaries[index], "windows"))
	rename = func(from, to string) error {
		if to == target && strings.Contains(from, ".new-") {
			return errors.New("locked")
		}
		return os.Rename(from, to)
	}
	t.Cleanup(func() { rename = os.Rename })
}

func TestRollbackWhenASwapFails(t *testing.T) {
	for _, index := range []int{1, 2} {
		t.Run(Binaries[index], func(t *testing.T) {
			f := serve(t, fakeRelease(t, "v1.6.0"))
			dir := installDir(t, "windows")
			failStagedRename(t, dir, index)

			err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
			if err == nil || !strings.Contains(err.Error(), "nothing was changed") {
				t.Fatalf("err = %v, want a rollback error", err)
			}
			assertUnchanged(t, dir, "windows")
		})
	}
}

func TestRollbackFailureIsReported(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := installDir(t, "windows")
	first := filepath.Join(dir, binaryName(Binaries[0], "windows"))
	third := filepath.Join(dir, binaryName(Binaries[2], "windows"))
	rename = func(from, to string) error {
		switch {
		case to == third && strings.Contains(from, ".new-"):
			return errors.New("locked")
		case to == first && strings.HasSuffix(from, ".old"):
			return errors.New("denied")
		}
		return os.Rename(from, to)
	}
	defer func() { rename = os.Rename }()

	err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
	if err == nil {
		t.Fatal("no error")
	}
	msg := err.Error()
	if strings.Contains(msg, "nothing was changed") {
		t.Errorf("claims nothing changed after a failed restore: %s", msg)
	}
	if !strings.Contains(msg, "could not restore script-manager.exe; rename script-manager.exe.old back") {
		t.Errorf("restore failure not named: %s", msg)
	}
	if _, err := os.Stat(first + ".old"); err != nil {
		t.Error("the .old copy the error points to is gone")
	}
}

func TestExistingOldIsReplaced(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := installDir(t, "windows")
	stale := filepath.Join(dir, "script-manager.exe.old")
	os.WriteFile(stale, []byte("stale"), 0o644)
	if err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(stale); string(data) != "script-manager.exe current" {
		t.Errorf(".old = %q, want the replaced binary", data)
	}
}

func TestLockedOldGetsUniqueName(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := installDir(t, "windows")
	locked := filepath.Join(dir, "script-manager.exe.old")
	os.WriteFile(locked, []byte("running"), 0o644)
	remove = func(p string) error {
		if p == locked {
			return errors.New("in use")
		}
		return os.Remove(p)
	}
	defer func() { remove = os.Remove }()

	if err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if data, _ := os.ReadFile(locked); string(data) != "running" {
		t.Error("the locked .old was touched")
	}
	matches, _ := filepath.Glob(locked + "-*")
	if len(matches) != 1 {
		t.Fatalf("unique .old copies: %v", matches)
	}
	if data, _ := os.ReadFile(matches[0]); string(data) != "script-manager.exe current" {
		t.Errorf("%s = %q", matches[0], data)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "script-manager.exe")); string(data) != "script-manager.exe v1.6.0" {
		t.Errorf("new binary = %q", data)
	}
}

func TestDuplicateChecksumRefused(t *testing.T) {
	rel := fakeRelease(t, "v1.6.0")
	win, _ := testserver.ArchiveNames("v1.6.0")
	sums := testserver.Checksums(rel.Assets, win)
	rel.Assets[ChecksumsAsset] = append(append([]byte{}, sums...), sums...)
	f := serve(t, rel)
	dir := installDir(t, "windows")
	err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
	if err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("err = %v, want a duplicate-entry refusal", err)
	}
	assertUnchanged(t, dir, "windows")
}

func TestMissingChecksumsNotInstallable(t *testing.T) {
	rel := fakeRelease(t, "v1.6.0")
	delete(rel.Assets, ChecksumsAsset)
	f := serve(t, rel)
	res, err := f.client("v1.5.1", "windows").Check(context.Background())
	if err != nil || !res.Available || res.Installable {
		t.Fatalf("res = %+v, err = %v; want available but not installable", res, err)
	}
	if res.ReleaseURL == "" {
		t.Error("no release page to point to")
	}
}

func TestSizeCap(t *testing.T) {
	old := maxBinaryBytes
	maxBinaryBytes = 16
	defer func() { maxBinaryBytes = old }()
	big := []byte(strings.Repeat("x", 64))

	win, _ := testserver.ArchiveNames("v1.6.0")
	honest, err := testserver.Zip([]testserver.Entry{
		{Name: "script-manager.exe", Body: big}, {Name: "script-manager-gui.exe"}, {Name: "sm-config-edit.exe"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateRaw(&zip.FileHeader{
		Name: "script-manager.exe", Method: zip.Store, CRC32: crc32.ChecksumIEEE(big),
		CompressedSize64: uint64(len(big)), UncompressedSize64: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.Write(big)
	for _, n := range []string{"script-manager-gui.exe", "sm-config-edit.exe"} {
		zw.Create(n)
	}
	zw.Close()

	for name, data := range map[string][]byte{"honest header": honest, "understated header": buf.Bytes()} {
		t.Run(name, func(t *testing.T) {
			rel := &testserver.Release{Tag: "v1.6.0", Assets: map[string][]byte{win: data}}
			rel.Assets[ChecksumsAsset] = testserver.Checksums(rel.Assets, win)
			f := serve(t, rel)
			dir := installDir(t, "windows")
			err := checkAndInstall(t, f.client("v1.5.1", "windows"), dir)
			if err == nil || !strings.Contains(err.Error(), "larger than allowed") {
				t.Fatalf("err = %v, want a size refusal", err)
			}
			assertUnchanged(t, dir, "windows")
		})
	}
}

func TestUnknownExecutableNameRefused(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := installDir(t, "windows")
	c := f.client("v1.5.1", "windows")
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = c.Install(context.Background(), res, filepath.Join(dir, "my-tool.exe"))
	if err == nil || !strings.Contains(err.Error(), "my-tool.exe") {
		t.Fatalf("err = %v, want a refusal naming the program", err)
	}
	assertUnchanged(t, dir, "windows")
}

func TestCancelledInstallChangesNothing(t *testing.T) {
	f := serve(t, fakeRelease(t, "v1.6.0"))
	dir := installDir(t, "windows")
	c := f.client("v1.5.1", "windows")
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Install(ctx, res, filepath.Join(dir, "script-manager.exe")); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	assertUnchanged(t, dir, "windows")
}

// strictClient is a default-source client pointed at a test server.
func strictClient(t *testing.T, url string) *Client {
	t.Helper()
	t.Setenv(EnvBaseURL, "")
	c := NewClient("v1.5.1", "windows", "amd64")
	if !c.strict {
		t.Fatal("default client is not strict")
	}
	c.BaseURL = url
	return c
}

func TestStrictAssetURLs(t *testing.T) {
	win, _ := testserver.ArchiveNames("v1.6.0")
	cases := map[string]struct {
		archiveURL string
		ok         bool
	}{
		"github":     {downloadPrefix + "v1.6.0/" + win, true},
		"other host": {"https://evil.example/" + win, false},
		"other tag":  {downloadPrefix + "v1.5.0/" + win, false},
		"plain http": {"http://github.com/mko88/script-manager/releases/download/v1.6.0/" + win, false},
		"other repo": {"https://github.com/someone/script-manager/releases/download/v1.6.0/" + win, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"tag_name":"v1.6.0","assets":[{"name":%q,"browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}`,
					win, tc.archiveURL, downloadPrefix+"v1.6.0/checksums.txt")
			}))
			defer srv.Close()
			res, err := strictClient(t, srv.URL).Check(context.Background())
			if tc.ok && (err != nil || !res.Installable) {
				t.Fatalf("res = %+v, err = %v", res, err)
			}
			if !tc.ok && err == nil {
				t.Fatal("unexpected asset URL accepted")
			}
		})
	}
}

func TestStrictRedirects(t *testing.T) {
	mk := func(u string) *http.Request {
		r, _ := http.NewRequest(http.MethodGet, u, nil)
		return r
	}
	if err := httpsOnlyRedirect(mk("https://objects.githubusercontent.com/x"), nil); err != nil {
		t.Errorf("https redirect refused: %v", err)
	}
	if err := httpsOnlyRedirect(mk("http://objects.githubusercontent.com/x"), nil); err == nil {
		t.Error("http redirect accepted")
	}
	if c := strictClient(t, "https://example.invalid"); c.HTTP.CheckRedirect == nil {
		t.Error("strict client has no redirect check")
	}
	t.Setenv(EnvBaseURL, "http://127.0.0.1:1")
	if c := NewClient("v1.5.1", "windows", "amd64"); c.strict || c.HTTP.CheckRedirect != nil {
		t.Error("override client is strict")
	}
}

func TestOverrideValidation(t *testing.T) {
	cases := map[string]bool{
		"https://example.com/x":  true,
		"http://127.0.0.1:8765":  true,
		"http://localhost:8765/": true,
		"http://[::1]:8765":      true,
		"http://example.com":     false,
		"http://127.0.0.2":       false,
		"ftp://127.0.0.1":        false,
		"localhost:8765":         false,
	}
	for base, ok := range cases {
		t.Setenv(EnvBaseURL, base)
		c := NewClient("v1.5.1", "windows", "amd64")
		if c.Override() == "" {
			t.Errorf("%s: override not reported", base)
		}
		err := c.overrideErr
		if !ok && (err == nil || !strings.Contains(err.Error(), EnvBaseURL)) {
			t.Errorf("%s: err = %v, want a refusal", base, err)
		}
		if ok && err != nil {
			t.Errorf("%s refused: %v", base, err)
		}
		if !ok {
			if _, err := c.Check(context.Background()); err == nil || !strings.Contains(err.Error(), EnvBaseURL) {
				t.Errorf("%s: Check err = %v", base, err)
			}
		}
	}
}
