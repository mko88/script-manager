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
	return NewApp(func() (*config.Config, error) {
		return &config.Config{
			Shell:   []string{"bash", "-c"},
			Items:   []map[string]any{{"name": "test"}},
			Actions: actions,
		}, nil
	})
}

// waitForInlineDone polls GetInlineStatus until Running is false, returning
// the final status — used by tests exercising RunActionInline, which starts
// the process and returns immediately, with GetInlineStatus as the only way
// to observe progress and completion.
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

// TestRunActionInlineClearsStaleEntryOnFailedRestart guards against a real
// but hard-to-hit-through-the-UI gap: a key with a finished run's entry
// still in a.inlineRuns must not keep reporting that run's exit code/output
// once a later RunActionInline call for the same key fails before ever
// starting a new process — the whole entry, not just its output file, has
// to be cleared, or GetInlineStatus would misreport a stale result.
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

// waitForInlineRunning blocks until GetInlineStatus reports Running for the
// given item/action pair — used by tests that need RunActionInline's
// background process still active so the test's own goroutine can act
// concurrently against it (CancelInlineAction, or a second RunActionInline
// for the same pair expected to be rejected).
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

// TestRunActionInlineAllowsConcurrentDifferentActions is the backend half of
// "switch to another action while one is running" — a second, different
// action must be free to start and run to completion independently while
// the first is still going, each tracked under its own item/action key.
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

	// The slow action should still be unaffected, running independently.
	if !a.GetInlineStatus(0, 0).Running {
		t.Error("slow action status = not running, want it still running after the fast one finished")
	}
	slowStatus := waitForInlineDone(t, a, 0, 0)
	if slowStatus.ExitCode != 0 || !strings.Contains(slowStatus.Output, "slow-done") {
		t.Errorf("slow action final status = %+v, want exit 0 and output containing %q", slowStatus, "slow-done")
	}
}

// TestGetInlineStatusPersistsAfterCompletion is the backend half of
// "switching back to a finished action still shows its result" — a
// completed run's status must stay readable indefinitely (not just once)
// until a new run for that same key replaces it.
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
	// A child process the shell spawns and waits on, so cancel only truly
	// works if it kills the whole process group/tree — killing just the
	// shell would silently orphan this sleep, leaving it running.
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
