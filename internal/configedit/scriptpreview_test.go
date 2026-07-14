package configedit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewScriptFileEmptyPath(t *testing.T) {
	a := NewApp("")
	got := a.PreviewScriptFile("")
	if got.Error != "" || got.Content != "" {
		t.Errorf("PreviewScriptFile(\"\") = %+v, want a zero value", got)
	}
}

func TestPreviewScriptFileMissing(t *testing.T) {
	a := NewApp("")
	got := a.PreviewScriptFile(filepath.Join(t.TempDir(), "does-not-exist.sh"))
	if got.Error == "" {
		t.Error("expected an error for a missing file")
	}
	if got.Content != "" {
		t.Errorf("Content = %q, want empty on error", got.Content)
	}
}

func TestPreviewScriptFileReadsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deploy.sh")
	content := "#!/bin/bash\necho \"hello\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp("")
	got := a.PreviewScriptFile(path)
	if got.Error != "" {
		t.Fatalf("unexpected error: %s", got.Error)
	}
	if got.Content != content {
		t.Errorf("Content = %q, want %q", got.Content, content)
	}
}
