package update

import (
	"regexp"
	"strconv"
)

// semver is a release version. legacy marks a pre-v1.4.0 four-segment tag,
// which orders below every three-segment version.
type semver struct {
	legacy bool
	parts  [3]int
}

var (
	releaseTag  = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)
	legacyTag   = regexp.MustCompile(`^v?\d+\.\d+\.\d+\.\d+$`)
	buildSuffix = regexp.MustCompile(`(-\d+-g[0-9a-f]+)?(-dirty)?$`)
)

// parseTag parses a release tag: vMajor.Minor.Patch, or a legacy
// four-segment tag.
func parseTag(s string) (semver, bool) {
	if legacyTag.MatchString(s) {
		return semver{legacy: true}, true
	}
	m := releaseTag.FindStringSubmatch(s)
	if m == nil {
		return semver{}, false
	}
	var v semver
	for i := range v.parts {
		v.parts[i], _ = strconv.Atoi(m[i+1])
	}
	return v, true
}

// parseBuild parses this build's version.Version, a `git describe` string:
// the tag itself, or the tag plus -N-gHASH and/or -dirty. Anything else
// (dev, a bare commit hash) is not a comparable build.
func parseBuild(s string) (semver, bool) {
	return parseTag(buildSuffix.ReplaceAllString(s, ""))
}

// Checkable reports whether a build with this version string can check for
// updates at all.
func Checkable(buildVersion string) bool {
	_, ok := parseBuild(buildVersion)
	return ok
}

func (v semver) newerThan(o semver) bool {
	if v.legacy || o.legacy {
		return !v.legacy && o.legacy
	}
	for i := range v.parts {
		if v.parts[i] != o.parts[i] {
			return v.parts[i] > o.parts[i]
		}
	}
	return false
}
