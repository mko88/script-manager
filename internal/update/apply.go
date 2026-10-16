package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxArchiveBytes   = 512 << 20
	maxBinaryBytes    = 256 << 20
	maxChecksumsBytes = 1 << 20
)

// Binaries are the executables a release ships; all three are replaced
// together.
var Binaries = []string{"script-manager", "script-manager-gui", "sm-config-edit"}

// rename is swapped out in tests to inject failures.
var rename = os.Rename

func binaryName(base, goos string) string {
	if goos == "windows" {
		return base + ".exe"
	}
	return base
}

// Install downloads res's archive, verifies it against the release's
// checksums.txt, and replaces the three binaries in dir. On any failure dir
// is left as it was.
func (c *Client) Install(ctx context.Context, res Result, dir string) error {
	if !res.Available {
		return errors.New("no update available")
	}
	if err := checkWritable(dir); err != nil {
		return err
	}
	if res.checksumsURL == "" {
		return fmt.Errorf("release %s has no %s, so its download can't be verified; install it manually from %s",
			res.Latest, ChecksumsAsset, res.ReleaseURL)
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
// and returns the digest for name.
func findChecksum(r io.Reader, name string) (string, error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		sum := strings.ToLower(fields[0])
		if b, err := hex.DecodeString(sum); err != nil || len(b) != sha256.Size {
			return "", fmt.Errorf("%s has a malformed entry for %s", ChecksumsAsset, name)
		}
		return sum, nil
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("reading %s: %w", ChecksumsAsset, err)
	}
	return "", fmt.Errorf("%s has no entry for %s, so it can't be verified", ChecksumsAsset, name)
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
		if f.UncompressedSize64 > maxBinaryBytes {
			return fmt.Errorf("archive entry %q is larger than allowed", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = add(name, rc)
		rc.Close()
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
// <name>.old. Any failure restores every file already swapped.
func (c *Client) swap(staged map[string]string, dir string) error {
	type done struct {
		target, old string
		hadOld      bool
	}
	var swapped []done
	rollback := func() {
		for i := len(swapped) - 1; i >= 0; i-- {
			s := swapped[i]
			if s.hadOld {
				rename(s.old, s.target)
			} else {
				os.Remove(s.target)
			}
		}
	}

	for _, b := range Binaries {
		target := filepath.Join(dir, binaryName(b, c.GOOS))
		old := target + ".old"
		d := done{target: target, old: old}
		if _, err := os.Stat(target); err == nil {
			os.Remove(old)
			if err := rename(target, old); err != nil {
				rollback()
				return fmt.Errorf("replacing %s: %w; nothing was changed", filepath.Base(target), err)
			}
			d.hadOld = true
		}
		if err := rename(staged[b], target); err != nil {
			if d.hadOld {
				rename(old, target)
			}
			rollback()
			return fmt.Errorf("replacing %s: %w; nothing was changed", filepath.Base(target), err)
		}
		swapped = append(swapped, d)
	}
	return nil
}

// CleanupOld removes the .old copies a previous update left in dir. A copy
// that is still running (Windows locks it) stays until a later start.
func CleanupOld(dir, goos string) {
	if dir == "" {
		return
	}
	for _, b := range Binaries {
		os.Remove(filepath.Join(dir, binaryName(b, goos)+".old"))
	}
}
