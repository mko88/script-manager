package gui

import (
	"errors"
	"reflect"
	"testing"

	"script-manager/internal/config"
)

func TestLoadError(t *testing.T) {
	t.Run("surfaces the initial load error", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) { return &config.Config{}, errors.New("boom") })
		if got := a.LoadError(); got != "boom" {
			t.Errorf("LoadError() = %q, want %q", got, "boom")
		}
	})
	t.Run("empty when the initial load succeeds", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) { return &config.Config{}, nil })
		if got := a.LoadError(); got != "" {
			t.Errorf("LoadError() = %q, want empty", got)
		}
	})
}

func TestGetActionGroups(t *testing.T) {
	t.Run("converts the catalog", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) {
			return &config.Config{ActionGroups: []config.ActionGroup{
				{ID: "safe", Title: "Safe", Color: "#2ca02c"},
				{ID: "diagnostics"},
			}}, nil
		})
		got := a.GetActionGroups()
		want := []ActionGroupDTO{
			{ID: "safe", Title: "Safe", Color: "#2ca02c"},
			{ID: "diagnostics"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetActionGroups() = %+v, want %+v", got, want)
		}
	})
	t.Run("empty catalog returns an empty (non-nil) slice", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) { return &config.Config{}, nil })
		got := a.GetActionGroups()
		if got == nil || len(got) != 0 {
			t.Errorf("GetActionGroups() = %#v, want a non-nil empty slice", got)
		}
	})
}

func TestReloadConfig(t *testing.T) {
	t.Run("total failure keeps the previous config and returns the error", func(t *testing.T) {
		calls := 0
		a := NewApp(func() (*config.Config, error) {
			calls++
			if calls == 1 {
				return &config.Config{SourcePath: "/ok.yaml", Shell: []string{"bash"}}, nil
			}
			return &config.Config{}, errors.New("boom")
		})
		warning, err := a.ReloadConfig()
		if err == nil {
			t.Fatal("expected an error")
		}
		if warning != "" {
			t.Errorf("warning = %q, want empty", warning)
		}
		if a.cfg.SourcePath != "/ok.yaml" {
			t.Errorf("previous config should be kept, got SourcePath %q", a.cfg.SourcePath)
		}
	})
	t.Run("fallback success surfaces a warning instead of an error", func(t *testing.T) {
		a := NewApp(func() (*config.Config, error) {
			return &config.Config{SourcePath: "/fallback.yaml"}, errors.New("config-win.yaml: boom")
		})
		warning, err := a.ReloadConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if warning == "" {
			t.Error("expected a non-empty warning")
		}
		if a.cfg.SourcePath != "/fallback.yaml" {
			t.Errorf("SourcePath = %q, want /fallback.yaml", a.cfg.SourcePath)
		}
	})
}
