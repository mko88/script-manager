package scriptparams

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	usageLine       = regexp.MustCompile(`(?i)^\s*usage\s*:\s*(.*)$`)
	placeholderTok  = regexp.MustCompile(`^(<[^>\s]+>|\{[^}\s]+\}|\([^)\s]*\|[^)\s]*\)|\[[^\]\s]+\]|[A-Z][A-Z0-9_]*)(\.\.\.)?$`)
	descChoices     = regexp.MustCompile(`\{([^{}|\s]+(?:\|[^{}|\s]+)+)\}|\(([^()|\s]+(?:\|[^()|\s]+)+)\)`)
	descRequired    = regexp.MustCompile(`(?i)\brequired\b`)
	descNotRequired = regexp.MustCompile(`(?i)\b(not|optional)\b[^.]*\brequired\b`)
	descDefault     = regexp.MustCompile("(?i)\\bdefault(?:s)?(?:\\s+is|\\s*[:=])?\\s+[\"'`]?([^\\s\"'`),;\\]]+)")
	getoptsCall     = regexp.MustCompile(`\bgetopts\s+["']?([A-Za-z0-9:]+)["']?`)
	optionName      = regexp.MustCompile(`^(--?|/)([\p{L}\p{N}?][\p{L}\p{N}_.-]*)$`)
	usageIgnored    = map[string]bool{"options": true, "option": true, "opts": true, "flags": true, "args": true, "arguments": true, "...": true}
)

func parseShell(text string) Spec {
	lines := headerLines(text, stripShellComment, isShellPreamble)
	spec := parseUsage(lines, false)
	if !hasOptions(spec.Params) {
		if m := getoptsCall.FindStringSubmatch(text); m != nil {
			spec.Params = append(getoptsParams(m[1]), spec.Params...)
		}
	}
	return spec
}

func parseBatch(text string) Spec {
	return parseUsage(headerLines(text, stripBatchComment, isBatchPreamble), true)
}

func hasOptions(params []Param) bool {
	for _, p := range params {
		if !p.Positional() {
			return true
		}
	}
	return false
}

func getoptsParams(optstring string) []Param {
	optstring = strings.TrimPrefix(optstring, ":")
	var out []Param
	for i := 0; i < len(optstring); i++ {
		c := optstring[i]
		if c == ':' {
			continue
		}
		p := Param{Name: string(c), Flag: "-" + string(c), Kind: Flag}
		if i+1 < len(optstring) && optstring[i+1] == ':' {
			p.Kind = Value
		}
		out = append(out, p)
	}
	return out
}

func isShellPreamble(line string) bool { return false }

func isBatchPreamble(line string) bool {
	l := strings.ToLower(strings.TrimSpace(line))
	l = strings.TrimPrefix(l, "@")
	return l == "echo off" || strings.HasPrefix(l, "setlocal")
}

func stripShellComment(line string) (string, bool) {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "#") {
		return "", false
	}
	return strings.TrimPrefix(strings.TrimLeft(t, "#"), " "), true
}

func stripBatchComment(line string) (string, bool) {
	t := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(t, "::") {
		return strings.TrimPrefix(t[2:], " "), true
	}
	t = strings.TrimPrefix(t, "@")
	if len(t) >= 3 && strings.EqualFold(t[:3], "rem") && (len(t) == 3 || t[3] == ' ' || t[3] == '\t') {
		if len(t) == 3 {
			return "", true
		}
		return t[4:], true
	}
	return "", false
}

// headerLines is the comment text at the top of a script, before its first
// line of code.
func headerLines(text string, strip func(string) (string, bool), preamble func(string) bool) []string {
	var out []string
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if i == 0 && strings.HasPrefix(line, "#!") {
			continue
		}
		if c, ok := strip(line); ok {
			out = append(out, strings.TrimRight(c, " \t"))
			continue
		}
		if strings.TrimSpace(line) == "" || preamble(line) {
			out = append(out, "")
			continue
		}
		break
	}
	return out
}

func isOptionStart(t string, slash bool) bool {
	if len(t) < 2 {
		return false
	}
	if t[0] == '-' {
		if t[1] == '-' {
			return len(t) > 2 && isIdent(t[2])
		}
		return isIdent(t[1]) || t[1] == '?'
	}
	return slash && t[0] == '/' && isIdent(t[1])
}

func indentOf(s string) int {
	return len(s) - len(strings.TrimLeft(s, " \t"))
}

func parseUsage(lines []string, slash bool) Spec {
	var spec Spec
	var options, positional []Param
	last := -1
	lastIndent := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		t := strings.TrimSpace(line)
		if t == "" {
			last = -1
			continue
		}
		if m := usageLine.FindStringSubmatch(line); m != nil {
			rest := strings.TrimSpace(m[1])
			for rest == "" && i+1 < len(lines) {
				i++
				rest = strings.TrimSpace(lines[i])
			}
			positional = append(positional, usagePositionals(rest, slash)...)
			last = -1
			continue
		}
		if isOptionStart(t, slash) {
			if p, ok := parseOptionLine(t, slash); ok {
				options = append(options, p)
				last = len(options) - 1
				lastIndent = indentOf(line)
				continue
			}
		}
		if last >= 0 && indentOf(line) > lastIndent {
			p := &options[last]
			p.Description = strings.TrimSpace(p.Description + " " + t)
			fillFromDescription(p)
			continue
		}
		last = -1
		if spec.Synopsis == "" && len(options) == 0 && !strings.HasSuffix(t, ":") {
			spec.Synopsis = t
		}
	}
	spec.Params = append(options, positional...)
	return spec
}

func parseOptionLine(t string, slash bool) (Param, bool) {
	var flags []string
	placeholder := ""
	pos := 0
	for {
		for pos < len(t) && (t[pos] == ' ' || t[pos] == ',' || t[pos] == '\t') {
			pos++
		}
		end := pos
		for end < len(t) && t[end] != ' ' && t[end] != ',' && t[end] != '\t' {
			end++
		}
		tok := t[pos:end]
		if tok == "" || !isOptionStart(tok, slash) {
			break
		}
		name, ph, _ := strings.Cut(tok, "=")
		if !optionName.MatchString(name) {
			return Param{}, false
		}
		flags = append(flags, name)
		if ph != "" {
			placeholder = ph
		}
		pos = end
	}
	if len(flags) == 0 {
		return Param{}, false
	}

	rest := t[pos:]
	if placeholder == "" {
		trimmed := strings.TrimLeft(rest, " \t")
		gapBefore := len(rest) - len(trimmed)
		tok := trimmed
		if n := strings.IndexAny(trimmed, " \t"); n >= 0 {
			tok = trimmed[:n]
		}
		remain := trimmed[len(tok):]
		gapAfter := strings.HasPrefix(remain, "  ") || strings.HasPrefix(remain, "\t")
		if tok != "" && gapBefore < 2 && (placeholderTok.MatchString(tok) || gapAfter) {
			placeholder = tok
			rest = remain
		}
	}

	flag := flags[0]
	for _, f := range flags {
		if strings.HasPrefix(f, "--") {
			flag = f
			break
		}
	}
	m := optionName.FindStringSubmatch(flag)
	p := Param{
		Name:        m[2],
		Flag:        flag,
		Kind:        Flag,
		Description: collapseSpace(rest),
	}
	if placeholder != "" {
		p.Kind = Value
		if strings.HasSuffix(placeholder, "...") {
			p.Kind = MultiValue
			p.Repeat = true
			placeholder = strings.TrimSuffix(placeholder, "...")
		}
		if c := placeholderChoices(placeholder); len(c) > 0 {
			p.Choices = c
		}
	}
	fillFromDescription(&p)
	return p, true
}

func placeholderChoices(ph string) []string {
	if len(ph) < 2 {
		return nil
	}
	inner := ph
	switch ph[0] {
	case '{', '(', '<', '[':
		inner = ph[1 : len(ph)-1]
	}
	if !strings.Contains(inner, "|") {
		return nil
	}
	var out []string
	for _, c := range strings.Split(inner, "|") {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

func fillFromDescription(p *Param) {
	if p.Kind == Flag {
		return
	}
	if len(p.Choices) == 0 {
		if m := descChoices.FindStringSubmatch(p.Description); m != nil {
			p.Choices = strings.Split(m[1]+m[2], "|")
		}
	}
	if len(p.Choices) > 0 {
		if p.Kind == MultiValue {
			p.Kind = MultiChoice
		} else if p.Kind == Value {
			p.Kind = Choice
		}
	}
	if descRequired.MatchString(p.Description) && !descNotRequired.MatchString(p.Description) {
		p.Required = true
	}
	if p.Default == "" {
		if m := descDefault.FindStringSubmatch(p.Description); m != nil {
			p.Default = m[1]
		}
	}
}

func usagePositionals(rest string, slash bool) []Param {
	fields := strings.Fields(rest)
	if len(fields) > 0 {
		fields = fields[1:]
	}
	var out []Param
	for n, f := range fields {
		optional := false
		inner := f
		if strings.HasPrefix(inner, "[") && strings.HasSuffix(inner, "]") {
			optional = true
			inner = inner[1 : len(inner)-1]
		}
		multi := strings.HasSuffix(inner, "...")
		inner = strings.TrimSuffix(inner, "...")
		if inner == "" || inner[0] == '-' || slash && inner[0] == '/' || strings.ContainsAny(inner, "[]") {
			continue
		}
		var name string
		var choices []string
		switch {
		case strings.HasPrefix(inner, "<") && strings.HasSuffix(inner, ">"):
			name = inner[1 : len(inner)-1]
		case strings.HasPrefix(inner, "{") && strings.HasSuffix(inner, "}"):
			choices = placeholderChoices(inner)
			name = "arg" + strconv.Itoa(n+1)
		case optional:
			name = inner
		case inner == strings.ToUpper(inner) && placeholderTok.MatchString(inner):
			name = inner
		default:
			continue
		}
		if c := placeholderChoices("<" + name + ">"); len(c) > 0 {
			choices = c
			name = "arg" + strconv.Itoa(n+1)
		}
		if usageIgnored[strings.ToLower(name)] || name == "" {
			continue
		}
		p := Param{Name: name, Kind: Value, Required: !optional, Choices: choices}
		switch {
		case multi && len(choices) > 0:
			p.Kind = MultiChoice
		case multi:
			p.Kind = MultiValue
		case len(choices) > 0:
			p.Kind = Choice
		}
		out = append(out, p)
	}
	return out
}
