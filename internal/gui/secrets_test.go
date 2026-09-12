package gui

import (
	"testing"

	"script-manager/internal/config"
	"script-manager/internal/secret"
)

const pinAction = 0
const plainAction = 1

func lockedConfig(t *testing.T, pin, plaintext string) (*config.Config, []byte) {
	t.Helper()
	params, key, err := secret.NewParams(pin)
	if err != nil {
		t.Fatalf("NewParams() error = %v", err)
	}
	enc, err := secret.Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	return &config.Config{
		Secrets: &params,
		Actions: []config.Action{
			{ID: "needs-pin", Title: "Needs PIN", Cmd: "echo $PASSWORD", RequiresPIN: true},
			{ID: "no-pin", Title: "No PIN", Cmd: "echo $PASSWORD"},
		},
		Items: []map[string]any{{"name": "one", "password": enc, "host": "example"}},
	}, key
}

func TestActionWithoutRequiresPINLeavesLockedValuesUnset(t *testing.T) {
	cfg, _ := lockedConfig(t, "demo1234", "hunter2")
	a := &App{cfg: cfg}

	if a.ActionNeedsUnlock(0, plainAction) {
		t.Error("ActionNeedsUnlock() = true for an action that doesn't require a PIN")
	}

	merged, err := a.mergedItemForRun(cfg.Items[0], cfg.Actions[plainAction])
	if err != nil {
		t.Fatalf("mergedItemForRun() error = %v, want no error without a PIN", err)
	}
	if _, present := merged["password"]; present {
		t.Error("a locked value was passed to an action that doesn't require a PIN; it must be left unset")
	}
	if merged["host"] != "example" {
		t.Error("unlocked values should still be passed through")
	}
}

func TestActionWithRequiresPINNeedsUnlockThenDecrypts(t *testing.T) {
	cfg, _ := lockedConfig(t, "demo1234", "hunter2")
	a := &App{cfg: cfg}

	if !a.ActionNeedsUnlock(0, pinAction) {
		t.Fatal("ActionNeedsUnlock() = false for a PIN-requiring action with a locked value")
	}
	if _, err := a.mergedItemForRun(cfg.Items[0], cfg.Actions[pinAction]); err != secret.ErrLocked {
		t.Errorf("mergedItemForRun() before unlocking = %v, want ErrLocked", err)
	}

	if err := a.UnlockSecrets("demo1234"); err != nil {
		t.Fatalf("UnlockSecrets() error = %v", err)
	}
	if a.ActionNeedsUnlock(0, pinAction) {
		t.Error("ActionNeedsUnlock() = true after unlocking")
	}

	merged, err := a.mergedItemForRun(cfg.Items[0], cfg.Actions[pinAction])
	if err != nil {
		t.Fatalf("mergedItemForRun() after unlocking error = %v", err)
	}
	if merged["password"] != "hunter2" {
		t.Errorf("password = %v, want the decrypted value", merged["password"])
	}
}

func TestSwitchingConfigDropsTheSessionKey(t *testing.T) {
	first, _ := lockedConfig(t, "demo1234", "first-secret")
	second, _ := lockedConfig(t, "demo1234", "second-secret")

	a := &App{cfg: first}
	if err := a.UnlockSecrets("demo1234"); err != nil {
		t.Fatalf("UnlockSecrets() error = %v", err)
	}
	if !a.SecretsState().Unlocked {
		t.Fatal("expected the config to be unlocked")
	}

	a.setConfig(second)

	if a.SecretsState().Unlocked {
		t.Error("the session key survived a switch to a different config")
	}
	if !a.ActionNeedsUnlock(0, pinAction) {
		t.Error("ActionNeedsUnlock() = false after switching configs; the user would never be prompted")
	}
}

func TestReloadingTheSameConfigKeepsTheSessionKey(t *testing.T) {
	cfg, _ := lockedConfig(t, "demo1234", "secret")

	a := &App{cfg: cfg}
	if err := a.UnlockSecrets("demo1234"); err != nil {
		t.Fatalf("UnlockSecrets() error = %v", err)
	}

	sameParams := *cfg.Secrets
	reloaded := &config.Config{Secrets: &sameParams, Actions: cfg.Actions, Items: cfg.Items}
	a.setConfig(reloaded)

	if !a.SecretsState().Unlocked {
		t.Error("reloading the same config re-locked it; the PIN should only be asked again when the config changes")
	}
}

func TestStaleKeyIsDroppedWhenDecryptionFails(t *testing.T) {
	_, firstKey := lockedConfig(t, "demo1234", "first-secret")
	second, _ := lockedConfig(t, "demo1234", "second-secret")

	a := &App{cfg: second}
	a.secretKey = firstKey
	a.secretParams = second.Secrets

	if _, err := a.mergedItemForRun(second.Items[0], second.Actions[pinAction]); err == nil {
		t.Fatal("expected decryption with a foreign key to fail")
	}
	if a.SecretsState().Unlocked {
		t.Error("a key that failed to decrypt was kept; the next run would fail the same way instead of prompting")
	}
}

func TestActionNeedsUnlockOnlyWhenSomethingIsLocked(t *testing.T) {
	cfg, _ := lockedConfig(t, "demo1234", "secret")
	cfg.Items = append(cfg.Items, map[string]any{"name": "plain", "password": "not-locked"})

	a := &App{cfg: cfg}

	if !a.ActionNeedsUnlock(0, pinAction) {
		t.Error("ActionNeedsUnlock() = false for an item with a locked value")
	}
	if a.ActionNeedsUnlock(1, pinAction) {
		t.Error("ActionNeedsUnlock() = true for an item with nothing locked")
	}
}
