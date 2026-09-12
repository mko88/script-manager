// Package version reports which build of script-manager is running, for
// script-manager-gui's About panel.
package version

// Set at link time by build.sh (see its ldflags). The defaults are what a
// plain `go build` or `wails build` produces, so a binary built without
// them says "dev" rather than claiming to be a release.
var (
	// Version is `git describe --tags --always --dirty`: "v1.4.0.0" on a
	// clean release tag, "v1.4.0.0-4-g94b2835" a few commits past one, and
	// a "-dirty" suffix with uncommitted changes. Releases are tagged
	// vMajor.Minor.Patch.Build — see CLAUDE.md for which segment to bump.
	Version = "dev"
	// Commit is the short hash. Version usually carries it too, but not
	// when the build sits exactly on a tag — and a bug report wants it
	// either way.
	Commit = ""
	// Date is when the binary was linked, RFC 3339 in UTC.
	Date = ""
)

// Info is what the About panel renders.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
}

func Get() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}
