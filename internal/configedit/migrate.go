package configedit

import (
	"fmt"

	"script-manager/internal/configmigrate"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ensureFormat converts a pre-env config after asking, and reports
// configmigrate.ErrDeclined when the user says no.
func (a *App) ensureFormat(path string) error {
	_, err := configmigrate.Ensure(path, a.approveConversion)
	return err
}

func (a *App) approveConversion(path, backupPath string) bool {
	choice, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:  runtime.QuestionDialog,
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
