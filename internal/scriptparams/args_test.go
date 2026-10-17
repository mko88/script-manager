package scriptparams

import (
	"strings"
	"testing"
)

func TestQuoteHostileValues(t *testing.T) {
	cases := []struct {
		shell, in, want string
	}{
		{"pwsh", "plain", "'plain'"},
		{"pwsh", "a'; Remove-Item -Recurse C:\\; '", "'a''; Remove-Item -Recurse C:\\; '''"},
		{"powershell", "$(Get-Secret) `n \"x\"", "'$(Get-Secret) `n \"x\"'"},
		{"pwsh", "smart\u2019quote", "'smart\u2019\u2019quote'"},
		{"pwsh", "\u2018\u201A\u201B", "'\u2018\u2018\u201A\u201A\u201B\u201B'"},
		{"bash", "it's", `'it'\''s'`},
		{"bash", "$(rm -rf /); `id` \"x\" \\", "'$(rm -rf /); `id` \"x\" \\'"},
		{"sh", "line1\nline2", "'line1\nline2'"},
		{"cmd", `C:\Program Files\x`, `"C:\Program Files\x"`},
		{"cmd", "a b (c) = d", `"a b (c) = d"`},
	}
	for _, c := range cases {
		got, err := Quote(c.shell, c.in)
		if err != nil {
			t.Errorf("Quote(%q, %q) error: %v", c.shell, c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Quote(%q, %q) = %q, want %q", c.shell, c.in, got, c.want)
		}
	}
}

func TestQuoteCmdRejectsMetacharacters(t *testing.T) {
	for _, v := range []string{`a"b`, "100%", "%PATH%", "wow!", "a^b", "a&calc", "a|b", "a<b", "a>b", "a\r\nb", "a\nb"} {
		if _, err := Quote("cmd", v); err == nil {
			t.Errorf("Quote(cmd, %q) accepted", v)
		}
	}
	for _, sh := range []string{"cmd", "bash", "pwsh"} {
		if _, err := Quote(sh, "a\x00b"); err == nil {
			t.Errorf("Quote(%s) accepted a NUL", sh)
		}
	}
}

func psSpec() Spec {
	return Spec{Language: PowerShell, Params: []Param{
		{Name: "Configuration", Flag: "-Configuration", Kind: Choice, Choices: []string{"Debug", "Release"}},
		{Name: "Version", Flag: "-Version", Kind: Value},
		{Name: "Envs", Flag: "-Envs", Kind: MultiChoice, Choices: []string{"dev", "prod"}},
		{Name: "Tags", Flag: "-Tags", Kind: MultiValue},
		{Name: "Fail", Flag: "-Fail", Kind: Flag},
	}}
}

func TestArgsPowerShell(t *testing.T) {
	got, err := Args("pwsh", psSpec(), Values{
		"Configuration": "Release",
		"Version":       "1.0 'beta'",
		"Envs":          []any{"dev", "prod"},
		"Tags":          []string{"a", "", "b"},
		"Fail":          true,
		"Unknown":       "ignored",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `-Configuration 'Release' -Version '1.0 ''beta''' -Envs 'dev','prod' -Tags 'a','b' -Fail`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestArgsOmitsEmptyAndOffFlags(t *testing.T) {
	got, err := Args("pwsh", psSpec(), Values{"Version": "", "Fail": false, "Tags": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q, want nothing", got)
	}
}

func TestArgsPowerShellScriptFromOtherShell(t *testing.T) {
	got, err := Args("bash", psSpec(), Values{"Tags": []any{"a", "b c"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != `-Tags 'a,b c'` {
		t.Errorf("got %s", got)
	}
}

func TestArgsPythonStyle(t *testing.T) {
	spec := Spec{Language: Python, Params: []Param{
		{Name: "target", Kind: Value, Required: true},
		{Name: "extra", Kind: MultiValue},
		{Name: "verbose", Flag: "--verbose", Kind: Flag},
		{Name: "arch", Flag: "--arch", Kind: MultiChoice, Choices: []string{"x64", "arm64"}},
		{Name: "define", Flag: "-D", Kind: MultiValue, Repeat: true},
	}}
	got, err := Args("bash", spec, Values{
		"target":  "app",
		"extra":   []any{"x", "y"},
		"verbose": "true",
		"arch":    []any{"x64", "arm64"},
		"define":  []any{"A=1", "B=it's"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `--verbose --arch 'x64' 'arm64' -D 'A=1' -D 'B=it'\''s' 'app' 'x' 'y'`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestArgsCmd(t *testing.T) {
	spec := Spec{Language: Batch, Params: []Param{
		{Name: "clean", Flag: "/clean", Kind: Flag},
		{Name: "out", Flag: "--out", Kind: Value},
		{Name: "config", Kind: Value},
	}}
	got, err := Args("cmd", spec, Values{"clean": true, "out": `C:\out dir`, "config": "Release"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `/clean --out "C:\out dir" "Release"`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if _, err := Args("cmd", spec, Values{"out": "x & calc"}); err == nil {
		t.Error("cmd accepted &")
	}
}

func TestValidate(t *testing.T) {
	spec := Spec{Params: []Param{
		{Name: "env", Flag: "--env", Kind: Choice, Choices: []string{"dev", "prod"}, Required: true},
		{Name: "first", Kind: Value},
		{Name: "second", Kind: Value},
	}}
	cases := []struct {
		values Values
		want   string
	}{
		{Values{}, "required: env"},
		{Values{"env": "qa"}, "isn't one of dev, prod"},
		{Values{"env": []any{"dev", "prod"}}, "single value"},
		{Values{"env": "dev", "second": "x"}, "first needs a value"},
		{Values{"env": "dev", "first": "x", "second": "y"}, ""},
	}
	for _, c := range cases {
		err := Validate(spec, c.values)
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%v: unexpected error %v", c.values, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%v: error = %v, want %q", c.values, err, c.want)
		}
	}
	if _, err := Args("bash", spec, Values{}); err == nil {
		t.Error("Args skipped validation")
	}
}

func TestArgsRejectsUnusableFlag(t *testing.T) {
	spec := Spec{Params: []Param{{Name: "x", Flag: "-x;rm", Kind: Flag}}}
	if _, err := Args("bash", spec, Values{"x": true}); err == nil {
		t.Error("accepted a flag with shell syntax in it")
	}
}

func TestJoin(t *testing.T) {
	if got, _ := Join("-A 'b'", "  --raw $x  "); got != "-A 'b' --raw $x" {
		t.Errorf("got %q", got)
	}
	if got, _ := Join("", "x"); got != "x" {
		t.Errorf("got %q", got)
	}
	if got, _ := Join("y", ""); got != "y" {
		t.Errorf("got %q", got)
	}
	if _, err := Join("", "a\nrm -rf /"); err == nil {
		t.Error("accepted a line break")
	}
}

func TestExpandValues(t *testing.T) {
	up := func(s string) (string, error) { return strings.ToUpper(s), nil }
	got, err := ExpandValues(Values{"a": "x", "b": []any{"y", "z"}, "c": true}, up)
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "X" || got["c"] != true {
		t.Errorf("got %v", got)
	}
	if l := got["b"].([]string); l[0] != "Y" || l[1] != "Z" {
		t.Errorf("got %v", got["b"])
	}
}

func TestQuoteErrorsNeverEchoTheValue(t *testing.T) {
	const secret = "hunter2-s3cr3t"
	for _, v := range []string{secret + "%", secret + `"`, secret + `\`, secret + "\n", secret + "\x00"} {
		_, err := Quote("cmd", v)
		if err == nil {
			t.Errorf("cmd accepted %q", v)
			continue
		}
		if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("error leaks the value: %v", err)
		}
	}
	spec := Spec{Params: []Param{
		{Name: "token", Flag: "--token", Kind: Value},
		{Name: "env", Flag: "--env", Kind: Choice, Choices: []string{"dev"}},
	}}
	for _, values := range []Values{{"token": secret + "&calc"}, {"env": secret}} {
		_, err := Args("cmd", spec, values)
		if err == nil || strings.Contains(err.Error(), "hunter2") {
			t.Errorf("Args(%v) error = %v, want a refusal without the value", values, err)
		}
	}
}

func TestQuoteCmdRejectsTrailingBackslash(t *testing.T) {
	if _, err := Quote("cmd", `C:\dir\`); err == nil {
		t.Error("cmd accepted a value ending in a backslash")
	}
	if _, err := Quote("cmd", `C:\dir\x`); err != nil {
		t.Errorf("inner backslashes refused: %v", err)
	}
}

func TestArgsUnknownShell(t *testing.T) {
	spec := Spec{Params: []Param{
		{Name: "force", Flag: "--force", Kind: Flag},
		{Name: "env", Flag: "--env", Kind: Value},
	}}
	for _, values := range []Values{{"force": true}, {"env": "dev"}} {
		_, err := Args("fish", spec, values)
		if err == nil || !strings.Contains(err.Error(), "Arguments box") {
			t.Errorf("fish %v: err = %v", values, err)
		}
	}
	if got, err := Args("fish", spec, Values{}); err != nil || got != "" {
		t.Errorf("fish with nothing set = %q, %v", got, err)
	}
	for _, sh := range []string{"bash", "sh", "zsh", "dash", "ksh", "ash", "busybox"} {
		if _, err := Args(sh, spec, Values{"env": "dev"}); err != nil {
			t.Errorf("%s: %v", sh, err)
		}
	}
}

func TestArgsQuotesSpecialFlagsForSh(t *testing.T) {
	spec := Spec{Params: []Param{{Name: "help", Flag: "-?", Kind: Flag}}}
	if got, _ := Args("bash", spec, Values{"help": true}); got != "'-?'" {
		t.Errorf("bash = %q", got)
	}
	if got, _ := Args("pwsh", spec, Values{"help": true}); got != "-?" {
		t.Errorf("pwsh = %q", got)
	}
}

func TestArgsChecksOnlyEmittedFlags(t *testing.T) {
	spec := Spec{Params: []Param{
		{Name: "odd", Flag: "-x;y", Kind: Flag},
		{Name: "ok", Flag: "--ok", Kind: Flag},
	}}
	if got, err := Args("bash", spec, Values{"ok": true}); err != nil || got != "--ok" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestUnknownValues(t *testing.T) {
	spec := Spec{Params: []Param{{Name: "a", Flag: "-a", Kind: Value}}}
	got := UnknownNames(spec, Values{"a": "1", "b": "x", "c": "", "d": false, "e": []any{}, "f": "false", "g": true})
	if strings.Join(got, ",") != "b,g" {
		t.Errorf("UnknownNames = %v", got)
	}
}

func TestArgsCmdRefusesNonASCIIOptionNames(t *testing.T) {
	spec := Spec{Language: PowerShell, Params: []Param{
		{Name: "Größe", Flag: "-Größe", Kind: Value},
		{Name: "Schnell", Flag: "-Schnell", Kind: Flag},
	}}
	_, err := Args("cmd", spec, Values{"Größe": "1"})
	if err == nil || !strings.Contains(err.Error(), "Arguments box") {
		t.Errorf("cmd: err = %v", err)
	}
	if got, err := Args("cmd", spec, Values{"Schnell": true}); err != nil || got != "-Schnell" {
		t.Errorf("cmd ASCII flag = %q, %v", got, err)
	}
	if got, err := Args("pwsh", spec, Values{"Größe": "1"}); err != nil || got != "-Größe '1'" {
		t.Errorf("pwsh = %q, %v", got, err)
	}
}
