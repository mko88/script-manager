package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"script-manager/internal/config"
)

func inlineTestApp(actions ...config.Action) *App {
	return newTestApp(func() (*config.Config, error) {
		return &config.Config{
			Shell:   []string{"bash", "-c"},
			Items:   []config.Item{{Name: "test"}},
			Actions: actions,
		}, nil
	})
}

func waitForInlineDone(t *testing.T, a *App, itemIndex, actionIndex int) InlineStatusDTO {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		status := a.GetInlineStatus(itemIndex, actionIndex)
		if !status.Running {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("inline action did not finish within the deadline")
	return InlineStatusDTO{}
}

func TestRunActionInline(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hello-inline"})

	if err := a.RunActionInline(0, 0); err != nil {
		t.Fatalf("RunActionInline() error = %v", err)
	}
	status := waitForInlineDone(t, a, 0, 0)
	if status.ExitCode != 0 || !strings.Contains(status.Output, "hello-inline") {
		t.Errorf("final status = %+v, want exit 0 and output containing %q", status, "hello-inline")
	}
}

func TestRunActionInlineCapturesStderrAndNonZeroExit(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Fail", Cmd: "echo oops >&2; exit 3"})

	if err := a.RunActionInline(0, 0); err != nil {
		t.Fatalf("RunActionInline() error = %v", err)
	}
	status := waitForInlineDone(t, a, 0, 0)
	if status.ExitCode != 3 || !strings.Contains(status.Output, "oops") {
		t.Errorf("final status = %+v, want exit 3 and output containing %q", status, "oops")
	}
}

func TestRunActionInlineInvalidItemOrAction(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hi"})

	if err := a.RunActionInline(5, 0); err == nil {
		t.Error("expected an error for an out-of-range item")
	}
	if err := a.RunActionInline(0, 5); err == nil {
		t.Error("expected an error for an out-of-range action")
	}
}

func TestRunActionInlineRejectsInteractiveAction(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Prompt", Cmd: "read -r x", Interactive: true})

	if err := a.RunActionInline(0, 0); err == nil {
		t.Error("expected an error for an interactive action")
	}
	if a.GetInlineStatus(0, 0).Running {
		t.Error("interactive action must not end up marked as running")
	}
}

func TestRunActionInlineClearsStaleEntryOnFailedRestart(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hi"})

	staleOutPath := filepath.Join(t.TempDir(), "stale.log")
	if err := os.WriteFile(staleOutPath, []byte("old output"), 0o600); err != nil {
		t.Fatal(err)
	}
	key := inlineKey{itemIndex: 0, actionIndex: 5}
	a.inlineRuns[key] = &inlineRun{outPath: staleOutPath, exitCode: 3, errMsg: "old error"}

	if err := a.RunActionInline(0, 5); err == nil {
		t.Fatal("expected an error for an out-of-range action")
	}

	status := a.GetInlineStatus(0, 5)
	if status.Running || status.ExitCode != 0 || status.ErrMsg != "" || status.Output != "" {
		t.Errorf("GetInlineStatus after failed restart = %+v, want the zero value (stale entry must be cleared)", status)
	}
}

func waitForInlineRunning(t *testing.T, a *App, itemIndex, actionIndex int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if a.GetInlineStatus(itemIndex, actionIndex).Running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("inline action never started running within the deadline")
}

func TestRunActionInlineRejectsConcurrentRunsOfSameAction(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Sleep", Cmd: "sleep 2"})

	if err := a.RunActionInline(0, 0); err != nil {
		t.Fatalf("first RunActionInline() error = %v", err)
	}
	waitForInlineRunning(t, a, 0, 0)

	if err := a.RunActionInline(0, 0); err == nil {
		t.Error("expected an error running the same action again while it's still running")
	}

	if err := a.CancelInlineAction(0, 0); err != nil {
		t.Fatalf("CancelInlineAction() error = %v", err)
	}
	waitForInlineDone(t, a, 0, 0)
}

func TestRunActionInlineAllowsConcurrentDifferentActions(t *testing.T) {
	a := inlineTestApp(
		config.Action{Title: "Slow", Cmd: "sleep 2; echo slow-done"},
		config.Action{Title: "Fast", Cmd: "echo fast-done"},
	)

	if err := a.RunActionInline(0, 0); err != nil {
		t.Fatalf("RunActionInline(slow) error = %v", err)
	}
	waitForInlineRunning(t, a, 0, 0)

	if err := a.RunActionInline(0, 1); err != nil {
		t.Fatalf("RunActionInline(fast) error = %v, want the second action to start despite the first still running", err)
	}
	fastStatus := waitForInlineDone(t, a, 0, 1)
	if fastStatus.ExitCode != 0 || !strings.Contains(fastStatus.Output, "fast-done") {
		t.Errorf("fast action final status = %+v, want exit 0 and output containing %q", fastStatus, "fast-done")
	}

	if !a.GetInlineStatus(0, 0).Running {
		t.Error("slow action status = not running, want it still running after the fast one finished")
	}
	slowStatus := waitForInlineDone(t, a, 0, 0)
	if slowStatus.ExitCode != 0 || !strings.Contains(slowStatus.Output, "slow-done") {
		t.Errorf("slow action final status = %+v, want exit 0 and output containing %q", slowStatus, "slow-done")
	}
}

func TestGetInlineStatusPersistsAfterCompletion(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hello-inline"})

	if err := a.RunActionInline(0, 0); err != nil {
		t.Fatalf("RunActionInline() error = %v", err)
	}
	waitForInlineDone(t, a, 0, 0)

	for i := 0; i < 3; i++ {
		status := a.GetInlineStatus(0, 0)
		if status.Running || status.ExitCode != 0 || !strings.Contains(status.Output, "hello-inline") {
			t.Errorf("poll #%d after completion = %+v, want the same finished result every time", i, status)
		}
	}
}

func TestGetInlineStatusReflectsPartialOutputWhileRunning(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Slow", Cmd: "echo first; sleep 2; echo second"})

	if err := a.RunActionInline(0, 0); err != nil {
		t.Fatalf("RunActionInline() error = %v", err)
	}
	waitForInlineRunning(t, a, 0, 0)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		status := a.GetInlineStatus(0, 0)
		if strings.Contains(status.Output, "first") {
			if strings.Contains(status.Output, "second") {
				t.Fatalf("saw both lines while still mid-sleep: %+v", status)
			}
			if !status.Running {
				t.Fatalf("status already reports done while checking partial output: %+v", status)
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
		if time.Now().After(deadline) {
			t.Fatal("never observed partial output (\"first\") while the process was still running")
		}
	}

	status := waitForInlineDone(t, a, 0, 0)
	if status.ExitCode != 0 || !strings.Contains(status.Output, "first") || !strings.Contains(status.Output, "second") {
		t.Errorf("final status = %+v, want exit 0 and both lines", status)
	}
}

func TestCancelInlineActionKillsProcessTree(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Sleep", Cmd: "sleep 30"})

	if err := a.RunActionInline(0, 0); err != nil {
		t.Fatalf("RunActionInline() error = %v", err)
	}
	waitForInlineRunning(t, a, 0, 0)

	if err := a.CancelInlineAction(0, 0); err != nil {
		t.Fatalf("CancelInlineAction() error = %v", err)
	}
	waitForInlineDone(t, a, 0, 0)
}

func TestCancelInlineActionNoneRunning(t *testing.T) {
	a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hi"})
	if err := a.CancelInlineAction(0, 0); err == nil {
		t.Error("expected an error when no inline action is running")
	}
}

func TestSetConfigResetsSessionOnlyWhenTheFileChanges(t *testing.T) {
	newRun := func(t *testing.T, a *App, key inlineKey) string {
		t.Helper()
		outPath := filepath.Join(t.TempDir(), "out.log")
		if err := os.WriteFile(outPath, []byte("previous output"), 0o600); err != nil {
			t.Fatal(err)
		}
		a.inlineRuns[key] = &inlineRun{outPath: outPath, exitCode: 0}
		return outPath
	}

	t.Run("a different config clears the recorded runs", func(t *testing.T) {
		a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hi"})
		a.cfg.SourcePath = "/first.yaml"
		outPath := newRun(t, a, inlineKey{itemIndex: 0, actionIndex: 0})

		a.setConfig(&config.Config{SourcePath: "/second.yaml"})

		if status := a.GetInlineStatus(0, 0); status.Output != "" || status.ExitCode != 0 {
			t.Errorf("GetInlineStatus after switching config = %+v, want the zero value", status)
		}
		if _, err := os.Stat(outPath); !os.IsNotExist(err) {
			t.Errorf("output file of the previous config was left behind: %v", err)
		}
	})

	t.Run("reloading the same config keeps them", func(t *testing.T) {
		a := inlineTestApp(config.Action{Title: "Echo", Cmd: "echo hi"})
		a.cfg.SourcePath = "/same.yaml"
		newRun(t, a, inlineKey{itemIndex: 0, actionIndex: 0})

		a.setConfig(&config.Config{SourcePath: "/same.yaml"})

		if status := a.GetInlineStatus(0, 0); status.Output != "previous output" {
			t.Errorf("GetInlineStatus after reloading the same config = %+v, want the output kept", status)
		}
	})
}
