package scriptparams

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"script-manager/internal/shellquote"
)

// Values maps a parameter name to its value: a bool for a flag, a string,
// or a list of strings for a multi-valued parameter.
type Values map[string]any

type family int

const (
	famUnknown family = iota
	famSh
	famPwsh
	famCmd
)

func familyOf(shellBase string) family {
	switch shellBase {
	case "pwsh", "powershell":
		return famPwsh
	case "cmd":
		return famCmd
	case "bash", "sh", "zsh", "dash", "ksh", "ash", "busybox":
		return famSh
	}
	return famUnknown
}

// cmdUnsafe are the characters cmd.exe still interprets inside double
// quotes, or that end the quoted argument.
const cmdUnsafe = "\"%!^&|<>\r\n"

// Quote renders value as one literal argument in the stub language of
// shellBase. Errors never include the value, which may hold a secret.
func Quote(shellBase, value string) (string, error) {
	if strings.ContainsRune(value, 0) {
		return "", errors.New("the value contains a NUL character")
	}
	switch familyOf(shellBase) {
	case famPwsh:
		return shellquote.PowerShell(value), nil
	case famSh:
		return shellquote.POSIX(value), nil
	case famCmd:
		if strings.ContainsAny(value, cmdUnsafe) {
			return "", errors.New(`the value contains a character cmd can't take inside quotes (" % ! ^ & | < > or a line break); use the Arguments box instead`)
		}
		// A trailing backslash would escape the closing quote for the
		// program reading its command line.
		if strings.HasSuffix(value, `\`) {
			return "", errors.New("the value ends in a backslash, which can't be passed safely through cmd; use the Arguments box instead")
		}
		return `"` + value + `"`, nil
	}
	return "", unknownShellError(shellBase)
}

func unknownShellError(shellBase string) error {
	return fmt.Errorf("parameter values can't be quoted for the %s shell; use the Arguments box instead", shellBase)
}

// flagToken quotes an option name for sh when it holds a character the
// shell would treat specially, such as the ? in -?.
func flagToken(fam family, flag string) string {
	if fam != famSh {
		return flag
	}
	for _, r := range flag {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("-_./=", r) {
			return shellquote.POSIX(flag)
		}
	}
	return flag
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
					return fmt.Errorf("%s: the value isn't one of %s", p.Name, strings.Join(p.Choices, ", "))
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

// UnknownNames lists the names in values that carry something to pass but
// aren't parameters of spec.
func UnknownNames(spec Spec, values Values) []string {
	known := make(map[string]bool, len(spec.Params))
	for _, p := range spec.Params {
		known[p.Name] = true
	}
	var out []string
	for k, v := range values {
		if known[k] {
			continue
		}
		if len(Strings(v)) > 0 && !isOff(v) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func isOff(v any) bool {
	s, ok := v.(string)
	return ok && (strings.EqualFold(s, "false") || s == "0")
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
	flag := func(p Param) (string, error) {
		if !optionName.MatchString(p.Flag) {
			return "", fmt.Errorf("%s: %q is not a usable option name", p.Name, p.Flag)
		}
		return flagToken(fam, p.Flag), nil
	}
	for _, p := range spec.Params {
		v := values[p.Name]
		if p.Kind == Flag {
			if isOn(v) {
				f, err := flag(p)
				if err != nil {
					return "", err
				}
				tokens = append(tokens, f)
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
		f, err := flag(p)
		if err != nil {
			return "", err
		}
		if p.Repeat {
			for _, q := range quoted {
				tokens = append(tokens, f, q)
			}
			continue
		}
		tokens = append(tokens, f)
		tokens = append(tokens, quoted...)
	}
	all := append(tokens, positional...)
	if fam == famUnknown && len(all) > 0 {
		return "", unknownShellError(shellBase)
	}
	return strings.Join(all, " "), nil
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
