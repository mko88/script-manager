package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxArchiveBytes   = 512 << 20
	maxChecksumsBytes = 1 << 20
)

// maxBinaryBytes caps each extracted binary; a variable so tests can lower it.
var maxBinaryBytes int64 = 256 << 20

// Binaries are the executables a release ships; all three are replaced
// together.
var Binaries = []string{"script-manager", "script-manager-gui", "sm-config-edit"}

// rename and remove are swapped out in tests to inject failures.
var (
	rename = os.Rename
	remove = os.Remove
)

func binaryName(base, goos string) string {
	if goos == "windows" {
		return base + ".exe"
	}
	return base
}

func (c *Client) knownBinary(name string) bool {
	for _, b := range Binaries {
		if name == binaryName(b, c.GOOS) {
			return true
		}
	}
	return false
}

// Install downloads res's archive, verifies it against the release's
// checksums.txt, and replaces the three binaries in the directory of exe,
// the running executable. On any failure the directory is left as it was,
// or the error names what could not be restored.
func (c *Client) Install(ctx context.Context, res Result, exe string) error {
	if !res.Available {
		return errors.New("no update available")
	}
	if !res.Installable || res.checksumsURL == "" {
		return fmt.Errorf("release %s has no %s, so its download can't be verified; install it manually from %s",
			res.Latest, ChecksumsAsset, res.ReleaseURL)
	}
	if exe == "" {
		return errors.New("can't locate the running executable")
	}
	if name := filepath.Base(exe); !c.knownBinary(name) {
		return fmt.Errorf("can't update: this program is named %s, but only %s can be updated in place; rename it back or update by hand",
			name, strings.Join(Binaries, ", "))
	}
	dir := filepath.Dir(exe)
	if err := checkWritable(dir); err != nil {
		return err
	}

	want, err := c.expectedSum(ctx, res)
	if err != nil {
		return err
	}

	archive, err := c.download(ctx, res.archiveURL, want)
	if err != nil {
		return err
	}
	defer os.Remove(archive)

	staged, err := c.stage(archive, res.archiveName, dir)
	defer func() {
		for _, p := range staged {
			os.Remove(p)
		}
	}()
	if err != nil {
		return fmt.Errorf("unpacking %s: %w", res.archiveName, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.BeforeSwap != nil {
		c.BeforeSwap()
	}
	return c.swap(staged, dir)
}

func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".sm-update-probe-*")
	if err != nil {
		return fmt.Errorf("can't update: no write access to %s", dir)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}

func (c *Client) expectedSum(ctx context.Context, res Result) (string, error) {
	resp, err := c.get(ctx, res.checksumsURL, "")
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", ChecksumsAsset, err)
	}
	defer resp.Body.Close()
	sum, err := findChecksum(io.LimitReader(resp.Body, maxChecksumsBytes), res.archiveName)
	if err != nil {
		return "", fmt.Errorf("release %s: %w", res.Latest, err)
	}
	return sum, nil
}

// findChecksum reads sha256sum output ("<hex>  <name>", or "<hex> *<name>")
// and returns the digest for name, which must appear exactly once.
func findChecksum(r io.Reader, name string) (string, error) {
	sc := bufio.NewScanner(r)
	found := ""
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("%s lists %s more than once", ChecksumsAsset, name)
		}
		sum := strings.ToLower(fields[0])
		if b, err := hex.DecodeString(sum); err != nil || len(b) != sha256.Size {
			return "", fmt.Errorf("%s has a malformed entry for %s", ChecksumsAsset, name)
		}
		found = sum
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("reading %s: %w", ChecksumsAsset, err)
	}
	if found == "" {
		return "", fmt.Errorf("%s has no entry for %s, so it can't be verified", ChecksumsAsset, name)
	}
	return found, nil
}

// download saves url to a temp file and checks its SHA-256 against want.
func (c *Client) download(ctx context.Context, url, want string) (string, error) {
	resp, err := c.get(ctx, url, "")
	if err != nil {
		return "", fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp("", "sm-update-*.download")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxArchiveBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	switch {
	case err != nil:
		err = fmt.Errorf("downloading update: %w", err)
	case n > maxArchiveBytes:
		err = errors.New("downloading update: archive is larger than allowed")
	case hex.EncodeToString(h.Sum(nil)) != want:
		err = errors.New("downloaded update failed its SHA-256 check; nothing was installed")
	}
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// stage extracts the three binaries from archive into temp files in dir,
// keyed by base name. Whatever it returns must be removed by the caller,
// even on error.
func (c *Client) stage(archive, archiveName, dir string) (map[string]string, error) {
	wanted := map[string]string{}
	for _, b := range Binaries {
		wanted[binaryName(b, c.GOOS)] = b
	}
	staged := map[string]string{}
	add := func(entry string, r io.Reader) error {
		base, ok := wanted[entry]
		if !ok {
			return nil
		}
		if _, dup := staged[base]; dup {
			return fmt.Errorf("archive contains %s twice", entry)
		}
		p, err := writeStaged(dir, entry, r)
		if err != nil {
			return err
		}
		staged[base] = p
		return nil
	}

	var err error
	switch {
	case strings.HasSuffix(archiveName, ".zip"):
		err = walkZip(archive, add)
	case strings.HasSuffix(archiveName, ".tar.gz"):
		err = walkTarGz(archive, add)
	default:
		err = fmt.Errorf("unsupported archive format")
	}
	if err != nil {
		return staged, err
	}
	for _, b := range Binaries {
		if _, ok := staged[b]; !ok {
			return staged, fmt.Errorf("archive has no %s", binaryName(b, c.GOOS))
		}
	}
	return staged, nil
}

func writeStaged(dir, entry string, r io.Reader) (string, error) {
	f, err := os.CreateTemp(dir, "."+entry+".new-*")
	if err != nil {
		return "", err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxBinaryBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxBinaryBytes {
		err = fmt.Errorf("%s is larger than allowed", entry)
	}
	if err == nil {
		err = os.Chmod(f.Name(), 0o755)
	}
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// entryName validates an archive entry's path and returns it cleaned.
// Traversal and absolute paths fail the whole archive.
func entryName(name string) (string, error) {
	n := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(n, "/") || filepath.IsAbs(name) || (len(n) > 1 && n[1] == ':') {
		return "", fmt.Errorf("archive entry %q has an absolute path", name)
	}
	for _, part := range strings.Split(n, "/") {
		if part == ".." {
			return "", fmt.Errorf("archive entry %q escapes the archive", name)
		}
	}
	return path.Clean(n), nil
}

func walkZip(archive string, add func(string, io.Reader) error) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		name, err := entryName(f.Name)
		if err != nil {
			return err
		}
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %q is a symlink", f.Name)
		}
		if !mode.IsRegular() {
			continue
		}
		if f.UncompressedSize64 > uint64(maxBinaryBytes) {
			return fmt.Errorf("archive entry %q is larger than allowed", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = add(name, rc)
		rc.Close()
		if errors.Is(err, zip.ErrFormat) {
			// The reader reports more data than the header declared this way.
			return fmt.Errorf("archive entry %q is larger than allowed by its header", f.Name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func walkTarGz(archive string, add func(string, io.Reader) error) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name, err := entryName(hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("archive entry %q is a link", hdr.Name)
		case tar.TypeReg:
		default:
			continue
		}
		if hdr.Size > maxBinaryBytes {
			return fmt.Errorf("archive entry %q is larger than allowed", hdr.Name)
		}
		if err := add(name, tr); err != nil {
			return err
		}
	}
}

// swap moves each staged binary into place, keeping the current one as
// <name>.old. Any failure restores every file already moved; files that
// can't be restored are named in the error.
func (c *Client) swap(staged map[string]string, dir string) error {
	type moved struct {
		target, old string // old is "" when there was no previous file
		replaced    bool   // the new file is in place at target
	}
	var done []moved

	restore := func() []string {
		var failed []string
		for i := len(done) - 1; i >= 0; i-- {
			m := done[i]
			if m.replaced {
				if err := remove(m.target); err != nil && !os.IsNotExist(err) && m.old == "" {
					failed = append(failed, fmt.Sprintf("could not remove the new %s; delete it by hand", filepath.Base(m.target)))
					continue
				}
			}
			if m.old != "" {
				if err := rename(m.old, m.target); err != nil {
					failed = append(failed, fmt.Sprintf("could not restore %s; rename %s back to %s by hand",
						filepath.Base(m.target), filepath.Base(m.old), filepath.Base(m.target)))
				}
			}
		}
		return failed
	}
	fail := func(target string, err error) error {
		msg := fmt.Sprintf("replacing %s: %v", filepath.Base(target), err)
		if failed := restore(); len(failed) > 0 {
			return fmt.Errorf("%s; %s", msg, strings.Join(failed, "; "))
		}
		return fmt.Errorf("%s; nothing was changed", msg)
	}

	for _, b := range Binaries {
		target := filepath.Join(dir, binaryName(b, c.GOOS))
		m := moved{target: target}
		if _, err := os.Stat(target); err == nil {
			old, err := oldName(target)
			if err != nil {
				return fail(target, err)
			}
			if err := rename(target, old); err != nil {
				return fail(target, err)
			}
			m.old = old
		}
		done = append(done, m)
		if err := rename(staged[b], target); err != nil {
			return fail(target, err)
		}
		done[len(done)-1].replaced = true
	}
	return nil
}

// oldName is where target's current file is kept: <name>.old, or a unique
// <name>.old-<random> when a previous .old can't be removed (on Windows, an
// old copy that is still running).
func oldName(target string) (string, error) {
	old := target + ".old"
	if err := remove(old); err == nil || os.IsNotExist(err) {
		return old, nil
	}
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return old + "-" + hex.EncodeToString(b), nil
}

// staleAfter is how old a staging or probe file must be before CleanupOld
// treats it as abandoned rather than belonging to an update in progress.
const staleAfter = 10 * time.Minute

// CleanupOld removes what earlier updates left in dir: <name>.old and
// <name>.old-<8 hex> copies, and staging and probe files older than
// staleAfter. A copy that is still running (Windows locks it) stays until a
// later start.
func CleanupOld(dir, goos string) {
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if name := e.Name(); leftover(name, goos) || (stagingFile(name, goos) && stale(e)) {
			os.Remove(filepath.Join(dir, name))
		}
	}
}

func leftover(name, goos string) bool {
	for _, b := range Binaries {
		old := binaryName(b, goos) + ".old"
		if name == old {
			return true
		}
		if suffix, ok := strings.CutPrefix(name, old+"-"); ok && isHex8(suffix) {
			return true
		}
	}
	return false
}

func stagingFile(name, goos string) bool {
	if strings.HasPrefix(name, ".sm-update-probe-") {
		return true
	}
	for _, b := range Binaries {
		if strings.HasPrefix(name, "."+binaryName(b, goos)+".new-") {
			return true
		}
	}
	return false
}

func stale(e os.DirEntry) bool {
	info, err := e.Info()
	return err == nil && time.Since(info.ModTime()) > staleAfter
}

func isHex8(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}
