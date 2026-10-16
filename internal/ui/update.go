package ui

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"script-manager/internal/update"
)

type updateCheckedMsg struct {
	res    update.Result
	err    error
	manual bool
}

type updateInstalledMsg struct{ err error }

// EnableUpdates turns on Ctrl+U, and the background check at startup when
// checkOnStartup is set. dir holds the binaries to replace.
func (a *App) EnableUpdates(c *update.Client, dir string, checkOnStartup bool) {
	a.updater = c
	a.updateDir = dir
	a.checkOnStartup = checkOnStartup
}

// RestartRequested reports whether the app quit to start an installed
// update.
func (a *App) RestartRequested() bool { return a.restart }

func (a *App) startupUpdateCheck() tea.Cmd {
	if a.updater == nil || !a.checkOnStartup || !update.Checkable(a.updater.Current) {
		return nil
	}
	return a.checkForUpdate(false)
}

func (a *App) checkForUpdate(manual bool) tea.Cmd {
	c := a.updater
	return func() tea.Msg {
		res, err := c.Check(context.Background())
		return updateCheckedMsg{res: res, err: err, manual: manual}
	}
}

// setPrompt shows a status message that stays until replaced.
func (a *App) setPrompt(text string) {
	a.msgToken++
	a.status.SetMessage(text)
}

func (a *App) onUpdateChecked(msg updateCheckedMsg) tea.Cmd {
	a.checkingUpdate = false
	if msg.err != nil {
		if !msg.manual {
			return nil
		}
		return a.flashMessage("Update check failed: "+msg.err.Error(), 5*time.Second)
	}
	if !msg.res.Available {
		a.updateResult = update.Result{}
		a.status.SetUpdateHint("")
		if msg.manual {
			return a.flashMessage("Up to date ("+msg.res.Current+")", 3*time.Second)
		}
		return nil
	}
	a.updateResult = msg.res
	a.status.SetUpdateHint(msg.res.Latest)
	if msg.manual {
		a.askUpdate()
	}
	return nil
}

// onUpdateKey handles Ctrl+U: confirm a known update, or check for one.
func (a *App) onUpdateKey() tea.Cmd {
	switch {
	case a.updater == nil || a.installing || a.checkingUpdate:
		return nil
	case a.updateResult.Available:
		a.askUpdate()
		return nil
	case !update.Checkable(a.updater.Current):
		return a.flashMessage(update.ErrNotCheckable.Error(), 3*time.Second)
	}
	a.checkingUpdate = true
	a.setPrompt("Checking for updates…")
	return a.checkForUpdate(true)
}

func (a *App) askUpdate() {
	a.updateConfirm = true
	a.setPrompt("Update to " + a.updateResult.Latest + " and restart? y: update, any other key: cancel. Replaces all three apps in this folder.")
}

func (a *App) updateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	a.updateConfirm = false
	if s := msg.String(); s != "y" && s != "Y" {
		a.msgToken++
		a.status.ClearMessage()
		return a, nil
	}
	a.installing = true
	a.setPrompt("Downloading and installing " + a.updateResult.Latest + "…")
	c, res, dir := a.updater, a.updateResult, a.updateDir
	return a, func() tea.Msg {
		if dir == "" {
			return updateInstalledMsg{errors.New("can't locate the running executable")}
		}
		return updateInstalledMsg{c.Install(context.Background(), res, dir)}
	}
}

func (a *App) onUpdateInstalled(msg updateInstalledMsg) tea.Cmd {
	a.installing = false
	if msg.err != nil {
		return a.flashMessage("Update failed: "+msg.err.Error(), 8*time.Second)
	}
	a.restart = true
	return tea.Quit
}
