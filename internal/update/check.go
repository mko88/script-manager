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
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the GitHub API root for the repository; releases are
	// read from <base>/releases/latest.
	DefaultBaseURL = "https://api.github.com/repos/mko88/script-manager"
	// EnvBaseURL overrides DefaultBaseURL, e.g. to point at a local test
	// server. It must be https://, or http:// on the loopback host.
	EnvBaseURL = "SCRIPT_MANAGER_UPDATE_URL"

	ChecksumsAsset = "checksums.txt"

	// downloadPrefix is the only place assets may come from when the default
	// source is in use; GitHub redirects from there to its CDN.
	downloadPrefix    = "https://github.com/mko88/script-manager/releases/download/"
	releasePagePrefix = "https://github.com/mko88/script-manager/releases/tag/"

	userAgent    = "script-manager-updater"
	CheckTimeout = 10 * time.Second
	maxJSONBytes = 1 << 20
)

// idleTimeout aborts a request whose response headers, or next body bytes,
// take longer than this.
var idleTimeout = 30 * time.Second

// ErrNotCheckable is returned for builds without a release version (dev
// builds, untagged commits), which never check for updates.
var ErrNotCheckable = errors.New("this build has no release version, so it can't check for updates")

// Result describes the latest release relative to the running build.
type Result struct {
	Current    string
	Latest     string // the release tag, e.g. v1.6.0
	Available  bool
	ReleaseURL string
	// Installable is false for a newer release that has no checksums.txt;
	// it can only be installed by hand from ReleaseURL.
	Installable bool

	archiveName  string
	archiveURL   string
	checksumsURL string
}

// Client checks for and installs releases. Build one with NewClient.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	GOOS    string
	GOARCH  string
	Current string
	// BeforeSwap, if set, runs once the download is verified and unpacked,
	// just before files are replaced; cancelling has no effect after it.
	BeforeSwap func()

	// strict pins asset URLs to the project's GitHub download path and
	// refuses non-https redirects; it is on unless EnvBaseURL is set.
	strict      bool
	override    string
	overrideErr error
}

// NewClient returns a client for the running build, honouring EnvBaseURL.
func NewClient(current, goos, goarch string) *Client {
	c := &Client{
		BaseURL: DefaultBaseURL,
		GOOS:    goos,
		GOARCH:  goarch,
		Current: current,
		strict:  true,
	}
	if base := strings.TrimSpace(os.Getenv(EnvBaseURL)); base != "" {
		c.strict = false
		c.override = strings.TrimRight(base, "/")
		c.BaseURL = c.override
		c.overrideErr = validOverride(c.override)
	}
	c.HTTP = newHTTPClient(c.strict)
	return c
}

// Override is the EnvBaseURL value in effect, or "".
func (c *Client) Override() string { return c.override }

func validOverride(base string) error {
	u, err := url.Parse(base)
	if err == nil {
		switch {
		case u.Scheme == "https" && u.Host != "":
			return nil
		case u.Scheme == "http":
			switch u.Hostname() {
			case "127.0.0.1", "::1", "localhost":
				return nil
			}
		}
	}
	return fmt.Errorf("%s must be an https:// URL, or http:// on 127.0.0.1, ::1 or localhost", EnvBaseURL)
}

func newHTTPClient(strict bool) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	tr.TLSHandshakeTimeout = 15 * time.Second
	tr.ResponseHeaderTimeout = idleTimeout
	c := &http.Client{Transport: tr}
	if strict {
		c.CheckRedirect = httpsOnlyRedirect
	}
	return c
}

func httpsOnlyRedirect(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return fmt.Errorf("refusing redirect to non-https %s", req.URL.Redacted())
	}
	if len(via) >= 10 {
		return errors.New("too many redirects")
	}
	return nil
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
	if c.overrideErr != nil {
		return Result{}, c.overrideErr
	}
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()

	rel, err := c.fetchLatest(ctx)
	if err != nil {
		return Result{}, err
	}
	res := Result{Current: c.Current, Latest: rel.TagName, ReleaseURL: c.releasePage(rel)}
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
		default:
			continue
		}
		if c.strict && a.URL != downloadPrefix+rel.TagName+"/"+a.Name {
			return Result{}, fmt.Errorf("release %q: %s has an unexpected download URL %q", rel.TagName, a.Name, a.URL)
		}
	}
	if res.archiveURL == "" {
		return Result{}, fmt.Errorf("release %s has no %s download", rel.TagName, name)
	}
	res.Available = true
	res.Installable = res.checksumsURL != ""
	return res, nil
}

// releasePage is the release's web page. The default source builds it from
// the tag; an override's html_url is used only when it is http(s).
func (c *Client) releasePage(rel ghRelease) string {
	if c.strict {
		if _, ok := parseTag(rel.TagName); ok {
			return releasePagePrefix + rel.TagName
		}
		return ""
	}
	u, err := url.Parse(rel.HTMLURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return rel.HTMLURL
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

// get performs a GET and fails on any non-200 status. The body is cancelled
// when it stalls for idleTimeout.
func (c *Client) get(ctx context.Context, url, accept string) (*http.Response, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel(nil)
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		cancel(nil)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel(nil)
		return nil, fmt.Errorf("%q returned %s", url, resp.Status)
	}
	resp.Body = &idleBody{
		rc:     resp.Body,
		ctx:    ctx,
		cancel: cancel,
		timer:  time.AfterFunc(idleTimeout, func() { cancel(ErrStalled) }),
	}
	return resp, nil
}

// ErrStalled is returned when a download stops sending data for longer than
// the idle timeout.
var ErrStalled = errors.New("the download stalled")

type idleBody struct {
	rc     io.ReadCloser
	ctx    context.Context
	cancel context.CancelCauseFunc
	timer  *time.Timer
}

func (b *idleBody) Read(p []byte) (int, error) {
	n, err := b.rc.Read(p)
	if n > 0 {
		b.timer.Reset(idleTimeout)
	}
	if err != nil && err != io.EOF && errors.Is(context.Cause(b.ctx), ErrStalled) {
		err = ErrStalled
	}
	return n, err
}

func (b *idleBody) Close() error {
	b.timer.Stop()
	err := b.rc.Close()
	b.cancel(nil)
	return err
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
