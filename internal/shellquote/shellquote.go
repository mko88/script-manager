// Package shellquote renders strings as single literal arguments for the
// shells the apps write scripts for.
package shellquote

import "strings"

// PowerShell single-quotes s, doubling every character PowerShell treats
// as a single quote, including the typographic ones.
func PowerShell(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
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

// POSIX single-quotes s for sh and its relatives.
func POSIX(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
