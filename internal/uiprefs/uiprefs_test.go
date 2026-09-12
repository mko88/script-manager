package uiprefs

import "testing"

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	if _, err := Save(dir, Prefs{FontUI: "Cascadia Code", FontMono: "Consolas", ScalePercent: 120}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got := Load(dir)
	if got.FontUI != "Cascadia Code" || got.FontMono != "Consolas" || got.ScalePercent != 120 {
		t.Errorf("Load() = %+v, want the saved values", got)
	}
}

func TestScaleIsClamped(t *testing.T) {
	dir := t.TempDir()

	for _, tc := range []struct {
		in, want int
	}{
		{0, ScaleDefault},
		{10, ScaleMin},
		{500, ScaleMax},
		{140, 140},
	} {
		got, err := Save(dir, Prefs{ScalePercent: tc.in})
		if err != nil {
			t.Fatalf("Save(%d) error = %v", tc.in, err)
		}
		if got.ScalePercent != tc.want {
			t.Errorf("Save(%d).ScalePercent = %d, want %d", tc.in, got.ScalePercent, tc.want)
		}
		if reloaded := Load(dir); reloaded.ScalePercent != tc.want {
			t.Errorf("Load() after Save(%d) = %d, want %d", tc.in, reloaded.ScalePercent, tc.want)
		}
	}
}

func TestFontNamesAreTrimmed(t *testing.T) {
	dir := t.TempDir()

	got, _ := Save(dir, Prefs{FontUI: "  Segoe UI  ", FontMono: "\tConsolas\n"})

	if got.FontUI != "Segoe UI" || got.FontMono != "Consolas" {
		t.Errorf("Save() = %q/%q, want the names trimmed", got.FontUI, got.FontMono)
	}
}

func TestMissingFileAndDirGiveDefaults(t *testing.T) {
	if got := Load(t.TempDir()); got != Defaults() {
		t.Errorf("Load() on a fresh dir = %+v, want defaults", got)
	}
	if got := Load(""); got != Defaults() {
		t.Errorf("Load(\"\") = %+v, want defaults", got)
	}
}
