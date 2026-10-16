package shellquote

import "testing"

func TestPowerShell(t *testing.T) {
	cases := map[string]string{
		"plain":                  "'plain'",
		"it's":                   "'it''s'",
		`a'; Remove-Item C:\; '`: `'a''; Remove-Item C:\; '''`,
		"$(x) `n \"q\"":          "'$(x) `n \"q\"'",
		"smart\u2019q":           "'smart\u2019\u2019q'",
		"\u2018\u201A\u201B":     "'\u2018\u2018\u201A\u201A\u201B\u201B'",
		"Größe":                  "'Größe'",
	}
	for in, want := range cases {
		if got := PowerShell(in); got != want {
			t.Errorf("PowerShell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPOSIX(t *testing.T) {
	if got := POSIX("it's $(x)"); got != `'it'\''s $(x)'` {
		t.Errorf("POSIX = %q", got)
	}
}
