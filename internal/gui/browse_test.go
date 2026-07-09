package gui

import (
	"path/filepath"
	"runtime"
	"testing"

	"script-manager/internal/config"
)

func TestSiblingBinaryName(t *testing.T) {
	want := "sm-config-edit"
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if got := siblingBinaryName("sm-config-edit"); got != want {
		t.Errorf("siblingBinaryName() = %q, want %q", got, want)
	}
}

func TestConfigEditorArgvIncludesConfigPathWhenKnown(t *testing.T) {
	a := &App{cfg: &config.Config{SourcePath: "/some/config.yaml"}, exeDir: "/exe/dir"}
	bin, args := a.configEditorArgv()

	wantBin := filepath.Join("/exe/dir", siblingBinaryName("sm-config-edit"))
	if bin != wantBin {
		t.Errorf("bin = %q, want %q", bin, wantBin)
	}
	if len(args) != 2 || args[0] != "-config" || args[1] != "/some/config.yaml" {
		t.Errorf("args = %v, want [-config /some/config.yaml]", args)
	}
}

func TestConfigEditorArgvOmitsConfigPathWhenUnknown(t *testing.T) {
	a := &App{cfg: &config.Config{}, exeDir: "/exe/dir"}
	_, args := a.configEditorArgv()
	if len(args) != 0 {
		t.Errorf("args = %v, want none", args)
	}
}
