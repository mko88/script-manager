package action

import (
	"reflect"
	"strings"
	"testing"
)

func TestShellBasename(t *testing.T) {
	// Backslash paths are deliberately absent: filepath.Base splits them only
	// when the test itself runs on Windows, and these tests run on Linux.
	tests := map[string]string{
		"pwsh.exe":                   "pwsh",
		"cmd.exe":                    "cmd",
		"/usr/bin/bash":              "bash",
		"PowerShell.EXE":             "powershell",
		"C:/Program Files/pwsh/pwsh": "pwsh",
	}
	for in, want := range tests {
		if got := ShellBasename(in); got != want {
			t.Errorf("ShellBasename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScriptArgv(t *testing.T) {
	tests := []struct {
		name     string
		shell    []string
		stayOpen bool
		want     []string
	}{
		{
			name:     "pwsh strips -Command, adds -NoExit and -File",
			shell:    []string{"pwsh.exe", "-NoLogo", "-Command"},
			stayOpen: true,
			want:     []string{"pwsh.exe", "-NoLogo", "-NoExit", "-File", "s.ps1"},
		},
		{
			name:     "pwsh without stayOpen",
			shell:    []string{"pwsh.exe", "-Command"},
			stayOpen: false,
			want:     []string{"pwsh.exe", "-File", "s.ps1"},
		},
		{
			name:     "cmd stayOpen uses /k",
			shell:    []string{"cmd.exe", "/c"},
			stayOpen: true,
			want:     []string{"cmd.exe", "/k", "s.ps1"},
		},
		{
			name:     "cmd transient uses /c",
			shell:    []string{"cmd.exe"},
			stayOpen: false,
			want:     []string{"cmd.exe", "/c", "s.ps1"},
		},
		{
			name:     "posix shells strip -c and get the script appended",
			shell:    []string{"bash", "-c"},
			stayOpen: true,
			want:     []string{"bash", "s.sh"},
		},
		{
			name:     "posix shells keep other flags",
			shell:    []string{"/usr/bin/zsh", "--no-rcs"},
			stayOpen: false,
			want:     []string{"/usr/bin/zsh", "--no-rcs", "s.sh"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := "s.ps1"
			if strings.HasSuffix(tt.want[len(tt.want)-1], ".sh") {
				script = "s.sh"
			}
			got := ScriptArgv(tt.shell, script, tt.stayOpen)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ScriptArgv = %v, want %v", got, tt.want)
			}
		})
	}
}
