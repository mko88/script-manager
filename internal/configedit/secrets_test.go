package configedit

import (
	"testing"

	"script-manager/internal/secret"
)

func lockedDraft(t *testing.T, pin string) (*App, ConfigDTO) {
	t.Helper()
	a := &App{}
	params, err := a.CreateSecretsPIN(pin)
	if err != nil {
		t.Fatalf("CreateSecretsPIN() error = %v", err)
	}
	envValue, err := a.LockValue("vault-token")
	if err != nil {
		t.Fatalf("LockValue() error = %v", err)
	}
	itemValue, err := a.LockValue("item-password")
	if err != nil {
		t.Fatalf("LockValue() error = %v", err)
	}
	return a, ConfigDTO{
		Secrets:   params,
		EnvFields: []FieldDTO{{Key: "vault_token", Kind: "string", Value: envValue, Locked: true}},
		Items: []ItemDTO{{
			Name:   "one",
			Fields: []FieldDTO{{Key: "password", Kind: "string", Value: itemValue, Locked: true}, {Key: "host", Kind: "string", Value: "example"}},
		}},
	}
}

func TestChangePINReEncryptsEveryLockedValue(t *testing.T) {
	a, dto := lockedDraft(t, "demo1234")
	oldEnv := dto.EnvFields[0].Value
	oldItem := dto.Items[0].Fields[0].Value

	updated, err := a.ChangeSecretsPIN("demo1234", "newpin5678", dto)
	if err != nil {
		t.Fatalf("ChangeSecretsPIN() error = %v", err)
	}

	if updated.EnvFields[0].Value == oldEnv || updated.Items[0].Fields[0].Value == oldItem {
		t.Error("locked values were not re-encrypted under the new PIN")
	}
	if updated.Items[0].Fields[1].Value != "example" {
		t.Error("an unlocked value was modified")
	}

	params := secretsFromDTO(updated.Secrets)
	if _, err := params.Unlock("demo1234"); err != secret.ErrWrongPIN {
		t.Error("the old PIN still unlocks the config after a PIN change")
	}
	newKey, err := params.Unlock("newpin5678")
	if err != nil {
		t.Fatalf("the new PIN does not unlock the config: %v", err)
	}

	got, err := secret.Decrypt(newKey, updated.EnvFields[0].Value)
	if err != nil || got != "vault-token" {
		t.Errorf("env value after the PIN change = %q (err %v), want %q", got, err, "vault-token")
	}
	got, err = secret.Decrypt(newKey, updated.Items[0].Fields[0].Value)
	if err != nil || got != "item-password" {
		t.Errorf("item value after the PIN change = %q (err %v), want %q", got, err, "item-password")
	}
}

func TestChangePINRejectsWrongCurrentPIN(t *testing.T) {
	a, dto := lockedDraft(t, "demo1234")
	before := dto.EnvFields[0].Value

	if _, err := a.ChangeSecretsPIN("wrongpin", "newpin5678", dto); err != secret.ErrWrongPIN {
		t.Errorf("ChangeSecretsPIN() with a wrong current PIN = %v, want ErrWrongPIN", err)
	}
	if dto.EnvFields[0].Value != before {
		t.Error("a rejected PIN change still modified the values")
	}
}

func TestChangePINRejectsShortNewPIN(t *testing.T) {
	a, dto := lockedDraft(t, "demo1234")
	before := dto.EnvFields[0].Value

	if _, err := a.ChangeSecretsPIN("demo1234", "12", dto); err == nil {
		t.Error("ChangeSecretsPIN() accepted a new PIN below the minimum length")
	}
	if dto.EnvFields[0].Value != before {
		t.Error("a rejected PIN change still modified the values")
	}
}

func TestChangePINIsAtomicWhenOneValueFails(t *testing.T) {
	a, dto := lockedDraft(t, "demo1234")
	envBefore := dto.EnvFields[0].Value
	dto.Items[0].Fields[0].Value = secret.Prefix + "bm90LXJlYWwtY2lwaGVydGV4dA=="

	if _, err := a.ChangeSecretsPIN("demo1234", "newpin5678", dto); err == nil {
		t.Fatal("ChangeSecretsPIN() succeeded despite a value that could not be decrypted")
	}
	if dto.EnvFields[0].Value != envBefore {
		t.Error("a failed PIN change re-encrypted some values anyway; they would no longer match the stored secrets block")
	}
	if dto.Secrets == nil || secretsFromDTO(dto.Secrets).Salt == "" {
		t.Error("the secrets block was replaced despite the failure")
	}
}

func TestRemovePINDecryptsEverythingAndKeepsValuesMasked(t *testing.T) {
	a, dto := lockedDraft(t, "demo1234")

	updated, err := a.RemoveSecretsPIN("demo1234", dto)
	if err != nil {
		t.Fatalf("RemoveSecretsPIN() error = %v", err)
	}

	if updated.Secrets != nil {
		t.Error("the secrets block survived removing the PIN")
	}
	env := updated.EnvFields[0]
	if env.Value != "vault-token" {
		t.Errorf("env value = %q, want the decrypted plaintext", env.Value)
	}
	if env.Locked {
		t.Error("the field is still marked locked after the PIN was removed")
	}
	if !env.Secret {
		t.Error("a previously locked value should stay marked secret so it is still masked on screen")
	}
	if item := updated.Items[0].Fields[0]; item.Value != "item-password" || item.Locked {
		t.Errorf("item value = %q locked=%v, want the decrypted plaintext and locked=false", item.Value, item.Locked)
	}
	if a.SecretsState(nil).Unlocked {
		t.Error("the session key was kept after the PIN was removed")
	}
}

func TestRemovePINRejectsWrongPIN(t *testing.T) {
	a, dto := lockedDraft(t, "demo1234")
	before := dto.EnvFields[0].Value

	if _, err := a.RemoveSecretsPIN("nope1234", dto); err != secret.ErrWrongPIN {
		t.Errorf("RemoveSecretsPIN() with a wrong PIN = %v, want ErrWrongPIN", err)
	}
	if dto.EnvFields[0].Value != before || dto.Secrets == nil {
		t.Error("a rejected removal still changed the draft")
	}
}

func TestLockValueNeedsAnUnlockedConfig(t *testing.T) {
	a := &App{}
	if _, err := a.LockValue("secret"); err == nil {
		t.Error("LockValue() succeeded with no PIN entered")
	}
}
