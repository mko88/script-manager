package gui

import (
	"os/exec"
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

func TestLaunchConfigEditorSkipsWhenAlreadyRunning(t *testing.T) {
	// A non-nil configEditorCmd means a previously launched instance hasn't
	// exited yet (see the cmd.Wait() goroutine in LaunchConfigEditor) — it
	// never actually needs to run for this guard to be exercised.
	sentinel := exec.Command("echo")
	a := &App{configEditorCmd: sentinel}

	alreadyRunning, err := a.LaunchConfigEditor()
	if err != nil {
		t.Fatalf("LaunchConfigEditor() error = %v", err)
	}
	if !alreadyRunning {
		t.Error("alreadyRunning = false, want true when configEditorCmd is already set")
	}
	if a.configEditorCmd != sentinel {
		t.Error("configEditorCmd should be left untouched when an instance is already running")
	}
}
