package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const Prefix = "sm-enc:v1:"

const MinPINLength = 4

const (
	defaultTime    uint32 = 3
	defaultMemory  uint32 = 64 * 1024
	defaultThreads uint8  = 4
	keyLength      uint32 = 32
	saltLength            = 16
)

const checkPlaintext = "script-manager"

var (
	ErrWrongPIN    = errors.New("wrong PIN")
	ErrLocked      = errors.New("value is locked and no PIN has been entered")
	ErrPINTooShort = fmt.Errorf("PIN must be at least %d characters", MinPINLength)
)

type Params struct {
	Salt    string `yaml:"salt" json:"salt"`
	Time    uint32 `yaml:"time" json:"time"`
	Memory  uint32 `yaml:"memory" json:"memory"`
	Threads uint8  `yaml:"threads" json:"threads"`
	Check   string `yaml:"check" json:"check"`
}

func NewParams(pin string) (Params, []byte, error) {
	if len([]rune(pin)) < MinPINLength {
		return Params{}, nil, ErrPINTooShort
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return Params{}, nil, err
	}
	p := Params{
		Salt:    base64.StdEncoding.EncodeToString(salt),
		Time:    defaultTime,
		Memory:  defaultMemory,
		Threads: defaultThreads,
	}
	key := p.deriveKey(pin, salt)
	check, err := Encrypt(key, checkPlaintext)
	if err != nil {
		return Params{}, nil, err
	}
	p.Check = check
	return p, key, nil
}

func (p Params) Unlock(pin string) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(p.Salt)
	if err != nil || len(salt) == 0 {
		return nil, errors.New("config secrets block has no usable salt")
	}
	key := p.deriveKey(pin, salt)
	if p.Check == "" {
		return key, nil
	}
	if _, err := Decrypt(key, p.Check); err != nil {
		return nil, ErrWrongPIN
	}
	return key, nil
}

func SameParams(a, b *Params) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Salt == b.Salt && a.Check == b.Check &&
		a.Time == b.Time && a.Memory == b.Memory && a.Threads == b.Threads
}

func (p Params) deriveKey(pin string, salt []byte) []byte {
	t, m, th := p.Time, p.Memory, p.Threads
	if t == 0 {
		t = defaultTime
	}
	if m == 0 {
		m = defaultMemory
	}
	if th == 0 {
		th = defaultThreads
	}
	return argon2.IDKey([]byte(pin), salt, t, m, th, keyLength)
}

func Encrypt(key []byte, plaintext string) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return Prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func Decrypt(key []byte, value string) (string, error) {
	if !IsLocked(value) {
		return value, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, Prefix))
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrWrongPIN
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrWrongPIN
	}
	return string(plaintext), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func IsLocked(value string) bool {
	return strings.HasPrefix(value, Prefix)
}

func IsLockedValue(v any) bool {
	s, ok := v.(string)
	return ok && IsLocked(s)
}

func HasLocked(values map[string]any) bool {
	for _, v := range values {
		if IsLockedValue(v) {
			return true
		}
	}
	return false
}

const LockedDisplayText = "(locked)"

func Reveal(values map[string]any, key []byte) (map[string]any, error) {
	if !HasLocked(values) {
		return values, nil
	}
	if len(key) == 0 {
		return nil, ErrLocked
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		if s, ok := v.(string); ok && IsLocked(s) {
			plain, err := Decrypt(key, s)
			if err != nil {
				return nil, err
			}
			out[k] = plain
			continue
		}
		out[k] = v
	}
	return out, nil
}

func Redact(values map[string]any) map[string]any {
	if !HasLocked(values) {
		return values
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		if IsLockedValue(v) {
			out[k] = LockedDisplayText
			continue
		}
		out[k] = v
	}
	return out
}

func Strip(values map[string]any) map[string]any {
	if !HasLocked(values) {
		return values
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		if IsLockedValue(v) {
			continue
		}
		out[k] = v
	}
	return out
}

func Display(values map[string]any, key []byte) map[string]any {
	revealed, err := Reveal(values, key)
	if err != nil {
		return Redact(values)
	}
	return revealed
}
