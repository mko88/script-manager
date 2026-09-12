package configedit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"script-manager/internal/config"
	"script-manager/internal/secret"

	"gopkg.in/yaml.v3"
)

func looksLikeSecretKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.HasSuffix(lower, "secret") || strings.HasSuffix(lower, "password") || strings.HasSuffix(lower, "key")
}

func classifyValue(key string, v any) (kind, value string, secret bool) {
	secret = looksLikeSecretKey(key)
	switch t := v.(type) {
	case nil:
		return "yaml", "null", secret
	case string:
		if strings.Contains(t, "\n") {
			return "multiline", t, secret
		}
		return "string", t, secret
	case bool:
		return "bool", strconv.FormatBool(t), secret
	case int:
		return "number", strconv.FormatInt(int64(t), 10), secret
	case int64:
		return "number", strconv.FormatInt(t, 10), secret
	case uint64:
		return "number", strconv.FormatUint(t, 10), secret
	case float64:
		return "number", strconv.FormatFloat(t, 'g', -1, 64), secret
	default:
		out, err := yaml.Marshal(v)
		if err != nil {
			return "yaml", fmt.Sprintf("%v", v), secret
		}
		return "yaml", strings.TrimRight(string(out), "\n"), secret
	}
}

func decodeValue(kind, value string) (any, error) {
	switch kind {
	case "string", "multiline":
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
		kind, value, isSecret := classifyValue(k, m[k])
		locked := secret.IsLocked(value)
		fields = append(fields, FieldDTO{Key: k, Kind: kind, Value: value, Secret: isSecret || locked, Locked: locked})
	}
	return fields
}

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
		Script:      a.Script,
		Groups:      nonNil(append([]string(nil), a.Groups...)),
		NoWait:      a.NoWait,
		Interactive: a.Interactive,
		RequiresPIN: a.RequiresPIN,
	}
}

func actionFromDTO(dto ActionDTO) config.Action {
	return config.Action{
		ID:          dto.ID,
		Title:       dto.Title,
		Description: dto.Description,
		Cmd:         dto.Cmd,
		Script:      dto.Script,
		Groups:      append([]string(nil), dto.Groups...),
		NoWait:      dto.NoWait,
		Interactive: dto.Interactive,
		RequiresPIN: dto.RequiresPIN,
	}
}

func actionGroupToDTO(g config.ActionGroup) ActionGroupDTO {
	return ActionGroupDTO{ID: g.ID, Title: g.Title, Color: g.Color}
}

func actionGroupFromDTO(dto ActionGroupDTO) config.ActionGroup {
	return config.ActionGroup{ID: dto.ID, Title: dto.Title, Color: dto.Color}
}

func actionDTOToMap(a ActionDTO) map[string]any {
	m := map[string]any{"title": a.Title, "cmd": a.Cmd}
	if a.Script != "" {
		m["script"] = a.Script
	}
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
	if a.Interactive {
		m["interactive"] = true
	}
	if a.RequiresPIN {
		m["requiresPin"] = true
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

func ToConfigDTO(cfg *config.Config) ConfigDTO {
	dto := ConfigDTO{
		Secrets:      secretsToDTO(cfg.Secrets),
		Shell:        nonNil(append([]string(nil), cfg.Shell...)),
		Terminal:     terminalToDTO(cfg.Terminal),
		EnvFields:    FieldsFromMap(cfg.Env, nil),
		Display:      []DisplayDTO{},
		ActionGroups: []ActionGroupDTO{},
		Actions:      []ActionDTO{},
		Items:        []ItemDTO{},
	}
	for _, d := range cfg.Display {
		dto.Display = append(dto.Display, DisplayDTO{Name: d.Name, List: d.List, Details: d.Details})
	}
	for _, g := range cfg.ActionGroups {
		dto.ActionGroups = append(dto.ActionGroups, actionGroupToDTO(g))
	}
	for _, a := range cfg.Actions {
		dto.Actions = append(dto.Actions, actionToDTO(a))
	}
	for _, item := range cfg.Items {
		dto.Items = append(dto.Items, ToItemDTO(item))
	}
	return dto
}

func FromConfigDTO(dto ConfigDTO) (*config.Config, error) {
	cfg := &config.Config{
		Secrets:  secretsFromDTO(dto.Secrets),
		Shell:    append([]string(nil), dto.Shell...),
		Terminal: terminalFromDTO(dto.Terminal),
	}
	for _, d := range dto.Display {
		cfg.Display = append(cfg.Display, config.DisplayConfig{Name: d.Name, List: d.List, Details: d.Details})
	}
	for _, g := range dto.ActionGroups {
		cfg.ActionGroups = append(cfg.ActionGroups, actionGroupFromDTO(g))
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
