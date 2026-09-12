package secret

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	params, key, err := NewParams("1234")
	if err != nil {
		t.Fatalf("NewParams() error = %v", err)
	}

	enc, err := Encrypt(key, "hunter2")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if !IsLocked(enc) {
		t.Errorf("Encrypt() = %q, want the %q prefix", enc, Prefix)
	}
	if strings.Contains(enc, "hunter2") {
		t.Error("the plaintext is visible in the encrypted value")
	}

	got, err := Decrypt(key, enc)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got != "hunter2" {
		t.Errorf("Decrypt() = %q, want %q", got, "hunter2")
	}

	if _, err := params.Unlock("1234"); err != nil {
		t.Errorf("Unlock() with the right PIN failed: %v", err)
	}
}

func TestUnlockRejectsWrongPIN(t *testing.T) {
	params, _, err := NewParams("1234")
	if err != nil {
		t.Fatalf("NewParams() error = %v", err)
	}

	if _, err := params.Unlock("4321"); err != ErrWrongPIN {
		t.Errorf("Unlock() with a wrong PIN = %v, want ErrWrongPIN", err)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	_, key, _ := NewParams("1234")
	_, otherKey, _ := NewParams("1234")

	enc, _ := Encrypt(key, "hunter2")

	if _, err := Decrypt(otherKey, enc); err != ErrWrongPIN {
		t.Errorf("Decrypt() with a key from a different salt = %v, want ErrWrongPIN", err)
	}
}

func TestDecryptLeavesPlainValuesAlone(t *testing.T) {
	_, key, _ := NewParams("1234")

	got, err := Decrypt(key, "not encrypted")
	if err != nil {
		t.Fatalf("Decrypt() on a plain value error = %v", err)
	}
	if got != "not encrypted" {
		t.Errorf("Decrypt() = %q, want the value unchanged", got)
	}
}

func TestNewParamsRejectsShortPIN(t *testing.T) {
	if _, _, err := NewParams("12"); err == nil {
		t.Error("NewParams() accepted a PIN shorter than the minimum")
	}
}

func TestEachEncryptionIsDistinct(t *testing.T) {
	_, key, _ := NewParams("1234")

	first, _ := Encrypt(key, "same")
	second, _ := Encrypt(key, "same")

	if first == second {
		t.Error("encrypting the same plaintext twice produced identical ciphertext; the nonce is not random")
	}
}

func TestHasLocked(t *testing.T) {
	_, key, _ := NewParams("1234")
	enc, _ := Encrypt(key, "x")

	if HasLocked(map[string]any{"a": "plain", "n": 3}) {
		t.Error("HasLocked() = true for a map with no locked values")
	}
	if !HasLocked(map[string]any{"a": "plain", "b": enc}) {
		t.Error("HasLocked() = false for a map containing a locked value")
	}
}
