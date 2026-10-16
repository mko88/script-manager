package scriptparams

import (
	"regexp"
	"strings"
)

var (
	psHelpKeyword  = regexp.MustCompile(`^\.([A-Za-z]+)(?:\s+(.*))?$`)
	psMandatory    = regexp.MustCompile(`(?i)\bMandatory\b(?:\s*=\s*\$?(\w+))?`)
	psHelpMessage  = regexp.MustCompile(`(?i)\bHelpMessage\s*=\s*('(?:[^']|'')*'|"(?:[^"` + "`" + `]|` + "`" + `.)*")`)
	psStringLit    = regexp.MustCompile(`'((?:[^']|'')*)'|"((?:[^"` + "`" + `]|` + "`" + `.)*)"`)
	psBlockComment = regexp.MustCompile(`(?s)<#(.*?)#>`)
	psParamStart   = regexp.MustCompile(`(?i)^param\s*\(`)
	psVariable     = regexp.MustCompile(`^\$(?:\{([^}]+)\}|([A-Za-z_][A-Za-z0-9_]*))`)
)

type psHelp struct {
	synopsis string
	params   map[string]string // lower-cased name → description
	order    []string
}

func parsePowerShell(text string) Spec {
	help := psFindHelp(text)
	var params []Param
	if body, ok := psParamBlock(text); ok {
		for _, entry := range psSplitTopLevel(body) {
			if p, ok := psParseEntry(entry); ok {
				if d, ok := help.params[strings.ToLower(p.Name)]; ok && d != "" {
					p.Description = d
				}
				params = append(params, p)
			}
		}
	} else {
		for _, name := range help.order {
			params = append(params, Param{Name: name, Flag: "-" + name, Kind: Value, Description: help.params[strings.ToLower(name)]})
		}
	}
	return Spec{Synopsis: help.synopsis, Params: params}
}

// psFindHelp reads the first comment block that holds help keywords, in
// either the <# #> form or a run of # lines.
func psFindHelp(text string) psHelp {
	var blocks []string
	for _, m := range psBlockComment.FindAllStringSubmatch(text, -1) {
		blocks = append(blocks, m[1])
	}
	var run []string
	flush := func() {
		if len(run) > 0 {
			blocks = append(blocks, strings.Join(run, "\n"))
			run = nil
		}
	}
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "#>") && !strings.HasPrefix(t, "<#") {
			run = append(run, strings.TrimPrefix(t, "#"))
			continue
		}
		flush()
	}
	flush()

	for _, b := range blocks {
		if h, ok := psParseHelp(b); ok {
			return h
		}
	}
	return psHelp{params: map[string]string{}}
}

func psParseHelp(block string) (psHelp, bool) {
	h := psHelp{params: map[string]string{}}
	found := false
	section, arg := "", ""
	var buf []string
	commit := func() {
		body := strings.TrimSpace(strings.Join(buf, "\n"))
		switch section {
		case "SYNOPSIS":
			h.synopsis = collapseSpace(body)
		case "PARAMETER":
			if arg != "" {
				key := strings.ToLower(arg)
				if _, dup := h.params[key]; !dup {
					h.order = append(h.order, arg)
				}
				h.params[key] = collapseSpace(body)
			}
		}
		buf = nil
	}
	for _, line := range strings.Split(block, "\n") {
		t := strings.TrimSpace(line)
		if m := psHelpKeyword.FindStringSubmatch(t); m != nil && psKnownHelpKeyword(m[1]) {
			commit()
			section, arg = strings.ToUpper(m[1]), strings.TrimSpace(m[2])
			found = true
			continue
		}
		buf = append(buf, t)
	}
	commit()
	return h, found
}

func psKnownHelpKeyword(k string) bool {
	switch strings.ToUpper(k) {
	case "SYNOPSIS", "DESCRIPTION", "PARAMETER", "EXAMPLE", "INPUTS", "OUTPUTS", "NOTES", "LINK",
		"COMPONENT", "ROLE", "FUNCTIONALITY", "FORWARDHELPTARGETNAME", "FORWARDHELPCATEGORY",
		"REMOTEHELPRUNSPACE", "EXTERNALHELP":
		return true
	}
	return false
}

// psParamBlock finds the script's own param( ... ), which can only follow
// comments, #requires/using lines and attributes such as [CmdletBinding()].
func psParamBlock(text string) (string, bool) {
	i := 0
	for i < len(text) {
		switch c := text[i]; {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case strings.HasPrefix(text[i:], "<#"):
			end := strings.Index(text[i+2:], "#>")
			if end < 0 {
				return "", false
			}
			i += 2 + end + 2
		case c == '#':
			i = lineEnd(text, i)
		case c == '[':
			end, ok := psMatch(text, i)
			if !ok {
				return "", false
			}
			i = end + 1
		case hasWordPrefix(text[i:], "using"):
			i = lineEnd(text, i)
		default:
			if !psParamStart.MatchString(text[i:]) {
				return "", false
			}
			open := i + strings.IndexByte(text[i:], '(')
			end, ok := psMatch(text, open)
			if !ok {
				return "", false
			}
			return text[open+1 : end], true
		}
	}
	return "", false
}

func hasWordPrefix(s, word string) bool {
	if len(s) <= len(word) || !strings.EqualFold(s[:len(word)], word) {
		return false
	}
	c := s[len(word)]
	return c == ' ' || c == '\t'
}

func lineEnd(text string, i int) int {
	if n := strings.IndexByte(text[i:], '\n'); n >= 0 {
		return i + n + 1
	}
	return len(text)
}

// psSkip returns the index just past a string or comment starting at i,
// or i itself when there is none.
func psSkip(text string, i int) int {
	switch {
	case strings.HasPrefix(text[i:], "@'") || strings.HasPrefix(text[i:], `@"`):
		closer := "\n" + string(text[i+1]) + "@"
		if n := strings.Index(text[i+2:], closer); n >= 0 {
			return i + 2 + n + len(closer)
		}
		return len(text)
	case strings.HasPrefix(text[i:], "<#"):
		if n := strings.Index(text[i+2:], "#>"); n >= 0 {
			return i + 2 + n + 2
		}
		return len(text)
	case text[i] == '#':
		return lineEnd(text, i)
	case text[i] == '\'':
		for j := i + 1; j < len(text); j++ {
			if text[j] == '\'' {
				if j+1 < len(text) && text[j+1] == '\'' {
					j++
					continue
				}
				return j + 1
			}
		}
		return len(text)
	case text[i] == '"':
		for j := i + 1; j < len(text); j++ {
			switch text[j] {
			case '`':
				j++
			case '"':
				if j+1 < len(text) && text[j+1] == '"' {
					j++
					continue
				}
				return j + 1
			}
		}
		return len(text)
	}
	return i
}

// psMatch finds the bracket closing the one at open.
func psMatch(text string, open int) (int, bool) {
	depth := 0
	for i := open; i < len(text); {
		if j := psSkip(text, i); j != i {
			i = j
			continue
		}
		switch text[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
		i++
	}
	return 0, false
}

func psSplitTopLevel(body string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(body); {
		if j := psSkip(body, i); j != i {
			i = j
			continue
		}
		switch body[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, body[start:i])
				start = i + 1
			}
		}
		i++
	}
	return append(parts, body[start:])
}

func psParseEntry(entry string) (Param, bool) {
	var comments, attrs []string
	name, rest := "", ""
	for i := 0; i < len(entry); {
		c := entry[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case strings.HasPrefix(entry[i:], "<#"):
			j := psSkip(entry, i)
			comments = append(comments, strings.TrimSuffix(strings.TrimPrefix(entry[i:j], "<#"), "#>"))
			i = j
		case c == '#':
			j := lineEnd(entry, i)
			comments = append(comments, entry[i+1:j])
			i = j
		case c == '[':
			end, ok := psMatch(entry, i)
			if !ok {
				return Param{}, false
			}
			attrs = append(attrs, entry[i+1:end])
			i = end + 1
		case c == '$':
			m := psVariable.FindStringSubmatch(entry[i:])
			if m == nil {
				return Param{}, false
			}
			name = m[1] + m[2]
			rest = entry[i+len(m[0]):]
			i = len(entry)
		default:
			return Param{}, false
		}
	}
	if name == "" {
		return Param{}, false
	}

	p := Param{Name: name, Flag: "-" + name, Kind: Value}
	isSwitch, isArray := false, false
	for _, a := range attrs {
		a = strings.TrimSpace(a)
		head, _, hasArgs := strings.Cut(a, "(")
		head = strings.ToLower(strings.TrimSpace(head))
		if !hasArgs {
			typ := strings.ToLower(strings.ReplaceAll(a, " ", ""))
			if strings.HasSuffix(typ, "[]") {
				isArray = true
				typ = strings.TrimSuffix(typ, "[]")
			}
			switch typ {
			case "switch", "switchparameter", "system.management.automation.switchparameter":
				isSwitch = true
			}
			continue
		}
		switch head {
		case "parameter":
			if m := psMandatory.FindStringSubmatch(a); m != nil {
				v := strings.ToLower(m[1])
				p.Required = v != "false" && v != "0"
			}
			if m := psHelpMessage.FindStringSubmatch(a); m != nil {
				p.Description = psUnquote(m[1])
			}
		case "validateset":
			for _, m := range psStringLit.FindAllString(a, -1) {
				p.Choices = append(p.Choices, psUnquote(m))
			}
		}
	}
	if p.Description == "" && len(comments) > 0 {
		p.Description = collapseSpace(strings.Join(comments, " "))
	}

	switch {
	case isSwitch:
		p.Kind = Flag
		p.Choices = nil
		p.Required = false
	case len(p.Choices) > 0 && isArray:
		p.Kind = MultiChoice
	case len(p.Choices) > 0:
		p.Kind = Choice
	case isArray:
		p.Kind = MultiValue
	}

	if eq := strings.TrimSpace(rest); strings.HasPrefix(eq, "=") && p.Kind != Flag {
		p.Default = psDefault(strings.TrimSpace(stripPSComments(eq[1:])))
	}
	return p, true
}

func stripPSComments(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "<#") || s[i] == '#' {
			i = psSkip(s, i)
			continue
		}
		if j := psSkip(s, i); j != i {
			b.WriteString(s[i:j])
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func psDefault(expr string) string {
	inner := expr
	if strings.HasPrefix(inner, "@(") && strings.HasSuffix(inner, ")") {
		inner = inner[2 : len(inner)-1]
	}
	parts := psSplitTopLevel(inner)
	vals := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if psStringLit.FindString(part) == part {
			vals = append(vals, psUnquote(part))
			continue
		}
		if len(parts) == 1 {
			return expr
		}
		vals = append(vals, part)
	}
	return strings.Join(vals, ", ")
}

func psUnquote(s string) string {
	if len(s) < 2 {
		return s
	}
	switch s[0] {
	case '\'':
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	case '"':
		inner := s[1 : len(s)-1]
		inner = strings.ReplaceAll(inner, `""`, `"`)
		inner = strings.ReplaceAll(inner, "`\"", `"`)
		return strings.ReplaceAll(inner, "``", "`")
	}
	return s
}
