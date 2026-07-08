// Package render holds template helpers shared by the TUI and GUI frontends
// for expanding item/action templates and hiding secret values.
package render

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// codeSpanRe matches a single backtick code span, content included. Newlines
// are allowed in the content: a template author writing `{{.someMultilineField}}`
// directly still needs its span recognized so ProcessMaskSpans can catch and
// mask it, the same as any other multi-line value.
var codeSpanRe = regexp.MustCompile("`([^`]+)`")

// maskPrefix is the prefix injected by MaskFunc. It must not appear in normal
// values and must be valid inside a markdown code span.
const maskPrefix = "GLMASK__"

// MaskFunc is exposed as the "mask" template function. It encodes the actual
// value so the rendering pipeline can detect and hide it; the clipboard still
// receives the real value.
func MaskFunc(value string) string {
	return maskPrefix + base64.RawURLEncoding.EncodeToString([]byte(value))
}

// MaskedDisplayText is what's shown in place of a masked entry's real value.
// A genuine secret gives no hint of its shape, so it's always bullets; a
// value that's merely long (spans multiple lines) shows its line count
// instead — that doesn't leak anything a masked value needs to hide, and
// lets several such entries be told apart at a glance. Used both when
// ProcessMaskSpans first builds displayMd and, in the TUI, when
// reconstructing the same text for a highlighted copy-mode selection.
func MaskedDisplayText(value string) string {
	if n := strings.Count(value, "\n") + 1; n > 1 {
		return fmt.Sprintf("(%d-line value)", n)
	}
	return "••••••"
}

// ProcessMaskSpans scans the expanded template output for every code span
// and replaces the ones that need to be hidden with a placeholder — a span
// produced by MaskFunc (a real secret) is always one; so is a span whose
// real value spans multiple lines, treated as masked from here on same as
// a secret would be: a list item, a table row, and even a code span itself
// are each exactly one physical line, so a literal embedded newline would
// break whatever it's sitting inside of otherwise. Returns:
//   - displayMd: markdown with masked spans replaced by MaskedDisplayText
//   - copyValues: the real value for every span, in source order
//   - copyMasked: true for every span whose displayed text isn't its real value
func ProcessMaskSpans(expanded string) (displayMd string, copyValues []string, copyMasked []bool) {
	var result []byte
	lastIdx := 0
	for _, match := range codeSpanRe.FindAllStringSubmatchIndex(expanded, -1) {
		result = append(result, expanded[lastIdx:match[0]]...)
		value := strings.TrimSpace(expanded[match[2]:match[3]])
		lastIdx = match[1]

		real := value
		masked := false
		if strings.HasPrefix(value, maskPrefix) {
			if actual, err := base64.RawURLEncoding.DecodeString(value[len(maskPrefix):]); err == nil {
				real = string(actual)
				masked = true
			}
			// Invalid/corrupt marker: fall through and treat it as a plain value.
		} else if strings.Contains(value, "\n") {
			masked = true
		}

		if masked {
			result = append(result, []byte("`"+MaskedDisplayText(real)+"`")...)
		} else {
			result = append(result, expanded[match[0]:match[1]]...)
		}
		copyValues = append(copyValues, real)
		copyMasked = append(copyMasked, masked)
	}
	result = append(result, expanded[lastIdx:]...)
	return string(result), copyValues, copyMasked
}

// MarkNthCodeSpan replaces the n-th (0-based) backtick span in source with
// sentinel, leaving all other spans untouched.
func MarkNthCodeSpan(source string, n int, sentinel string) string {
	i := -1
	return codeSpanRe.ReplaceAllStringFunc(source, func(match string) string {
		i++
		if i == n {
			return sentinel
		}
		return match
	})
}
