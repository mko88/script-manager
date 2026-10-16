// Package update checks the project's GitHub releases for a newer build and
// installs it in place: it verifies the download against the release's
// checksums.txt and swaps all three binaries beside the running one.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the GitHub API root for the repository; releases are
	// read from <base>/releases/latest.
	DefaultBaseURL = "https://api.github.com/repos/mko88/script-manager"
	// EnvBaseURL overrides DefaultBaseURL, e.g. to point at a local test
	// server.
	EnvBaseURL = "SCRIPT_MANAGER_UPDATE_URL"

	ChecksumsAsset = "checksums.txt"

	userAgent    = "script-manager-updater"
	CheckTimeout = 10 * time.Second
	maxJSONBytes = 1 << 20
)

// ErrNotCheckable is returned for builds without a release version (dev
// builds, untagged commits), which never check for updates.
var ErrNotCheckable = errors.New("this build has no release version, so it can't check for updates")

// Result describes the latest release relative to the running build.
type Result struct {
	Current    string
	Latest     string // the release tag, e.g. v1.6.0
	Available  bool
	ReleaseURL string

	archiveName  string
	archiveURL   string
	checksumsURL string
}

// Client checks for and installs releases. The zero value is not usable;
// build one with NewClient.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	GOOS    string
	GOARCH  string
	Current string
}

// NewClient returns a client for the running build, honouring EnvBaseURL.
func NewClient(current, goos, goarch string) *Client {
	base := strings.TrimSpace(os.Getenv(EnvBaseURL))
	if base == "" {
		base = DefaultBaseURL
	}
	return &Client{
		BaseURL: strings.TrimRight(base, "/"),
		HTTP:    &http.Client{Timeout: 5 * time.Minute},
		GOOS:    goos,
		GOARCH:  goarch,
		Current: current,
	}
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	HTMLURL    string    `json:"html_url"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

// Check fetches the latest release and reports whether it is strictly newer
// than the running build. It downloads nothing beyond the release metadata.
func (c *Client) Check(ctx context.Context) (Result, error) {
	cur, ok := parseBuild(c.Current)
	if !ok {
		return Result{}, ErrNotCheckable
	}
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()

	rel, err := c.fetchLatest(ctx)
	if err != nil {
		return Result{}, err
	}
	res := Result{Current: c.Current, Latest: rel.TagName, ReleaseURL: rel.HTMLURL}
	if rel.Draft || rel.Prerelease {
		return res, nil
	}
	latest, ok := parseTag(rel.TagName)
	if !ok || !latest.newerThan(cur) {
		return res, nil
	}

	name, err := archiveName(rel.TagName, c.GOOS, c.GOARCH)
	if err != nil {
		return Result{}, err
	}
	for _, a := range rel.Assets {
		switch a.Name {
		case name:
			res.archiveName, res.archiveURL = a.Name, a.URL
		case ChecksumsAsset:
			res.checksumsURL = a.URL
		}
	}
	if res.archiveURL == "" {
		return Result{}, fmt.Errorf("release %s has no %s download", rel.TagName, name)
	}
	res.Available = true
	return res, nil
}

func (c *Client) fetchLatest(ctx context.Context) (ghRelease, error) {
	resp, err := c.get(ctx, c.BaseURL+"/releases/latest", "application/vnd.github+json")
	if err != nil {
		return ghRelease{}, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()

	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSONBytes)).Decode(&rel); err != nil {
		return ghRelease{}, fmt.Errorf("checking for updates: reading release info: %w", err)
	}
	return rel, nil
}

// get performs a GET and fails on any non-200 status.
func (c *Client) get(ctx context.Context, url, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return resp, nil
}

// archiveName is the release asset for a platform, as scripts/release.ps1
// names it.
func archiveName(tag, goos, goarch string) (string, error) {
	if goarch != "amd64" {
		return "", fmt.Errorf("no release build for %s/%s", goos, goarch)
	}
	switch goos {
	case "windows":
		return "script-manager-windows-amd64-" + tag + ".zip", nil
	case "linux":
		return "script-manager-linux-amd64-" + tag + ".tar.gz", nil
	}
	return "", fmt.Errorf("no release build for %s/%s", goos, goarch)
}
