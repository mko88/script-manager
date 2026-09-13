package configedit

import (
	"strings"

	"script-manager/internal/configmigrate"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ensureFormat converts a pre-env config after asking, and reports
// configmigrate.ErrDeclined when the user says no.
func (a *App) ensureFormat(path string) error {
	_, err := configmigrate.Ensure(path, a.approveConversion)
	return err
}

// A QuestionDialog is a Yes/No box on both Windows and Linux — Buttons is
// ignored there, so only "Yes" can mean yes.
func (a *App) approveConversion(path, backupPath string) bool {
	choice, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.QuestionDialog,
		Title:   "Convert config?",
		Message: configmigrate.PromptMessage(path, backupPath),
	})
	return err == nil && strings.EqualFold(choice, "Yes")
}
