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
// checkOnStartup is set. exe is the running executable; its siblings are
// replaced too.
func (a *App) EnableUpdates(c *update.Client, exe string, checkOnStartup bool) {
	a.updater = c
	a.updateExe = exe
	a.checkOnStartup = checkOnStartup
}

// RestartRequested reports whether the app quit to start an installed
// update.
func (a *App) RestartRequested() bool { return a.restart }

func (a *App) startupUpdateCheck() tea.Cmd {
	if a.updater == nil || !a.checkOnStartup || !update.Checkable(a.updater.Current) {
		return nil
	}
	a.checkingUpdate = true
	return a.checkForUpdate(false)
}

func (a *App) checkForUpdate(manual bool) tea.Cmd {
	c := a.updater
	return func() tea.Msg {
		res, err := c.Check(context.Background())
		return updateCheckedMsg{res: res, err: err, manual: manual}
	}
}

// modalOpen reports whether a prompt owns the status bar and the keyboard.
func (a *App) modalOpen() bool {
	return a.pinPrompt || a.updateConfirm || a.installing
}

// setPrompt shows a status message that stays until replaced.
func (a *App) setPrompt(text string) {
	a.msgToken++
	a.status.SetMessage(text)
}

func (a *App) clearPrompt() {
	a.msgToken++
	a.status.ClearMessage()
}

func (a *App) onUpdateChecked(msg updateCheckedMsg) tea.Cmd {
	a.checkingUpdate = false
	// A result that arrives under a prompt only updates the hint.
	quiet := !msg.manual || a.modalOpen()
	if msg.manual && !a.modalOpen() {
		a.clearPrompt()
	}
	if msg.err != nil {
		if quiet {
			return nil
		}
		return a.flashMessage("Update check failed: "+msg.err.Error(), 5*time.Second)
	}
	if !msg.res.Available {
		a.updateResult = update.Result{}
		a.status.SetUpdateHint("", true)
		if quiet {
			return nil
		}
		return a.flashMessage("Up to date ("+msg.res.Current+")", 3*time.Second)
	}
	a.updateResult = msg.res
	a.status.SetUpdateHint(msg.res.Latest, msg.res.Installable)
	if quiet {
		return nil
	}
	return a.offerUpdate()
}

// onUpdateKey handles Ctrl+U: offer a known update, or check for one.
func (a *App) onUpdateKey() tea.Cmd {
	switch {
	case a.updater == nil || a.checkingUpdate:
		return nil
	case a.updateResult.Available:
		return a.offerUpdate()
	case !update.Checkable(a.updater.Current):
		return a.flashMessage(update.ErrNotCheckable.Error(), 3*time.Second)
	}
	a.checkingUpdate = true
	a.setPrompt("Checking for updates…")
	return a.checkForUpdate(true)
}

func (a *App) offerUpdate() tea.Cmd {
	if !a.updateResult.Installable {
		return a.flashMessage(a.updateResult.Latest+" has no checksums, so it can't be installed here; download it from "+a.updateResult.ReleaseURL, 10*time.Second)
	}
	a.updateConfirm = true
	a.setPrompt("Update to " + a.updateResult.Latest + " and restart? y: update, any other key: cancel. Replaces all three apps in this folder.")
	return nil
}

func (a *App) updateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	a.updateConfirm = false
	if s := msg.String(); s != "y" && s != "Y" {
		a.clearPrompt()
		return a, nil
	}
	a.installing = true
	a.setPrompt("Downloading and installing " + a.updateResult.Latest + "…  (Ctrl+C to cancel)")
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelInstall = cancel
	c, res, exe := a.updater, a.updateResult, a.updateExe
	return a, func() tea.Msg {
		defer cancel()
		return updateInstalledMsg{c.Install(ctx, res, exe)}
	}
}

// installingKey ignores every key except Ctrl+C, which cancels the install.
func (a *App) installingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" && a.cancelInstall != nil {
		a.cancelInstall()
		a.setPrompt("Cancelling the update…")
	}
	return a, nil
}

func (a *App) onUpdateInstalled(msg updateInstalledMsg) tea.Cmd {
	a.installing = false
	a.cancelInstall = nil
	switch {
	case errors.Is(msg.err, context.Canceled):
		return a.flashMessage("Update cancelled; nothing was changed", 4*time.Second)
	case msg.err != nil:
		return a.flashMessage("Update failed: "+msg.err.Error(), 10*time.Second)
	}
	a.restart = true
	return tea.Quit
}
