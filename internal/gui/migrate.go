package gui

import (
	"errors"
	"os"

	"script-manager/internal/configmigrate"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ConversionPromptEvent carries a ConversionDTO to the frontend, which shows
// the dialog and replies through AnswerConversion.
const ConversionPromptEvent = "config:convert-prompt"

type ConversionDTO struct {
	Path    string `json:"path"`
	Backup  string `json:"backup"`
	Message string `json:"message"`
}

// ensureFormat converts a pre-env config after asking, and reports
// configmigrate.ErrDeclined when the user says no.
func (a *App) ensureFormat(path string) error {
	_, err := configmigrate.Ensure(path, a.approveConversion)
	return err
}

// ensureFormatOnChange is ensureFormat for reloads the user didn't ask for.
// A file the user already declined must not prompt again until it changes.
func (a *App) ensureFormatOnChange(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	a.declinedMu.Lock()
	declined := a.declinedAt.Equal(info.ModTime())
	a.declinedMu.Unlock()
	if declined {
		return configmigrate.ErrDeclined
	}

	err = a.ensureFormat(path)
	if errors.Is(err, configmigrate.ErrDeclined) {
		a.declinedMu.Lock()
		a.declinedAt = info.ModTime()
		a.declinedMu.Unlock()
	}
	return err
}

// approveConversion hands the question to the frontend and waits. Wails runs
// each binding call on its own goroutine, so blocking here doesn't stop the
// window from answering.
func (a *App) approveConversion(path, backupPath string) bool {
	answer := make(chan bool, 1)

	a.conversionMu.Lock()
	a.conversionAnswer = answer
	a.conversionMu.Unlock()

	defer func() {
		a.conversionMu.Lock()
		a.conversionAnswer = nil
		a.conversionMu.Unlock()
	}()

	wailsruntime.EventsEmit(a.ctx, ConversionPromptEvent, ConversionDTO{
		Path:    path,
		Backup:  backupPath,
		Message: configmigrate.PromptMessage(path, backupPath),
	})
	return <-answer
}

// AnswerConversion delivers the frontend's reply to the pending prompt.
func (a *App) AnswerConversion(convert bool) {
	a.conversionMu.Lock()
	answer := a.conversionAnswer
	a.conversionMu.Unlock()
	if answer != nil {
		answer <- convert
	}
}
