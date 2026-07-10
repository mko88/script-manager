package configedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"script-manager/internal/gui"
)

func TestMessagesPathFor(t *testing.T) {
	dir := "/exe/dir"

	if path, err := messagesPathFor(dir, "gui"); err != nil || path != filepath.Join(dir, gui.GUIMessagesFilename) {
		t.Errorf("messagesPathFor(gui) = (%q, %v)", path, err)
	}
	if path, err := messagesPathFor(dir, "configedit"); err != nil || path != filepath.Join(dir, configEditMessagesFilename) {
		t.Errorf("messagesPathFor(configedit) = (%q, %v)", path, err)
	}
	if _, err := messagesPathFor(dir, "bogus"); err == nil {
		t.Error("expected an error for an unknown target")
	}
}

func TestGetEditableMessagesSelf(t *testing.T) {
	dir := t.TempDir()
	a := &App{exeDir: dir, defaultMessages: []byte(`{"nav":{"messages":"Messages"}}`)}

	got, err := a.GetEditableMessages("configedit")
	if err != nil {
		t.Fatalf("GetEditableMessages(configedit) error = %v", err)
	}
	nav, _ := got["nav"].(map[string]interface{})
	if nav["messages"] != "Messages" {
		t.Errorf("got = %v, want nav.messages = Messages", got)
	}
}

func TestGetEditableMessagesGuiMissingFile(t *testing.T) {
	a := &App{exeDir: t.TempDir()}

	_, err := a.GetEditableMessages("gui")
	if err == nil {
		t.Fatal("expected an error when script-manager-gui has never run")
	}
	if !strings.Contains(err.Error(), "run it at least once") {
		t.Errorf("error = %q, want a hint to run script-manager-gui first", err.Error())
	}
}

func TestGetEditableMessagesGuiReadsExistingFile(t *testing.T) {
	dir := t.TempDir()
	guiPath := filepath.Join(dir, gui.GUIMessagesFilename)
	if err := os.WriteFile(guiPath, []byte(`{"nav":{"items":"Items"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{exeDir: dir}

	got, err := a.GetEditableMessages("gui")
	if err != nil {
		t.Fatalf("GetEditableMessages(gui) error = %v", err)
	}
	nav, _ := got["nav"].(map[string]interface{})
	if nav["items"] != "Items" {
		t.Errorf("got = %v, want nav.items = Items", got)
	}
}

func TestSaveMessagesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := &App{exeDir: dir}

	data := map[string]interface{}{"nav": map[string]interface{}{"items": "Edited"}}
	if err := a.SaveMessages("gui", data); err != nil {
		t.Fatalf("SaveMessages() error = %v", err)
	}

	got, err := a.GetEditableMessages("gui")
	if err != nil {
		t.Fatalf("GetEditableMessages(gui) error = %v", err)
	}
	nav, _ := got["nav"].(map[string]interface{})
	if nav["items"] != "Edited" {
		t.Errorf("got = %v, want nav.items = Edited", got)
	}
}

func TestSaveMessagesUnknownTarget(t *testing.T) {
	a := &App{exeDir: t.TempDir()}
	if err := a.SaveMessages("bogus", map[string]interface{}{}); err == nil {
		t.Error("expected an error for an unknown target")
	}
}
