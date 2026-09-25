package configedit

import (
	"fmt"

	"script-manager/internal/config"
	"script-manager/internal/secret"
)

type SecretsStateDTO struct {
	Configured bool `json:"configured"`
	Unlocked   bool `json:"unlocked"`
}

func secretsToDTO(p *secret.Params) *SecretsDTO {
	if p == nil {
		return nil
	}
	return &SecretsDTO{Salt: p.Salt, Time: p.Time, Memory: p.Memory, Threads: p.Threads, Check: p.Check}
}

func secretsFromDTO(d *SecretsDTO) *secret.Params {
	if d == nil {
		return nil
	}
	return &secret.Params{Salt: d.Salt, Time: d.Time, Memory: d.Memory, Threads: d.Threads, Check: d.Check}
}

func (a *App) SecretsState(params *SecretsDTO) SecretsStateDTO {
	return SecretsStateDTO{Configured: params != nil, Unlocked: len(a.secretKey) > 0}
}

func (a *App) CreateSecretsPIN(pin string) (*SecretsDTO, error) {
	params, key, err := secret.NewParams(pin)
	if err != nil {
		return nil, err
	}
	a.secretKey = key
	a.secretParams = &params
	return secretsToDTO(&params), nil
}

func (a *App) UnlockSecrets(pin string, params *SecretsDTO) error {
	p := secretsFromDTO(params)
	if p == nil {
		return fmt.Errorf("this config has no PIN set yet")
	}
	key, err := p.Unlock(pin)
	if err != nil {
		return err
	}
	a.secretKey = key
	a.secretParams = p
	return nil
}

func (a *App) LockSecrets() {
	a.secretKey = nil
	a.secretParams = nil
}

func (a *App) ChangeSecretsPIN(oldPIN, newPIN string, dto ConfigDTO) (ConfigDTO, error) {
	current := secretsFromDTO(dto.Secrets)
	if current == nil {
		return dto, fmt.Errorf("this config has no PIN set yet")
	}
	oldKey, err := current.Unlock(oldPIN)
	if err != nil {
		return dto, err
	}
	params, newKey, err := secret.NewParams(newPIN)
	if err != nil {
		return dto, err
	}

	type rekeyed struct {
		fields []FieldDTO
		index  int
		value  string
	}
	var pending []rekeyed

	collect := func(fields []FieldDTO) error {
		for i, f := range fields {
			if !secret.IsLocked(f.Value) {
				continue
			}
			plain, err := secret.Decrypt(oldKey, f.Value)
			if err != nil {
				return fmt.Errorf("%q could not be decrypted with the current PIN, so the PIN was not changed", f.Key)
			}
			reEncrypted, err := secret.Encrypt(newKey, plain)
			if err != nil {
				return err
			}
			pending = append(pending, rekeyed{fields, i, reEncrypted})
		}
		return nil
	}

	if err := collect(dto.EnvFields); err != nil {
		return dto, err
	}
	for _, item := range dto.Items {
		if err := collect(item.Fields); err != nil {
			return dto, err
		}
	}

	for _, p := range pending {
		p.fields[p.index].Value = p.value
	}
	dto.Secrets = secretsToDTO(&params)
	a.secretKey = newKey
	a.secretParams = &params
	return dto, nil
}

// RemoveSecretsPIN keeps every locked field declared as a secret but empties
// it; the next save deletes this user's local secrets file.
func (a *App) RemoveSecretsPIN(pin string, dto ConfigDTO) (ConfigDTO, error) {
	current := secretsFromDTO(dto.Secrets)
	if current == nil {
		return dto, fmt.Errorf("this config has no PIN set")
	}
	if _, err := current.Unlock(pin); err != nil {
		return dto, err
	}
	unset := func(fields []FieldDTO) {
		for i := range fields {
			if fields[i].Locked {
				fields[i] = FieldDTO{Key: fields[i].Key, Kind: "string", Secret: true, Locked: true, Unset: true}
			}
		}
	}
	unset(dto.EnvFields)
	for i := range dto.Items {
		unset(dto.Items[i].Fields)
	}
	dto.Secrets = nil
	a.secretKey = nil
	a.secretParams = nil
	a.dropLocalSecrets = true
	return dto, nil
}

func (a *App) NewSecretsID() (string, error) {
	return config.NewSecretsID()
}

func (a *App) SecretsFilePath(id string) string {
	return config.LocalSecretsPath(id)
}

func (a *App) LockValue(plain string) (string, error) {
	if len(a.secretKey) == 0 {
		return "", fmt.Errorf("enter the config PIN first")
	}
	if secret.IsLocked(plain) {
		return plain, nil
	}
	return secret.Encrypt(a.secretKey, plain)
}

func (a *App) RevealValue(value string) (string, error) {
	if !secret.IsLocked(value) {
		return value, nil
	}
	if len(a.secretKey) == 0 {
		return "", fmt.Errorf("enter the config PIN first")
	}
	return secret.Decrypt(a.secretKey, value)
}
