// Package configedit is the Wails-bound backend for sm-config-edit, the
// structured config.yaml editor. It mirrors internal/gui's shape (an App
// struct whose exported methods become the frontend's bindings) but reads
// and writes config.Config instead of just reading it.
package configedit

// TitlesDTO mirrors config.TitlesConfig.
type TitlesDTO struct {
	Items   string `json:"items"`
	Actions string `json:"actions"`
	Details string `json:"details"`
	Command string `json:"command"`
}

// DisplayDTO mirrors config.DisplayConfig.
type DisplayDTO struct {
	Name    string `json:"name"`
	List    string `json:"list"`
	Details string `json:"details"`
}

// TerminalDTO flattens config.TerminalConfig's scalar-or-list ambiguity into
// an explicit mode a radio group can drive directly.
type TerminalDTO struct {
	Mode string   `json:"mode"` // "auto" | "name" | "argv"
	Name string   `json:"name"`
	Argv []string `json:"argv"`
}

// ActionDTO mirrors config.Action. It's used both for the global actions
// list and for an item's customActions, since both are []Action-shaped.
type ActionDTO struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Cmd         string   `json:"cmd"`
	Groups      []string `json:"groups"`
	NoWait      bool     `json:"noWait"`
}

// ActionGroupDTO mirrors config.ActionGroup — the catalog entry. Actions and
// items still just reference a group by plain string ID (via Groups/
// actionGroups), unchanged by this addition.
type ActionGroupDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
}

// FieldDTO edits one entry of a map[string]any (Env, or an item's
// non-reserved keys) without needing a widget per possible YAML shape.
// String/bool/number values get their own kind and a plain-value widget;
// anything else (nested map/list, or a value of an unrecognized type)
// becomes Kind "yaml", edited as a raw YAML snippet in Value and re-parsed on
// save — the escape hatch that keeps this scheme minimal. "multiline" is
// still a plain string underneath (same as "string" once saved) — it only
// picks a different edit widget, a textarea for an existing value with
// embedded newlines.
//
// Secret is independent of Kind — a lock toggle in the UI, not a kind of its
// own — so any field, including a multi-line one, can be edited masked. Like
// Kind it's a display hint only: it never round-trips through the saved
// YAML, and is re-derived fresh from the key's name (see looksLikeSecretKey)
// every time the field is classified, defaulting a key that looks like a
// secret to masked without the user having to notice and toggle it.
type FieldDTO struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"` // "string" | "multiline" | "number" | "bool" | "yaml"
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

// ItemDTO mirrors one entry of config.Items: the five reserved keys get
// dedicated fields; everything else is a generic Fields grid.
type ItemDTO struct {
	Name          string      `json:"name"`
	Display       string      `json:"display"`
	Actions       []string    `json:"actions"`
	ActionGroups  []string    `json:"actionGroups"`
	CustomActions []ActionDTO `json:"customActions"`
	Fields        []FieldDTO  `json:"fields"`
}

// ConfigDTO mirrors config.Config as a whole.
type ConfigDTO struct {
	Shell        []string         `json:"shell"`
	Display      []DisplayDTO     `json:"display"`
	Titles       TitlesDTO        `json:"titles"`
	Terminal     TerminalDTO      `json:"terminal"`
	EnvFields    []FieldDTO       `json:"envFields"`
	Items        []ItemDTO        `json:"items"`
	ActionGroups []ActionGroupDTO `json:"actionGroups"`
	Actions      []ActionDTO      `json:"actions"`
}

// StateDTO is what the frontend fetches after any operation that (re)loads a
// config: InitialState, NewBlank, BrowseOpen.
type StateDTO struct {
	Config  ConfigDTO `json:"config"`
	Path    string    `json:"path"`
	Warning string    `json:"warning"`
}

// SaveResultDTO is Save's result: the path actually written to, so the
// frontend can update its "current file" display after a Save-As.
type SaveResultDTO struct {
	Path string `json:"path"`
}

// ValidationIssueDTO is one finding from ValidateConfig.
type ValidationIssueDTO struct {
	Severity string `json:"severity"` // "error" | "warning"
	Message  string `json:"message"`
}

// PreviewDTO is an item's rendered list label + details HTML, built from
// draft (not-yet-saved) form state.
type PreviewDTO struct {
	ListLabel     string   `json:"listLabel"`
	DetailsHTML   string   `json:"detailsHtml"`
	MissingFields []string `json:"missingFields"`
	Error         string   `json:"error"`
}

// ActionPreviewDTO is an action's rendered description/command, built from
// draft form state.
type ActionPreviewDTO struct {
	Description string `json:"description"`
	Cmd         string `json:"cmd"`
	Error       string `json:"error"`
}
