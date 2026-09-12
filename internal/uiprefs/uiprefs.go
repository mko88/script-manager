package uiprefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const Filename = "sm-ui.json"

const (
	ScaleMin     = 70
	ScaleMax     = 200
	ScaleStep    = 10
	ScaleDefault = 100
)

type Prefs struct {
	FontUI       string `json:"fontUi"`
	FontMono     string `json:"fontMono"`
	ScalePercent int    `json:"scalePercent"`
}

func Defaults() Prefs {
	return Prefs{ScalePercent: ScaleDefault}
}

func Load(dir string) Prefs {
	if dir == "" {
		return Defaults()
	}
	data, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		return Defaults()
	}
	var p Prefs
	if err := json.Unmarshal(data, &p); err != nil {
		return Defaults()
	}
	return normalize(p)
}

func Save(dir string, p Prefs) (Prefs, error) {
	p = normalize(p)
	if dir == "" {
		return p, nil
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return p, err
	}
	return p, os.WriteFile(filepath.Join(dir, Filename), data, 0o644)
}

func normalize(p Prefs) Prefs {
	p.FontUI = strings.TrimSpace(p.FontUI)
	p.FontMono = strings.TrimSpace(p.FontMono)
	switch {
	case p.ScalePercent == 0:
		p.ScalePercent = ScaleDefault
	case p.ScalePercent < ScaleMin:
		p.ScalePercent = ScaleMin
	case p.ScalePercent > ScaleMax:
		p.ScalePercent = ScaleMax
	}
	return p
}
