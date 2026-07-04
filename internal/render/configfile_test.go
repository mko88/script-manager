package render

import "testing"

func TestExpandConfigFileNoPlaceholder(t *testing.T) {
	src := "plain markdown, nothing to expand"
	if got := ExpandConfigFile(src, "/etc/config.yaml"); got != src {
		t.Errorf("ExpandConfigFile with no placeholder = %q, want unchanged %q", got, src)
	}
}

func TestExpandConfigFileSubstitutes(t *testing.T) {
	got := ExpandConfigFile("Loaded from "+ConfigFilePlaceholder+".", "/opt/app/config-win.yaml")
	want := "Loaded from /opt/app/config-win.yaml."
	if got != want {
		t.Errorf("ExpandConfigFile = %q, want %q", got, want)
	}
}

func TestExpandConfigFileEmptyPath(t *testing.T) {
	got := ExpandConfigFile(ConfigFilePlaceholder, "")
	if got != noConfigFile {
		t.Errorf("ExpandConfigFile with empty path = %q, want %q", got, noConfigFile)
	}
}
