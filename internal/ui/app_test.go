package ui

import (
	"errors"
	"testing"

	"script-manager/internal/config"
)

func TestInitFlashesLoadError(t *testing.T) {
	a := NewApp(&config.Config{}, nil, errors.New("boom"))
	if cmd := a.Init(); cmd == nil {
		t.Fatal("expected a non-nil Init command when loadErr is set")
	}
	if a.status.message == "" {
		t.Fatal("expected a status message to be set immediately after Init")
	}
}

func TestInitNoLoadError(t *testing.T) {
	a := NewApp(&config.Config{}, nil, nil)
	a.Init()
	if a.status.message != "" {
		t.Errorf("expected no status message without a load error, got %q", a.status.message)
	}
}
