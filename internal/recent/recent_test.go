package recent

import (
	"path/filepath"
	"strconv"
	"testing"
)

func TestAddPrependsDedupesAndCaps(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < Max+3; i++ {
		Add(dir, filepath.Join(dir, "config"+strconv.Itoa(i)+".yaml"))
	}

	got := Load(dir)
	if len(got) != Max {
		t.Fatalf("Load() returned %d paths, want the %d most recent", len(got), Max)
	}
	newest := filepath.Join(dir, "config"+strconv.Itoa(Max+2)+".yaml")
	if got[0] != newest {
		t.Errorf("Load()[0] = %q, want the most recently added %q", got[0], newest)
	}

	Add(dir, got[3])
	got = Load(dir)
	if got[0] != filepath.Join(dir, "config"+strconv.Itoa(Max-1)+".yaml") {
		t.Errorf("re-adding an existing path should move it to the front, got %q", got[0])
	}
	if len(got) != Max {
		t.Errorf("re-adding an existing path changed the count: got %d, want %d", len(got), Max)
	}
}

func TestAddStoresAbsolutePaths(t *testing.T) {
	dir := t.TempDir()

	Add(dir, "config.yaml")

	got := Load(dir)
	if len(got) != 1 {
		t.Fatalf("Load() = %v, want one entry", got)
	}
	if !filepath.IsAbs(got[0]) {
		t.Errorf("Load()[0] = %q, want an absolute path", got[0])
	}
}

func TestRemoveAndClear(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yaml")
	b := filepath.Join(dir, "b.yaml")
	Add(dir, a)
	Add(dir, b)

	if got := Remove(dir, a); len(got) != 1 || got[0] != b {
		t.Errorf("Remove(a) = %v, want only %q left", got, b)
	}
	if got := Load(dir); len(got) != 1 {
		t.Errorf("Remove() didn't persist: Load() = %v", got)
	}

	if got := Clear(dir); len(got) != 0 {
		t.Errorf("Clear() = %v, want empty", got)
	}
	if got := Load(dir); len(got) != 0 {
		t.Errorf("Clear() didn't persist: Load() = %v", got)
	}
}

func TestLoadMissingOrUnwritableDir(t *testing.T) {
	if got := Load(t.TempDir()); len(got) != 0 {
		t.Errorf("Load() on a fresh dir = %v, want empty", got)
	}
	if got := Load(""); len(got) != 0 {
		t.Errorf("Load(\"\") = %v, want empty", got)
	}
	if got := Add("", "config.yaml"); len(got) != 0 {
		t.Errorf("Add with no app-data dir = %v, want empty", got)
	}
}
