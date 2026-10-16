package gui

import (
	"errors"
	"fmt"

	"script-manager/internal/action"
	"script-manager/internal/config"
	"script-manager/internal/presets"
	"script-manager/internal/scriptparams"
	"script-manager/internal/scriptsource"
)

// RunInputDTO is what the Parameters panel holds: values for the script's
// documented parameters and free-text arguments.
type RunInputDTO struct {
	Values map[string]any `json:"values"`
	Args   string         `json:"args"`
}

type CommandPreviewDTO struct {
	Line  string `json:"line"`
	Error string `json:"error"`
}

type PresetsDTO struct {
	Presets []presets.Preset `json:"presets"`
	Warning string           `json:"warning"`
}

func (a *App) actionAt(itemIndex, actionIndex int) (*config.Item, config.Action, presets.Key, error) {
	item := a.itemAt(itemIndex)
	if item == nil {
		return nil, config.Action{}, presets.Key{}, fmt.Errorf("invalid item")
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return nil, config.Action{}, presets.Key{}, fmt.Errorf("invalid action")
	}
	act := actions[actionIndex]
	key := presets.Key{Action: act.ID}
	if key.Action == "" {
		key.Action = act.Title
	}
	if actionIndex >= len(actions)-len(item.CustomActions) {
		key.Item = item.Name
		key.Custom = true
	}
	return item, act, key, nil
}

func scriptSpec(path string) scriptparams.Spec {
	if path == "" {
		return scriptparams.Spec{}
	}
	text, err := scriptsource.Read(path)
	if err != nil {
		return scriptparams.Spec{}
	}
	return scriptparams.Parse(path, text)
}

func (a *App) shellBase() string {
	if len(a.cfg.Shell) == 0 {
		return ""
	}
	return action.ShellBasename(a.cfg.Shell[0])
}

// commandArgs renders input as argument text for the action's stub, with
// templates in the values expanded against merged. Values only apply to a
// script whose parameters are documented.
func (a *App) commandArgs(scriptPath string, merged map[string]any, input RunInputDTO) (string, error) {
	expand := func(s string) (string, error) { return action.Expand(s, merged) }
	extra, err := expand(input.Args)
	if err != nil {
		return "", fmt.Errorf("arguments template error: %w", err)
	}
	rendered := ""
	if spec := scriptSpec(scriptPath); spec.Documented {
		values, err := scriptparams.ExpandValues(input.Values, expand)
		if err != nil {
			return "", fmt.Errorf("parameter template error: %w", err)
		}
		rendered, err = scriptparams.Args(a.shellBase(), spec, values)
		if err != nil {
			return "", err
		}
	}
	return scriptparams.Join(rendered, extra)
}

// PreviewCommand is the line Run would execute with input, with secrets
// masked.
func (a *App) PreviewCommand(itemIndex, actionIndex int, input RunInputDTO) CommandPreviewDTO {
	item, act, _, err := a.actionAt(itemIndex, actionIndex)
	if err != nil {
		return CommandPreviewDTO{Error: err.Error()}
	}
	if len(a.cfg.Shell) == 0 {
		return CommandPreviewDTO{Error: "no shell configured"}
	}
	merged := a.mergedItem(item)
	if act.Script != "" {
		script, err := action.Expand(act.Script, merged)
		if err != nil {
			return CommandPreviewDTO{Error: fmt.Sprintf("script path template error: %v", err)}
		}
		args, err := a.commandArgs(script, merged, input)
		if err != nil {
			return CommandPreviewDTO{Line: action.ScriptInvocation(a.shellBase(), script, ""), Error: err.Error()}
		}
		return CommandPreviewDTO{Line: action.ScriptInvocation(a.shellBase(), script, args)}
	}
	cmd := action.Preview(act.Cmd, merged)
	args, err := a.commandArgs("", merged, input)
	if err != nil {
		return CommandPreviewDTO{Line: cmd, Error: err.Error()}
	}
	return CommandPreviewDTO{Line: action.AppendArgs(cmd, args)}
}

func (a *App) presetsPath() (string, error) {
	src := a.configSourcePath()
	if src == "" {
		return "", errors.New("no config file is loaded")
	}
	return presets.PathFor(src), nil
}

func (a *App) GetPresets(itemIndex, actionIndex int) PresetsDTO {
	_, _, key, err := a.actionAt(itemIndex, actionIndex)
	if err != nil {
		return PresetsDTO{}
	}
	path, err := a.presetsPath()
	if err != nil {
		return PresetsDTO{}
	}
	f, err := presets.Load(path)
	dto := PresetsDTO{Presets: f.List(key)}
	if err != nil {
		dto.Warning = err.Error()
	}
	return dto
}

func (a *App) SavePreset(itemIndex, actionIndex int, preset presets.Preset) (PresetsDTO, error) {
	return a.editPresets(itemIndex, actionIndex, func(f *presets.File, key presets.Key) error {
		return f.Put(key, preset)
	})
}

func (a *App) DeletePreset(itemIndex, actionIndex int, name string) (PresetsDTO, error) {
	return a.editPresets(itemIndex, actionIndex, func(f *presets.File, key presets.Key) error {
		f.Delete(key, name)
		return nil
	})
}

// editPresets refuses to write over a presets file that doesn't parse, so
// a typo made by hand doesn't cost every other preset in it.
func (a *App) editPresets(itemIndex, actionIndex int, edit func(*presets.File, presets.Key) error) (PresetsDTO, error) {
	_, _, key, err := a.actionAt(itemIndex, actionIndex)
	if err != nil {
		return PresetsDTO{}, err
	}
	path, err := a.presetsPath()
	if err != nil {
		return PresetsDTO{}, err
	}
	f, err := presets.Load(path)
	if err != nil {
		return PresetsDTO{}, fmt.Errorf("fix or remove the presets file first: %w", err)
	}
	if err := edit(f, key); err != nil {
		return PresetsDTO{}, err
	}
	if err := presets.Save(path, f); err != nil {
		return PresetsDTO{}, err
	}
	return PresetsDTO{Presets: f.List(key)}, nil
}
