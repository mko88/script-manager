package scriptparams

import (
	"fmt"
	"sort"
	"strings"
)

// Values maps a parameter name to its value: a bool for a flag, a string,
// or a list of strings for a multi-valued parameter.
type Values map[string]any

type family int

const (
	famSh family = iota
	famPwsh
	famCmd
)

func familyOf(shellBase string) family {
	switch shellBase {
	case "pwsh", "powershell":
		return famPwsh
	case "cmd":
		return famCmd
	}
	return famSh
}

// cmdUnsafe are the characters cmd.exe still interprets inside double
// quotes, or that end the quoted argument.
const cmdUnsafe = "\"%!^&|<>\r\n"

// Quote renders value as one literal argument in the stub language of
// shellBase.
func Quote(shellBase, value string) (string, error) {
	if strings.ContainsRune(value, 0) {
		return "", fmt.Errorf("a value contains a NUL character")
	}
	switch familyOf(shellBase) {
	case famPwsh:
		return psQuote(value), nil
	case famCmd:
		if i := strings.IndexAny(value, cmdUnsafe); i >= 0 {
			return "", fmt.Errorf("the value %q contains %s, which can't be passed safely to cmd; use the Arguments field instead", value, describeChar(value[i]))
		}
		return `"` + value + `"`, nil
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'", nil
}

func describeChar(c byte) string {
	switch c {
	case '\r', '\n':
		return "a line break"
	}
	return fmt.Sprintf("%q", string(c))
}

// psQuote doubles every character PowerShell treats as a single quote,
// including the typographic ones.
func psQuote(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\'', 0x2018, 0x2019, 0x201A, 0x201B:
			b.WriteRune(r)
		}
		b.WriteRune(r)
	}
	b.WriteByte('\'')
	return b.String()
}

// Strings reads a value as a list of non-empty strings.
func Strings(v any) []string {
	var out []string
	add := func(s string) {
		if s != "" {
			out = append(out, s)
		}
	}
	switch x := v.(type) {
	case nil:
	case string:
		add(x)
	case []string:
		for _, s := range x {
			add(s)
		}
	case []any:
		for _, e := range x {
			if e != nil {
				add(fmt.Sprint(e))
			}
		}
	case bool:
		if x {
			add("true")
		}
	default:
		add(fmt.Sprint(x))
	}
	return out
}

func isOn(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(x, "true")
	}
	return false
}

// ExpandValues applies expand to every string in values.
func ExpandValues(values Values, expand func(string) (string, error)) (Values, error) {
	out := make(Values, len(values))
	for k, v := range values {
		switch x := v.(type) {
		case string:
			s, err := expand(x)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			out[k] = s
		case []any, []string:
			list := Strings(x)
			exp := make([]string, len(list))
			for i, s := range list {
				e, err := expand(s)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", k, err)
				}
				exp[i] = e
			}
			out[k] = exp
		default:
			out[k] = v
		}
	}
	return out, nil
}

// Validate checks required parameters, choices and the order of
// positional values.
func Validate(spec Spec, values Values) error {
	var missing []string
	gap := ""
	for _, p := range spec.Params {
		v := values[p.Name]
		if p.Kind == Flag {
			continue
		}
		vals := Strings(v)
		if !p.Multi() && len(vals) > 1 {
			return fmt.Errorf("%s takes a single value", p.Name)
		}
		if len(vals) == 0 {
			if p.Required {
				missing = append(missing, p.Name)
			}
			if p.Positional() && gap == "" {
				gap = p.Name
			}
			continue
		}
		if p.Positional() && gap != "" {
			return fmt.Errorf("%s needs a value before %s can be passed", gap, p.Name)
		}
		if len(p.Choices) > 0 {
			for _, s := range vals {
				if !contains(p.Choices, s) {
					return fmt.Errorf("%q is not a valid value for %s (choose from %s)", s, p.Name, strings.Join(p.Choices, ", "))
				}
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("required: %s", strings.Join(missing, ", "))
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// Args renders the values of spec's parameters as argument text for the
// stub shellBase runs the script from: flags as bare switches, every value
// a separate quoted argument, empty values left out.
func Args(shellBase string, spec Spec, values Values) (string, error) {
	if err := Validate(spec, values); err != nil {
		return "", err
	}
	fam := familyOf(shellBase)
	var tokens, positional []string
	for _, p := range spec.Params {
		if !p.Positional() && !optionName.MatchString(p.Flag) {
			return "", fmt.Errorf("%s: %q is not a usable option name", p.Name, p.Flag)
		}
		v := values[p.Name]
		if p.Kind == Flag {
			if isOn(v) {
				tokens = append(tokens, p.Flag)
			}
			continue
		}
		vals := Strings(v)
		if len(vals) == 0 {
			continue
		}
		quoted := make([]string, len(vals))
		for i, s := range vals {
			q, err := Quote(shellBase, s)
			if err != nil {
				return "", fmt.Errorf("%s: %w", p.Name, err)
			}
			quoted[i] = q
		}
		if p.Multi() && spec.Language == PowerShell {
			if fam == famPwsh {
				quoted = []string{strings.Join(quoted, ",")}
			} else {
				q, err := Quote(shellBase, strings.Join(vals, ","))
				if err != nil {
					return "", fmt.Errorf("%s: %w", p.Name, err)
				}
				quoted = []string{q}
			}
		}
		if p.Positional() {
			positional = append(positional, quoted...)
			continue
		}
		if p.Repeat {
			for _, q := range quoted {
				tokens = append(tokens, p.Flag, q)
			}
			continue
		}
		tokens = append(tokens, p.Flag)
		tokens = append(tokens, quoted...)
	}
	return strings.Join(append(tokens, positional...), " "), nil
}

// Join appends the free-text arguments, which are passed through as
// written, to the rendered ones.
func Join(rendered, extra string) (string, error) {
	extra = strings.TrimSpace(extra)
	if strings.ContainsAny(extra, "\r\n\x00") {
		return "", fmt.Errorf("arguments must fit on one line")
	}
	switch {
	case rendered == "":
		return extra, nil
	case extra == "":
		return rendered, nil
	}
	return rendered + " " + extra, nil
}
