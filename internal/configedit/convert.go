package configedit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"script-manager/internal/config"

	"gopkg.in/yaml.v3"
)

// classifyValue picks the FieldDTO kind/value pair for an existing
// map[string]any value, decoded moments earlier by yaml.v3. Anything that
// isn't a plain string/bool/number falls back to a YAML snippet, which is
// the same shape decodeValue's "yaml" case parses back.
func classifyValue(v any) (kind, value string) {
	switch t := v.(type) {
	case nil:
		return "yaml", "null"
	case string:
		return "string", t
	case bool:
		return "bool", strconv.FormatBool(t)
	case int:
		return "number", strconv.FormatInt(int64(t), 10)
	case int64:
		return "number", strconv.FormatInt(t, 10)
	case uint64:
		return "number", strconv.FormatUint(t, 10)
	case float64:
		return "number", strconv.FormatFloat(t, 'g', -1, 64)
	default:
		out, err := yaml.Marshal(v)
		if err != nil {
			return "yaml", fmt.Sprintf("%v", v)
		}
		return "yaml", strings.TrimRight(string(out), "\n")
	}
}

// decodeValue is classifyValue's inverse. Number decoding tries ParseInt
// before ParseFloat so an integer like 42 doesn't round-trip back out
// reformatted through a float path (42.0, exponential notation, ...).
func decodeValue(kind, value string) (any, error) {
	switch kind {
	case "string":
		return value, nil
	case "bool":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid bool %q", value)
		}
		return b, nil
	case "number":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i, nil
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", value)
		}
		return f, nil
	case "yaml":
		var parsed any
		if err := yaml.Unmarshal([]byte(value), &parsed); err != nil {
			return nil, fmt.Errorf("invalid yaml: %w", err)
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("unknown field kind %q", kind)
	}
}

// FieldsFromMap converts every key of m not in exclude into a FieldDTO,
// sorted alphabetically by key — map[string]any has no stable order to begin
// with, and yaml.v3 sorts map keys on marshal anyway, so alphabetical is
// simply the one true order end-to-end.
func FieldsFromMap(m map[string]any, exclude map[string]bool) []FieldDTO {
	keys := make([]string, 0, len(m))
	for k := range m {
		if exclude[k] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fields := make([]FieldDTO, 0, len(keys))
	for _, k := range keys {
		kind, value := classifyValue(m[k])
		fields = append(fields, FieldDTO{Key: k, Kind: kind, Value: value})
	}
	return fields
}

// FieldsToMap decodes a []FieldDTO back into a map[string]any. A field with
// an empty key is skipped (the frontend's "add field" row before a key is
// typed).
func FieldsToMap(fields []FieldDTO) (map[string]any, error) {
	out := make(map[string]any, len(fields))
	for _, f := range fields {
		if f.Key == "" {
			continue
		}
		v, err := decodeValue(f.Kind, f.Value)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", f.Key, err)
		}
		out[f.Key] = v
	}
	return out, nil
}

// nonNil returns s, or a non-nil empty slice if s is nil. Go's JSON encoder
// marshals a nil slice as `null` rather than `[]`; the frontend treats every
// DTO slice field as always-iterable (Svelte {#each}, .map, .some, .includes),
// so a nil slice reaching the frontend throws — and since that throw happens
// inside a Svelte reactive statement, it silently breaks all further
// reactivity for the rest of the session, not just the one expression. Every
// slice field exposed on a DTO must go through this before being returned.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func actionToDTO(a config.Action) ActionDTO {
	return ActionDTO{
		ID:          a.ID,
		Title:       a.Title,
		Description: a.Description,
		Cmd:         a.Cmd,
		Groups:      nonNil(append([]string(nil), a.Groups...)),
		NoWait:      a.NoWait,
	}
}

func actionFromDTO(dto ActionDTO) config.Action {
	return config.Action{
		ID:          dto.ID,
		Title:       dto.Title,
		Description: dto.Description,
		Cmd:         dto.Cmd,
		Groups:      append([]string(nil), dto.Groups...),
		NoWait:      dto.NoWait,
	}
}

// actionDTOToMap encodes an ActionDTO the same way a hand-written
// customActions entry is shaped, for round-tripping through
// config.ParseCustomActions.
func actionDTOToMap(a ActionDTO) map[string]any {
	m := map[string]any{"title": a.Title, "cmd": a.Cmd}
	if a.ID != "" {
		m["id"] = a.ID
	}
	if a.Description != "" {
		m["description"] = a.Description
	}
	if len(a.Groups) > 0 {
		m["groups"] = toAnySlice(a.Groups)
	}
	if a.NoWait {
		m["noWait"] = true
	}
	return m
}

var reservedItemKeys = map[string]bool{
	config.KeyName:          true,
	config.KeyDisplay:       true,
	config.KeyActions:       true,
	config.KeyActionGroups:  true,
	config.KeyCustomActions: true,
}

// ToItemDTO decodes one config.Items entry, reusing config.AsStringSlice/
// config.ParseCustomActions — the exact same logic config.ActionsForItem
// relies on at runtime — so the editor's interpretation of a reserved key
// can never drift from how it's actually consumed.
func ToItemDTO(item map[string]any) ItemDTO {
	dto := ItemDTO{
		Name:          config.StrVal(item[config.KeyName]),
		Display:       config.StrVal(item[config.KeyDisplay]),
		Actions:       []string{},
		ActionGroups:  []string{},
		CustomActions: []ActionDTO{},
	}
	if ids, ok := config.AsStringSlice(item[config.KeyActions]); ok {
		dto.Actions = ids
	}
	if groups, ok := config.AsStringSlice(item[config.KeyActionGroups]); ok {
		dto.ActionGroups = groups
	}
	for _, a := range config.ParseCustomActions(item[config.KeyCustomActions]) {
		dto.CustomActions = append(dto.CustomActions, actionToDTO(a))
	}
	dto.Fields = FieldsFromMap(item, reservedItemKeys)
	return dto
}

// FromItemDTO is ToItemDTO's inverse: reserved keys are omitted entirely
// when empty, matching the hand-written style config.yaml examples use.
func FromItemDTO(dto ItemDTO) (map[string]any, error) {
	item := make(map[string]any)
	if dto.Name != "" {
		item[config.KeyName] = dto.Name
	}
	if dto.Display != "" {
		item[config.KeyDisplay] = dto.Display
	}
	if len(dto.Actions) > 0 {
		item[config.KeyActions] = toAnySlice(dto.Actions)
	}
	if len(dto.ActionGroups) > 0 {
		item[config.KeyActionGroups] = toAnySlice(dto.ActionGroups)
	}
	if len(dto.CustomActions) > 0 {
		custom := make([]any, len(dto.CustomActions))
		for i, a := range dto.CustomActions {
			custom[i] = actionDTOToMap(a)
		}
		item[config.KeyCustomActions] = custom
	}
	extra, err := FieldsToMap(dto.Fields)
	if err != nil {
		name := dto.Name
		if name == "" {
			name = "(unnamed)"
		}
		return nil, fmt.Errorf("item %q: %w", name, err)
	}
	for k, v := range extra {
		item[k] = v
	}
	return item, nil
}

func terminalToDTO(t config.TerminalConfig) TerminalDTO {
	switch {
	case len(t.Argv) > 0:
		return TerminalDTO{Mode: "argv", Argv: append([]string(nil), t.Argv...)}
	case t.Name != "":
		return TerminalDTO{Mode: "name", Name: t.Name, Argv: []string{}}
	default:
		return TerminalDTO{Mode: "auto", Argv: []string{}}
	}
}

func terminalFromDTO(dto TerminalDTO) config.TerminalConfig {
	switch dto.Mode {
	case "name":
		return config.TerminalConfig{Name: dto.Name}
	case "argv":
		return config.TerminalConfig{Argv: dto.Argv}
	default:
		return config.TerminalConfig{}
	}
}

// ToConfigDTO converts a whole loaded config for editing.
func ToConfigDTO(cfg *config.Config) ConfigDTO {
	dto := ConfigDTO{
		Shell: nonNil(append([]string(nil), cfg.Shell...)),
		Titles: TitlesDTO{
			Items:   cfg.Titles.Items,
			Actions: cfg.Titles.Actions,
			Details: cfg.Titles.Details,
			Command: cfg.Titles.Command,
		},
		Terminal:  terminalToDTO(cfg.Terminal),
		EnvFields: FieldsFromMap(cfg.Env, nil),
		Display:   []DisplayDTO{},
		Actions:   []ActionDTO{},
		Items:     []ItemDTO{},
	}
	for _, d := range cfg.Display {
		dto.Display = append(dto.Display, DisplayDTO{Name: d.Name, List: d.List, Details: d.Details})
	}
	for _, a := range cfg.Actions {
		dto.Actions = append(dto.Actions, actionToDTO(a))
	}
	for _, item := range cfg.Items {
		dto.Items = append(dto.Items, ToItemDTO(item))
	}
	return dto
}

// FromConfigDTO is ToConfigDTO's inverse, used by Save.
func FromConfigDTO(dto ConfigDTO) (*config.Config, error) {
	cfg := &config.Config{
		Shell: append([]string(nil), dto.Shell...),
		Titles: config.TitlesConfig{
			Items:   dto.Titles.Items,
			Actions: dto.Titles.Actions,
			Details: dto.Titles.Details,
			Command: dto.Titles.Command,
		},
		Terminal: terminalFromDTO(dto.Terminal),
	}
	for _, d := range dto.Display {
		cfg.Display = append(cfg.Display, config.DisplayConfig{Name: d.Name, List: d.List, Details: d.Details})
	}
	for _, a := range dto.Actions {
		cfg.Actions = append(cfg.Actions, actionFromDTO(a))
	}
	env, err := FieldsToMap(dto.EnvFields)
	if err != nil {
		return nil, fmt.Errorf("env: %w", err)
	}
	if len(env) > 0 {
		cfg.Env = env
	}
	for _, itemDTO := range dto.Items {
		item, err := FromItemDTO(itemDTO)
		if err != nil {
			return nil, err
		}
		cfg.Items = append(cfg.Items, item)
	}
	return cfg, nil
}
