package render

import (
	"reflect"
	"strings"
	"testing"
)

func TestProcessMaskSpansMixed(t *testing.T) {
	src := "user: `bob`\npass: `" + MaskFunc("s3cret") + "`\n"

	displayMd, copyValues, copyMasked := ProcessMaskSpans(src)

	if !strings.Contains(displayMd, "`bob`") {
		t.Errorf("unmasked span altered: %q", displayMd)
	}
	if !strings.Contains(displayMd, "`••••••`") {
		t.Errorf("masked span not replaced with bullets: %q", displayMd)
	}
	if strings.Contains(displayMd, "s3cret") || strings.Contains(displayMd, maskPrefix) {
		t.Errorf("secret or marker leaked into display output: %q", displayMd)
	}
	if want := []string{"bob", "s3cret"}; !reflect.DeepEqual(copyValues, want) {
		t.Errorf("copyValues = %v, want %v", copyValues, want)
	}
	if want := []bool{false, true}; !reflect.DeepEqual(copyMasked, want) {
		t.Errorf("copyMasked = %v, want %v", copyMasked, want)
	}
}

func TestProcessMaskSpansNoSpans(t *testing.T) {
	src := "plain text, no code spans"
	displayMd, copyValues, copyMasked := ProcessMaskSpans(src)
	if displayMd != src {
		t.Errorf("displayMd = %q, want unchanged input", displayMd)
	}
	if len(copyValues) != 0 || len(copyMasked) != 0 {
		t.Errorf("want no copy values, got %v / %v", copyValues, copyMasked)
	}
}

func TestProcessMaskSpansInvalidMarker(t *testing.T) {
	// A value that merely looks like a marker but holds invalid base64 must be
	// passed through as a normal span, not dropped.
	src := "`" + maskPrefix + "!!!not-base64!!!`"
	displayMd, copyValues, copyMasked := ProcessMaskSpans(src)
	if displayMd != src {
		t.Errorf("invalid marker should render unchanged, got %q", displayMd)
	}
	if len(copyValues) != 1 || copyMasked[0] {
		t.Errorf("invalid marker should be treated as unmasked value, got %v / %v", copyValues, copyMasked)
	}
}

func TestProcessMaskSpansMultiline(t *testing.T) {
	// Mirrors the real pipeline: ExpandAllEnv (or a hand-written
	// `{{.field}}` reference) can hand ProcessMaskSpans a code span whose
	// real value spans multiple lines, embedded in a single-line construct
	// (here, a bullet list) that a literal newline would otherwise break.
	// It's treated exactly like a masked secret from here on — same
	// copyMasked=true — just with a line-count placeholder instead of bullets.
	cert := "-----BEGIN CERTIFICATE-----\nMIIB<dummy>\n-----END CERTIFICATE-----"
	src := "- **A:** `plain`\n- **CERT:** `" + cert + "`\n- **B:** `other`\n"

	displayMd, copyValues, copyMasked := ProcessMaskSpans(src)

	if strings.Contains(displayMd, "MIIB") {
		t.Errorf("expected the real multiline value masked, not inlined: %q", displayMd)
	}
	if !strings.Contains(displayMd, "`(3-line value)`") {
		t.Errorf("expected a 3-line placeholder, got %q", displayMd)
	}
	// The bullets before and after the multiline entry must be untouched —
	// confirms the replacement didn't spill past its own code span and
	// swallow neighboring lines.
	if !strings.Contains(displayMd, "- **A:** `plain`\n") || !strings.Contains(displayMd, "\n- **B:** `other`\n") {
		t.Errorf("neighboring entries altered: %q", displayMd)
	}
	if n := strings.Count(displayMd, "\n"); n != 3 {
		t.Errorf("displayMd spans %d lines, want 3 (one per bullet): %q", n, displayMd)
	}

	if want := []string{"plain", cert, "other"}; !reflect.DeepEqual(copyValues, want) {
		t.Errorf("copyValues = %v, want %v", copyValues, want)
	}
	if want := []bool{false, true, false}; !reflect.DeepEqual(copyMasked, want) {
		t.Errorf("copyMasked = %v, want %v", copyMasked, want)
	}
}

func TestMaskedDisplayText(t *testing.T) {
	if got := MaskedDisplayText("s3cret"); got != "••••••" {
		t.Errorf("MaskedDisplayText(single-line) = %q, want bullets", got)
	}
	if got := MaskedDisplayText("a\nb\nc"); got != "(3-line value)" {
		t.Errorf("MaskedDisplayText(3 lines) = %q, want %q", got, "(3-line value)")
	}
}

func TestMarkNthCodeSpan(t *testing.T) {
	src := "a `one` b `two` c `three`"

	got := MarkNthCodeSpan(src, 1, "SENTINEL")
	if want := "a `one` b SENTINEL c `three`"; got != want {
		t.Errorf("MarkNthCodeSpan = %q, want %q", got, want)
	}

	if got := MarkNthCodeSpan(src, 5, "SENTINEL"); got != src {
		t.Errorf("out-of-range n should leave source unchanged, got %q", got)
	}
}
