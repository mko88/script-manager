package render

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

var codeSpanRe = regexp.MustCompile("`([^`]+)`")

const maskPrefix = "GLMASK__"

func MaskFunc(value string) string {
	return maskPrefix + base64.RawURLEncoding.EncodeToString([]byte(value))
}

func MaskedDisplayText(value string) string {
	if n := strings.Count(value, "\n") + 1; n > 1 {
		return fmt.Sprintf("(%d-line value)", n)
	}
	return "••••••"
}

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
