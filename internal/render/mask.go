// Package render holds template helpers shared by the TUI and GUI frontends
// for expanding item/action templates and hiding secret values.
package render

import (
	"encoding/base64"
	"regexp"
	"strings"
)

var codeSpanRe = regexp.MustCompile("`([^`\n]+)`")

// maskPrefix is the prefix injected by MaskFunc. It must not appear in normal
// values and must be valid inside a markdown code span.
const maskPrefix = "GLMASK__"

// MaskFunc is exposed as the "mask" template function. It encodes the actual
// value so the rendering pipeline can detect and hide it; the clipboard still
// receives the real value.
func MaskFunc(value string) string {
	return maskPrefix + base64.RawURLEncoding.EncodeToString([]byte(value))
}

// ProcessMaskSpans scans the expanded template output for code spans whose
// content was produced by MaskFunc. Returns:
//   - displayMd: markdown with mask markers replaced by ••••••
//   - copyValues: actual values for the clipboard (decoded for masked spans)
//   - copyMasked: true for each index whose value was masked
func ProcessMaskSpans(expanded string) (displayMd string, copyValues []string, copyMasked []bool) {
	var result []byte
	lastIdx := 0
	for _, match := range codeSpanRe.FindAllStringSubmatchIndex(expanded, -1) {
		result = append(result, expanded[lastIdx:match[0]]...)
		value := strings.TrimSpace(expanded[match[2]:match[3]])
		if strings.HasPrefix(value, maskPrefix) {
			if actual, err := base64.RawURLEncoding.DecodeString(value[len(maskPrefix):]); err == nil {
				result = append(result, []byte("`••••••`")...)
				copyValues = append(copyValues, string(actual))
				copyMasked = append(copyMasked, true)
				lastIdx = match[1]
				continue
			}
		}
		result = append(result, expanded[match[0]:match[1]]...)
		copyValues = append(copyValues, value)
		copyMasked = append(copyMasked, false)
		lastIdx = match[1]
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
