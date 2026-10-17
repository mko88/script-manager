package scriptparams

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func parseFixture(t *testing.T, name string) Spec {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return Parse(name, string(data))
}

func param(t *testing.T, spec Spec, name string) Param {
	t.Helper()
	for _, p := range spec.Params {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("no parameter %q in %+v", name, spec.Params)
	return Param{}
}

func names(spec Spec) []string {
	var out []string
	for _, p := range spec.Params {
		out = append(out, p.Name)
	}
	return out
}

func TestPowerShellSimulateBuild(t *testing.T) {
	spec := parseFixture(t, "Simulate-Build.ps1")
	if !spec.Documented || spec.Language != PowerShell {
		t.Fatalf("spec = %+v", spec)
	}
	if got, want := names(spec), []string{"Configuration", "Version", "DelayMs", "Fail"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	cfg := param(t, spec, "Configuration")
	if cfg.Kind != Choice || !reflect.DeepEqual(cfg.Choices, []string{"Debug", "Release"}) || cfg.Default != "Release" || cfg.Flag != "-Configuration" {
		t.Errorf("Configuration = %+v", cfg)
	}
	if v := param(t, spec, "Version"); v.Kind != Value || v.Default != "0.0.0-dev" {
		t.Errorf("Version = %+v", v)
	}
	if d := param(t, spec, "DelayMs"); d.Default != "400" || d.Description != "Milliseconds to wait between steps, to make the output arrive gradually." {
		t.Errorf("DelayMs = %+v", d)
	}
	if f := param(t, spec, "Fail"); f.Kind != Flag || f.Description == "" {
		t.Errorf("Fail = %+v", f)
	}
	if spec.Synopsis == "" {
		t.Error("synopsis not read")
	}
}

func TestPowerShellHelpAndParamBlock(t *testing.T) {
	spec := parseFixture(t, "Deploy-Service.ps1")
	if got, want := names(spec), []string{"Service", "Environments", "Tags", "Strategy", "Replicas", "DryRun"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if spec.Synopsis != "Deploys a service to one or more environments." {
		t.Errorf("synopsis = %q", spec.Synopsis)
	}
	svc := param(t, spec, "Service")
	if !svc.Required || svc.Description != "Name of the service to deploy." {
		t.Errorf("Service = %+v", svc)
	}
	env := param(t, spec, "Environments")
	if env.Kind != MultiChoice || env.Required || !reflect.DeepEqual(env.Choices, []string{"dev", "staging", "prod"}) || env.Default != "dev, staging" {
		t.Errorf("Environments = %+v", env)
	}
	if tags := param(t, spec, "Tags"); tags.Kind != MultiValue {
		t.Errorf("Tags = %+v", tags)
	}
	if s := param(t, spec, "Strategy"); s.Kind != Choice || s.Default != "safe" {
		t.Errorf("Strategy = %+v", s)
	}
	if r := param(t, spec, "Replicas"); r.Default != "(Get-Default)" || r.Description != "Replica count" {
		t.Errorf("Replicas = %+v", r)
	}
	if d := param(t, spec, "DryRun"); d.Kind != Flag || d.Description != "Print what would happen." {
		t.Errorf("DryRun = %+v", d)
	}
}

func TestPowerShellLineHelp(t *testing.T) {
	spec := parseFixture(t, "line-help.ps1")
	p := param(t, spec, "Path")
	if !p.Required || p.Description != "The folder to clean." {
		t.Errorf("Path = %+v", p)
	}
	if f := param(t, spec, "Force"); f.Kind != Flag {
		t.Errorf("Force = %+v", f)
	}
	if spec.Synopsis != "Cleans a folder." {
		t.Errorf("synopsis = %q", spec.Synopsis)
	}
}

func TestUndocumented(t *testing.T) {
	for _, name := range []string{"no-docs.ps1", "malformed.ps1", "plain.py", "nodocs.sh"} {
		spec := parseFixture(t, name)
		if spec.Documented || len(spec.Params) != 0 {
			t.Errorf("%s: spec = %+v, want undocumented", name, spec)
		}
	}
	if spec := Parse("tool.exe", "MZ\x00\x01"); spec.Documented {
		t.Errorf("binary: %+v", spec)
	}
	if spec := Parse("x.ps1", "param("); spec.Documented {
		t.Errorf("truncated: %+v", spec)
	}
	if spec := Parse("x.py", "p.add_argument("); spec.Documented {
		t.Errorf("truncated python: %+v", spec)
	}
}

func TestPythonArgparse(t *testing.T) {
	spec := parseFixture(t, "build.py")
	if spec.Language != Python || !spec.Documented {
		t.Fatalf("spec = %+v", spec)
	}
	if got, want := names(spec), []string{"target", "extra", "verbose", "cache", "mode", "arch", "define", "jobs", "retries"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if spec.Synopsis != "Builds the project." {
		t.Errorf("synopsis = %q", spec.Synopsis)
	}
	if p := param(t, spec, "target"); !p.Positional() || !p.Required || p.Kind != Value {
		t.Errorf("target = %+v", p)
	}
	if p := param(t, spec, "extra"); !p.Positional() || p.Required || p.Kind != MultiValue {
		t.Errorf("extra = %+v", p)
	}
	if p := param(t, spec, "verbose"); p.Kind != Flag || p.Flag != "--verbose" || p.Description != "talk more" {
		t.Errorf("verbose = %+v", p)
	}
	if p := param(t, spec, "cache"); p.Kind != Flag || p.Flag != "--no-cache" {
		t.Errorf("cache = %+v", p)
	}
	mode := param(t, spec, "mode")
	if mode.Kind != Choice || !reflect.DeepEqual(mode.Choices, []string{"debug", "release"}) || mode.Default != "debug" || mode.Description != "build mode (default: debug)" {
		t.Errorf("mode = %+v", mode)
	}
	if p := param(t, spec, "arch"); p.Kind != MultiChoice || !reflect.DeepEqual(p.Choices, []string{"x64", "arm64"}) || p.Description != `architectures, "quoted"` {
		t.Errorf("arch = %+v", p)
	}
	if p := param(t, spec, "define"); p.Kind != MultiValue || !p.Repeat || p.Flag != "--define" {
		t.Errorf("define = %+v", p)
	}
	if p := param(t, spec, "jobs"); !p.Required || p.Default != "4" || p.Description != `parallel jobs\n` {
		t.Errorf("jobs = %+v", p)
	}
	if p := param(t, spec, "retries"); p.Kind != Value || len(p.Choices) != 0 {
		t.Errorf("retries = %+v", p)
	}
}

func TestShellHeader(t *testing.T) {
	spec := parseFixture(t, "deploy.sh")
	if got, want := names(spec), []string{"force", "env", "tag", "n", "mode", "q", "timeout", "target", "region"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if spec.Synopsis != "Deploys the app." {
		t.Errorf("synopsis = %q", spec.Synopsis)
	}
	if p := param(t, spec, "force"); p.Kind != Flag || p.Flag != "--force" || p.Description != "Skip the confirmation" {
		t.Errorf("force = %+v", p)
	}
	env := param(t, spec, "env")
	if env.Kind != Choice || !env.Required || !reflect.DeepEqual(env.Choices, []string{"dev", "staging", "prod"}) {
		t.Errorf("env = %+v", env)
	}
	if p := param(t, spec, "tag"); p.Kind != MultiValue || !p.Repeat {
		t.Errorf("tag = %+v", p)
	}
	if p := param(t, spec, "n"); p.Flag != "-n" || p.Kind != Value || p.Default != "latest" {
		t.Errorf("n = %+v", p)
	}
	if p := param(t, spec, "mode"); p.Kind != Choice || !reflect.DeepEqual(p.Choices, []string{"fast", "safe"}) {
		t.Errorf("mode = %+v", p)
	}
	if p := param(t, spec, "q"); p.Kind != Flag || p.Description != "Quiet" {
		t.Errorf("q = %+v", p)
	}
	if p := param(t, spec, "timeout"); p.Kind != Value || p.Description != "How long to wait, in seconds" {
		t.Errorf("timeout = %+v", p)
	}
	if p := param(t, spec, "target"); !p.Positional() || !p.Required {
		t.Errorf("target = %+v", p)
	}
	if p := param(t, spec, "region"); !p.Positional() || p.Required {
		t.Errorf("region = %+v", p)
	}
}

func TestShellGetoptsFallback(t *testing.T) {
	spec := parseFixture(t, "getopts.sh")
	if got, want := names(spec), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if param(t, spec, "a").Kind != Flag || param(t, spec, "b").Kind != Value || param(t, spec, "b").Flag != "-b" {
		t.Errorf("spec = %+v", spec.Params)
	}
}

func TestBatchHeader(t *testing.T) {
	spec := parseFixture(t, "build.bat")
	if spec.Language != Batch {
		t.Fatalf("language = %q", spec.Language)
	}
	if got, want := names(spec), []string{"clean", "v", "out", "config", "platform"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if p := param(t, spec, "clean"); p.Kind != Flag || p.Flag != "/clean" {
		t.Errorf("clean = %+v", p)
	}
	if p := param(t, spec, "v"); p.Kind != Choice || p.Flag != "/v" || len(p.Choices) != 3 {
		t.Errorf("v = %+v", p)
	}
	if p := param(t, spec, "out"); p.Flag != "--out" || p.Kind != Value {
		t.Errorf("out = %+v", p)
	}
	if p := param(t, spec, "config"); !p.Required || !p.Positional() {
		t.Errorf("config = %+v", p)
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		path, text string
		want       Language
	}{
		{"a.ps1", "", PowerShell},
		{"a.PY", "", Python},
		{"a.cmd", "", Batch},
		{"tool", "#!/usr/bin/env python3\n", Python},
		{"tool", "#!/usr/bin/env -S pwsh -NoProfile\n", PowerShell},
		{"tool", "#!/bin/bash\n", Shell},
		{"tool", "echo\n", Unknown},
		{"tool.exe", "#!/bin/bash\n", Shell},
	}
	for _, c := range cases {
		if got := Detect(c.path, c.text); got != c.want {
			t.Errorf("Detect(%q, %q) = %q, want %q", c.path, c.text, got, c.want)
		}
	}
}

func TestParseDropsUnusableOptionNames(t *testing.T) {
	spec := Parse("x.ps1", "param([string]${weird name}, [string]$Größe, [switch]$Ok)")
	if got := names(spec); !reflect.DeepEqual(got, []string{"Größe", "Ok"}) {
		t.Errorf("names = %v", got)
	}
	if p := param(t, spec, "Größe"); p.Flag != "-Größe" {
		t.Errorf("Größe = %+v", p)
	}
}

func TestPythonCallAfterHashInString(t *testing.T) {
	spec := Parse("x.py", `x = "#"; p.add_argument("--e", help="e")
s = "p.add_argument('--fake')"
`)
	if got := names(spec); !reflect.DeepEqual(got, []string{"e"}) {
		t.Errorf("names = %v", got)
	}
}

func TestParseIsLinearOnPathologicalInput(t *testing.T) {
	const size = 256 * 1024
	inputs := map[string]string{
		"open.py":    strings.Repeat(".add_argument(", size/14),
		"strings.py": strings.Repeat(`p.add_argument("--x", help="`, size/28),
		"triple.py":  `p.add_argument(` + strings.Repeat(`"""a`, size/4),
		"nested.py":  strings.Repeat("ArgumentParser(description=(", size/28),
		"open.ps1":   "param(" + strings.Repeat("[", size),
		"attrs.ps1":  strings.Repeat("[x(", size/3),
		"help.ps1":   strings.Repeat("<# .PARAMETER x\n", size/16),
		"quotes.ps1": "param(" + strings.Repeat("'", size),
		"opts.sh":    strings.Repeat("#  -a, --aa A  {x|y} default: z\n", size/32),
		"opts.bat":   strings.Repeat("REM  /a A  (x|y)\n", size/16),
	}
	for name, text := range inputs {
		start := time.Now()
		Parse(name, text)
		if d := time.Since(start); d > 500*time.Millisecond {
			t.Errorf("%s: %v for %d bytes", name, d, len(text))
		}
	}
}
