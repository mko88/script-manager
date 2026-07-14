package configedit

import "script-manager/internal/scriptsource"

// ScriptPreviewDTO is a script-mode action's source-file preview.
type ScriptPreviewDTO struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

// PreviewScriptFile reads path's content for the Action editor's "Script
// file" preview. path is used exactly as given — no {{.field}} template
// expansion — since ActionForm has no "current item" context to expand
// against (a global action isn't tied to one); a templated path just won't
// be found on disk, surfaced the same as any other missing file. An empty
// path returns a zero-value DTO (no error), so the preview area stays blank
// rather than showing a "file not found" for a field the user simply
// hasn't filled in yet.
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
