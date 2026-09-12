package ui

import (
	"time"

	"script-manager/internal/action"
	"script-manager/internal/config"
	"script-manager/internal/secret"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) needsUnlock(act config.Action) bool {
	if !act.RequiresPIN || len(a.secretKey) > 0 {
		return false
	}
	item := a.list.Selected()
	if item == nil {
		return false
	}
	return secret.HasLocked(action.Merge(a.globalEnv, item))
}

func (a *App) startPINPrompt(act config.Action) tea.Cmd {
	if a.cfg.Secrets == nil {
		return a.flashMessage("This config has locked values but no secrets block — re-save it in the config editor", 4*time.Second)
	}
	a.pinPrompt = true
	a.pinEntry = ""
	a.pinPending = act
	a.showPINPrompt("")
	return nil
}

func (a *App) showPINPrompt(suffix string) {
	masked := ""
	for range a.pinEntry {
		masked += "•"
	}
	a.status.SetMessage("PIN: " + masked + suffix + "   (enter to unlock, esc to cancel)")
}

func (a *App) cancelPINPrompt() {
	a.pinPrompt = false
	a.pinEntry = ""
	a.status.ClearMessage()
}

func (a *App) updatePINPrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		a.cancelPINPrompt()
		return a, nil
	case "backspace":
		if runes := []rune(a.pinEntry); len(runes) > 0 {
			a.pinEntry = string(runes[:len(runes)-1])
		}
		a.showPINPrompt("")
		return a, nil
	case "enter":
		key, err := a.cfg.Secrets.Unlock(a.pinEntry)
		if err != nil {
			a.pinEntry = ""
			a.showPINPrompt("  wrong PIN, try again")
			return a, nil
		}
		a.secretKey = key
		a.secretParams = a.cfg.Secrets
		act := a.pinPending
		a.pinPrompt = false
		a.pinEntry = ""
		a.status.ClearMessage()
		a.onItemChanged()
		return a, a.execAction(act)
	}

	if msg.Type == tea.KeyRunes {
		a.pinEntry += string(msg.Runes)
		a.showPINPrompt("")
	}
	return a, nil
}
