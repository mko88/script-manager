package gui

import (
	"errors"
	"fmt"
	"os"

	"script-manager/internal/configmigrate"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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

func (a *App) approveConversion(path, backupPath string) bool {
	choice, err := wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
		Type:  wailsruntime.QuestionDialog,
		Title: "Convert config?",
		Message: fmt.Sprintf(
			"%s stores item environment variables in the old format and has to be converted before it can be opened.\n\n"+
				"The original is saved as:\n%s\n\nComments in the file will be lost.",
			path, backupPath),
		Buttons:       []string{"Convert", "Cancel"},
		DefaultButton: "Convert",
		CancelButton:  "Cancel",
	})
	return err == nil && choice == "Convert"
}
