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
	manual := msg.manual || a.checkWanted
	a.checkWanted = false
	// A result that arrives under a prompt only updates the hint.
	quiet := !manual || a.modalOpen()
	if manual && !a.modalOpen() {
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
	case a.updater == nil:
		return nil
	case a.checkingUpdate:
		// The startup check is still running; report its result instead.
		a.checkWanted = true
		a.setPrompt("Checking for updates…")
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
		where := "the project's releases page"
		if a.updateResult.ReleaseURL != "" {
			where = a.updateResult.ReleaseURL
		}
		return a.flashMessage(a.updateResult.Latest+" has no checksums, so it can't be installed here; download it from "+where, 10*time.Second)
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
	a.swapping.Store(false)
	a.setPrompt("Downloading and installing " + a.updateResult.Latest + "…  (Ctrl+C to cancel)")
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelInstall = cancel
	c := *a.updater
	c.BeforeSwap = func() { a.swapping.Store(true) }
	res, exe := a.updateResult, a.updateExe
	return a, func() tea.Msg {
		defer cancel()
		return updateInstalledMsg{c.Install(ctx, res, exe)}
	}
}

// installingKey ignores every key except Ctrl+C, which cancels the install
// until the files start being replaced.
func (a *App) installingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() != "ctrl+c" || a.cancelInstall == nil {
		return a, nil
	}
	if a.swapping.Load() {
		a.setPrompt("Too late to cancel: the apps are being replaced…")
		return a, nil
	}
	a.cancelInstall()
	a.setPrompt("Cancelling the update…")
	return a, nil
}

func (a *App) onUpdateInstalled(msg updateInstalledMsg) tea.Cmd {
	a.installing = false
	a.cancelInstall = nil
	switch {
	case errors.Is(msg.err, update.ErrStalled):
		return a.flashMessage("Update failed: the download stalled; nothing was changed. Try again later.", 10*time.Second)
	case errors.Is(msg.err, context.Canceled):
		return a.flashMessage("Update cancelled; nothing was changed", 4*time.Second)
	case msg.err != nil:
		return a.flashMessage("Update failed: "+msg.err.Error(), 10*time.Second)
	}
	a.restart = true
	return tea.Quit
}
