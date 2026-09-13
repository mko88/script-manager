package gui

import (
	"fmt"

	"script-manager/internal/action"
	"script-manager/internal/config"
	"script-manager/internal/secret"
)

type SecretsStateDTO struct {
	Configured bool `json:"configured"`
	Unlocked   bool `json:"unlocked"`
	HasLocked  bool `json:"hasLocked"`
}

func (a *App) sessionKey() []byte {
	a.secretMu.RLock()
	defer a.secretMu.RUnlock()
	return a.secretKey
}

func (a *App) setConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}
	// A different file, not a reload of the same one — a reload keeps its
	// output, because the indices still mean what they meant.
	switched := a.configSourcePath() != "" && cfg.SourcePath != a.configSourcePath()

	a.secretMu.Lock()
	if !secret.SameParams(a.secretParams, cfg.Secrets) {
		a.secretKey = nil
		a.secretParams = nil
	}
	a.secretMu.Unlock()
	a.cfg = cfg
	a.watchConfigPath(cfg.SourcePath)

	if switched {
		a.resetSession()
	}
}

func (a *App) forgetSessionKey() {
	a.secretMu.Lock()
	a.secretKey = nil
	a.secretParams = nil
	a.secretMu.Unlock()
}

func (a *App) SecretsState() SecretsStateDTO {
	locked := secret.HasLocked(a.cfg.Env)
	for _, item := range a.cfg.Items {
		if locked {
			break
		}
		locked = secret.HasLocked(item.Env)
	}
	return SecretsStateDTO{
		Configured: a.cfg.Secrets != nil,
		Unlocked:   len(a.sessionKey()) > 0,
		HasLocked:  locked,
	}
}

func (a *App) UnlockSecrets(pin string) error {
	if a.cfg.Secrets == nil {
		return fmt.Errorf("this config has no PIN-protected values")
	}
	key, err := a.cfg.Secrets.Unlock(pin)
	if err != nil {
		return err
	}
	a.secretMu.Lock()
	a.secretKey = key
	a.secretParams = a.cfg.Secrets
	a.secretMu.Unlock()
	return nil
}

func (a *App) LockSecrets() {
	a.forgetSessionKey()
}

func (a *App) ActionNeedsUnlock(itemIndex, actionIndex int) bool {
	item := a.itemAt(itemIndex)
	if item == nil {
		return false
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return false
	}
	if !actions[actionIndex].RequiresPIN {
		return false
	}
	if !secret.HasLocked(action.Merge(a.cfg.Env, item)) {
		return false
	}
	return len(a.sessionKey()) == 0
}
