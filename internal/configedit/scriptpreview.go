package configedit

import "script-manager/internal/scriptsource"

type ScriptPreviewDTO struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

func (a *App) PreviewScriptFile(path string) ScriptPreviewDTO {
	if path == "" {
		return ScriptPreviewDTO{}
	}
	content, err := scriptsource.Read(path)
	if err != nil {
		return ScriptPreviewDTO{Error: err.Error()}
	}
	return ScriptPreviewDTO{Content: content}
}
