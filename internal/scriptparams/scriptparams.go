// Package scriptparams reads a script's parameters out of its own
// documentation: PowerShell comment-based help and param() blocks, Python
// argparse calls, and the usage/options comment headers of shell and batch
// scripts. The script is only ever read as text, never run.
package scriptparams

import (
	"path/filepath"
	"strings"
)

type Kind string

const (
	Flag        Kind = "flag"
	Value       Kind = "value"
	Choice      Kind = "choice"
	MultiChoice Kind = "multiChoice"
	MultiValue  Kind = "multiValue"
)

type Language string

const (
	PowerShell Language = "powershell"
	Python     Language = "python"
	Shell      Language = "shell"
	Batch      Language = "batch"
	Unknown    Language = ""
)

type Param struct {
	Name string `json:"name"`
	// Flag is the spelling emitted before the value: -Name, --name, -n, /n.
	// Empty for a positional parameter.
	Flag    string   `json:"flag"`
	Kind    Kind     `json:"kind"`
	Choices []string `json:"choices"`
	// Default is the script's own default, for display only: an empty
	// control leaves the parameter out and the script applies it.
	Default     string `json:"default"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	// Repeat emits the flag before each value of a multi-valued parameter
	// instead of once before all of them.
	Repeat bool `json:"repeat"`
}

func (p Param) Multi() bool { return p.Kind == MultiChoice || p.Kind == MultiValue }

func (p Param) Positional() bool { return p.Flag == "" }

type Spec struct {
	Language   Language `json:"language"`
	Documented bool     `json:"documented"`
	Synopsis   string   `json:"synopsis"`
	Params     []Param  `json:"params"`
}

const bom = "\xEF\xBB\xBF"

// Parse reads the parameters documented in text, a script stored at path.
// The path only decides the language; nothing is read from disk.
func Parse(path, text string) Spec {
	text = strings.ReplaceAll(strings.TrimPrefix(text, bom), "\r\n", "\n")
	lang := Detect(path, text)
	var spec Spec
	switch lang {
	case PowerShell:
		spec = parsePowerShell(text)
	case Python:
		spec = parsePython(text)
	case Shell:
		spec = parseShell(text)
	case Batch:
		spec = parseBatch(text)
	}
	spec.Language = lang
	spec.Params = dedupe(spec.Params)
	spec.Documented = len(spec.Params) > 0
	return spec
}

// Detect names a script's language from its extension, or its shebang
// when the extension says nothing.
func Detect(path, text string) Language {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ps1", ".psm1":
		return PowerShell
	case ".py", ".pyw":
		return Python
	case ".sh", ".bash", ".zsh", ".ksh", ".dash":
		return Shell
	case ".bat", ".cmd":
		return Batch
	}
	first, _, _ := strings.Cut(text, "\n")
	if !strings.HasPrefix(first, "#!") {
		return Unknown
	}
	interp := strings.Fields(first[2:])
	if len(interp) == 0 {
		return Unknown
	}
	name := filepath.Base(interp[0])
	if name == "env" {
		for _, f := range interp[1:] {
			if !strings.HasPrefix(f, "-") {
				name = filepath.Base(f)
				break
			}
		}
	}
	switch {
	case strings.HasPrefix(name, "pwsh"), strings.HasPrefix(name, "powershell"):
		return PowerShell
	case strings.HasPrefix(name, "python"):
		return Python
	case name == "sh", name == "bash", name == "zsh", name == "ksh", name == "dash", name == "ash":
		return Shell
	}
	return Unknown
}

func dedupe(params []Param) []Param {
	seen := make(map[string]bool, len(params))
	out := params[:0]
	for _, p := range params {
		if p.Name == "" || seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
