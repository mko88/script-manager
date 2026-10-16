// Package testserver serves a fake GitHub release for the updater: the
// releases/latest JSON, the platform archives and checksums.txt. The update
// tests use it, and so does its cmd, for trying an update by hand against
// SCRIPT_MANAGER_UPDATE_URL.
package testserver

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is one archive member. A non-empty Link makes it a symlink.
type Entry struct {
	Name string
	Body []byte
	Link string
}

// Release is what the server publishes as the latest release.
type Release struct {
	Tag        string
	Prerelease bool
	Assets     map[string][]byte
}

// Handler serves rel at /releases/latest, with each asset under
// /download/<name>. Every request's User-Agent is passed to onRequest, if
// set.
func Handler(rel *Release, onRequest func(*http.Request)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		if onRequest != nil {
			onRequest(r)
		}
		type asset struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		}
		names := make([]string, 0, len(rel.Assets))
		for n := range rel.Assets {
			names = append(names, n)
		}
		sort.Strings(names)
		assets := []asset{}
		for _, n := range names {
			assets = append(assets, asset{n, "http://" + r.Host + "/download/" + n})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name":   rel.Tag,
			"html_url":   "http://" + r.Host + "/releases/tag/" + rel.Tag,
			"draft":      false,
			"prerelease": rel.Prerelease,
			"assets":     assets,
		})
	})
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		if onRequest != nil {
			onRequest(r)
		}
		body, ok := rel.Assets[strings.TrimPrefix(r.URL.Path, "/download/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	})
	return mux
}

// Zip builds a zip archive; regular entries get mode 0755.
func Zip(entries []Entry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.Name, Method: zip.Deflate}
		body := e.Body
		if e.Link != "" {
			h.SetMode(os.ModeSymlink | 0o777)
			body = []byte(e.Link)
		} else {
			h.SetMode(0o755)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TarGz builds a gzipped tar archive; regular entries get mode 0755.
func TarGz(entries []Entry) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		h := &tar.Header{Name: e.Name, Mode: 0o755, Size: int64(len(e.Body)), Typeflag: tar.TypeReg}
		if e.Link != "" {
			h.Typeflag, h.Linkname, h.Size = tar.TypeSymlink, e.Link, 0
		}
		if err := tw.WriteHeader(h); err != nil {
			return nil, err
		}
		if e.Link == "" {
			if _, err := tw.Write(e.Body); err != nil {
				return nil, err
			}
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Checksums renders sha256sum output for the named assets.
func Checksums(assets map[string][]byte, names ...string) []byte {
	var b strings.Builder
	for _, n := range names {
		sum := sha256.Sum256(assets[n])
		fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(sum[:]), n)
	}
	return []byte(b.String())
}

// ArchiveNames are the release asset names for tag.
func ArchiveNames(tag string) (windows, linux string) {
	return "script-manager-windows-amd64-" + tag + ".zip", "script-manager-linux-amd64-" + tag + ".tar.gz"
}

var binaries = []string{"script-manager", "script-manager-gui", "sm-config-edit"}

// FromDir builds a release from the binaries in dir: a Windows zip when all
// three .exe files are there, a Linux tar.gz when all three extensionless
// ones are, and checksums.txt covering both.
func FromDir(dir, tag string) (*Release, error) {
	winName, linuxName := ArchiveNames(tag)
	rel := &Release{Tag: tag, Assets: map[string][]byte{}}
	var sums []string
	for _, p := range []struct {
		ext, name string
		pack      func([]Entry) ([]byte, error)
	}{{".exe", winName, Zip}, {"", linuxName, TarGz}} {
		var entries []Entry
		for _, b := range binaries {
			body, err := os.ReadFile(filepath.Join(dir, b+p.ext))
			if err != nil {
				entries = nil
				break
			}
			entries = append(entries, Entry{Name: b + p.ext, Body: body})
		}
		if entries == nil {
			continue
		}
		data, err := p.pack(entries)
		if err != nil {
			return nil, err
		}
		rel.Assets[p.name] = data
		sums = append(sums, p.name)
	}
	if len(sums) == 0 {
		return nil, fmt.Errorf("%s has neither all three .exe files nor all three Linux binaries", dir)
	}
	rel.Assets["checksums.txt"] = Checksums(rel.Assets, sums...)
	return rel, nil
}
