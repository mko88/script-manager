package configedit

type DisplayDTO struct {
	Name    string `json:"name"`
	List    string `json:"list"`
	Details string `json:"details"`
}

type TerminalDTO struct {
	Mode string   `json:"mode"`
	Name string   `json:"name"`
	Argv []string `json:"argv"`
}

type ActionDTO struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Cmd         string   `json:"cmd"`
	Script      string   `json:"script"`
	Groups      []string `json:"groups"`
	NoWait      bool     `json:"noWait"`
	Interactive bool     `json:"interactive"`
}

type ActionGroupDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
}

type FieldDTO struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type ItemDTO struct {
	Name          string      `json:"name"`
	Display       string      `json:"display"`
	Actions       []string    `json:"actions"`
	ActionGroups  []string    `json:"actionGroups"`
	CustomActions []ActionDTO `json:"customActions"`
	Fields        []FieldDTO  `json:"fields"`
}

type ConfigDTO struct {
	Shell        []string         `json:"shell"`
	Display      []DisplayDTO     `json:"display"`
	Terminal     TerminalDTO      `json:"terminal"`
	EnvFields    []FieldDTO       `json:"envFields"`
	Items        []ItemDTO        `json:"items"`
	ActionGroups []ActionGroupDTO `json:"actionGroups"`
	Actions      []ActionDTO      `json:"actions"`
}

type StateDTO struct {
	Config  ConfigDTO `json:"config"`
	Path    string    `json:"path"`
	Warning string    `json:"warning"`
}

type SaveResultDTO struct {
	Path string `json:"path"`
}

type ValidationIssueDTO struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type PreviewDTO struct {
	ListLabel     string   `json:"listLabel"`
	DetailsHTML   string   `json:"detailsHtml"`
	MissingFields []string `json:"missingFields"`
	Error         string   `json:"error"`
}

type ActionPreviewDTO struct {
	Description string `json:"description"`
	Cmd         string `json:"cmd"`
	Script      string `json:"script"`
	Error       string `json:"error"`
}
