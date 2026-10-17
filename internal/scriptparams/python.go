package scriptparams

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	pyKeyword = regexp.MustCompile(`(?s)^([A-Za-z_][A-Za-z0-9_]*)\s*=([^=].*|)$`)
	pyNumber  = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$`)
)

type pyCall struct {
	start, open int
	parser      bool
}

func parsePython(text string) Spec {
	calls := pyFindCalls(text)
	var spec Spec
	for n, c := range calls {
		// A call can't run into the next one, which keeps an unclosed
		// call from rescanning the rest of the file for every match.
		limit := len(text)
		if n+1 < len(calls) {
			limit = calls[n+1].start
		}
		args, ok := pyCallArgs(text[:limit], c.open)
		if !ok {
			continue
		}
		if c.parser {
			if spec.Synopsis == "" {
				_, kw := pySplitArgs(args)
				if s, ok := pyString(kw["description"]); ok {
					spec.Synopsis = collapseSpace(s)
				}
			}
			continue
		}
		if p, ok := pyParam(args); ok {
			spec.Params = append(spec.Params, p)
		}
	}
	return spec
}

// pyFindCalls finds the add_argument and ArgumentParser calls in code,
// passing over strings and comments, in one pass.
func pyFindCalls(text string) []pyCall {
	var calls []pyCall
	for i := 0; i < len(text); {
		if j := pySkip(text, i); j != i {
			i = j
			continue
		}
		name, parser := "", false
		switch {
		case text[i] == '.' && strings.HasPrefix(text[i:], ".add_argument"):
			name = ".add_argument"
		case text[i] == 'A' && strings.HasPrefix(text[i:], "ArgumentParser") && (i == 0 || !isIdent(text[i-1])):
			name, parser = "ArgumentParser", true
		default:
			i++
			continue
		}
		j := i + len(name)
		for j < len(text) && (text[j] == ' ' || text[j] == '\t') {
			j++
		}
		if j < len(text) && text[j] == '(' {
			calls = append(calls, pyCall{start: i, open: j, parser: parser})
		}
		i = j
	}
	return calls
}

func pyParam(args []string) (Param, bool) {
	positional, kw := pySplitArgs(args)
	var opts []string
	for _, a := range positional {
		s, ok := pyString(a)
		if !ok {
			return Param{}, false
		}
		opts = append(opts, s)
	}
	if len(opts) == 0 {
		return Param{}, false
	}

	p := Param{Kind: Value}
	if strings.HasPrefix(opts[0], "-") {
		p.Flag = opts[0]
		for _, o := range opts {
			if strings.HasPrefix(o, "--") {
				p.Flag = o
				break
			}
		}
		p.Name = strings.ReplaceAll(strings.TrimLeft(p.Flag, "-"), "-", "_")
	} else {
		p.Name = opts[0]
		p.Required = true
	}
	if d, ok := kw["dest"]; ok {
		if s, ok := pyString(d); ok && s != "" {
			p.Name = s
		}
	}

	if h, ok := kw["help"]; ok {
		if strings.Contains(h, "SUPPRESS") {
			return Param{}, false
		}
		if s, ok := pyString(h); ok {
			p.Description = collapseSpace(s)
		}
	}

	if a, ok := kw["action"]; ok {
		action, _ := pyString(a)
		switch action {
		case "store_true", "store_false", "count", "store_const", "append_const", "BooleanOptionalAction":
			p.Kind = Flag
		case "help", "version":
			return Param{}, false
		case "append", "extend":
			p.Kind = MultiValue
			p.Repeat = action == "append"
		}
		if strings.HasSuffix(strings.TrimSpace(a), "BooleanOptionalAction") {
			p.Kind = Flag
		}
	}
	if p.Kind == Flag {
		p.Required = false
		p.Description = strings.ReplaceAll(p.Description, "%(default)s", "")
		return p, true
	}

	if n, ok := kw["nargs"]; ok {
		n = strings.TrimSpace(n)
		s, isStr := pyString(n)
		switch {
		case isStr && (s == "+" || s == "*"):
			p.Kind = MultiValue
			if s == "*" && p.Positional() {
				p.Required = false
			}
		case isStr && s == "?":
			if p.Positional() {
				p.Required = false
			}
		case !isStr:
			if v, err := strconv.Atoi(n); err == nil && v > 1 {
				p.Kind = MultiValue
			}
		}
	}

	if c, ok := kw["choices"]; ok {
		if list, ok := pyList(c); ok && len(list) > 0 {
			p.Choices = list
			if p.Kind == MultiValue {
				p.Kind = MultiChoice
			} else {
				p.Kind = Choice
			}
		}
	}

	if r, ok := kw["required"]; ok && !p.Positional() {
		p.Required = strings.TrimSpace(r) == "True"
	}

	if d, ok := kw["default"]; ok {
		d = strings.TrimSpace(d)
		if s, ok := pyString(d); ok {
			p.Default = s
		} else if list, ok := pyList(d); ok {
			p.Default = strings.Join(list, ", ")
		} else if d != "None" {
			p.Default = d
		}
	}
	p.Description = strings.ReplaceAll(p.Description, "%(default)s", p.Default)
	return p, true
}

func pySplitArgs(args []string) (positional []string, kw map[string]string) {
	kw = map[string]string{}
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if m := pyKeyword.FindStringSubmatch(a); m != nil {
			kw[m[1]] = strings.TrimSpace(m[2])
			continue
		}
		positional = append(positional, a)
	}
	return positional, kw
}

// pyCallArgs splits the arguments of the call whose "(" is at open on its
// top-level commas.
func pyCallArgs(text string, open int) ([]string, bool) {
	var parts []string
	depth, start := 0, open+1
	for i := open; i < len(text); {
		if j := pySkip(text, i); j != i {
			i = j
			continue
		}
		switch text[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 {
				return append(parts, text[start:i]), true
			}
		case ',':
			if depth == 1 {
				parts = append(parts, text[start:i])
				start = i + 1
			}
		}
		i++
	}
	return nil, false
}

// pySkip returns the index just past a string literal or comment starting
// at i, or i when there is none.
func pySkip(text string, i int) int {
	if text[i] == '#' {
		if n := strings.IndexByte(text[i:], '\n'); n >= 0 {
			return i + n
		}
		return len(text)
	}
	j := i
	for j < len(text) && j-i < 3 && strings.ContainsRune("rRbBuUfF", rune(text[j])) {
		j++
	}
	if j >= len(text) || (text[j] != '\'' && text[j] != '"') {
		return i
	}
	if j > i && i > 0 && isIdent(text[i-1]) {
		return i
	}
	q := text[j]
	triple := strings.Repeat(string(q), 3)
	if strings.HasPrefix(text[j:], triple) {
		if n := strings.Index(text[j+3:], triple); n >= 0 {
			return j + 3 + n + 3
		}
		return len(text)
	}
	for k := j + 1; k < len(text); k++ {
		switch text[k] {
		case '\\':
			k++
		case q:
			return k + 1
		case '\n':
			return k
		}
	}
	return len(text)
}

func isIdent(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// pyString reads a string literal, or several adjacent ones, without
// evaluating anything else.
func pyString(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = strings.TrimSpace(expr[1 : len(expr)-1])
	}
	var b strings.Builder
	any := false
	for i := 0; i < len(expr); {
		c := expr[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\\' {
			i++
			continue
		}
		j := pySkip(expr, i)
		if j == i || expr[i] == '#' {
			if expr[i] == '#' {
				i = j
				continue
			}
			return "", false
		}
		b.WriteString(pyLiteralBody(expr[i:j]))
		any = true
		i = j
	}
	return b.String(), any
}

func pyLiteralBody(lit string) string {
	raw := false
	for len(lit) > 0 && lit[0] != '\'' && lit[0] != '"' {
		if lit[0] == 'r' || lit[0] == 'R' {
			raw = true
		}
		lit = lit[1:]
	}
	n := 1
	if len(lit) >= 6 && (strings.HasPrefix(lit, `"""`) || strings.HasPrefix(lit, `'''`)) {
		n = 3
	}
	if len(lit) < 2*n {
		return ""
	}
	body := lit[n : len(lit)-n]
	if raw {
		return body
	}
	r := strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\'`, "'", `\"`, `"`, `\\`, `\`)
	return r.Replace(body)
}

// pyList reads a list or tuple of string and number literals.
func pyList(expr string) ([]string, bool) {
	expr = strings.TrimSpace(expr)
	if len(expr) < 2 {
		return nil, false
	}
	if !(expr[0] == '[' && expr[len(expr)-1] == ']') && !(expr[0] == '(' && expr[len(expr)-1] == ')') {
		return nil, false
	}
	items, ok := pyCallArgs(expr, 0)
	if !ok {
		return nil, false
	}
	var out []string
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		if s, ok := pyString(it); ok {
			out = append(out, s)
			continue
		}
		if pyNumber.MatchString(it) {
			out = append(out, it)
			continue
		}
		return nil, false
	}
	return out, true
}
