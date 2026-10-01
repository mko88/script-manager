// Package ansi turns terminal output into plain text plus the styling its
// escape sequences asked for.
package ansi

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Span is one styled run of the parsed text. From and To count UTF-16 code
// units, which is how the frontend's editor addresses its document.
//
// FG and BG are "" for the default colour, "ansi-0" to "ansi-15" for a
// palette colour, "#rrggbb" for an exact one, or "fg"/"bg" when inverse
// video put the default text or background colour there.
type Span struct {
	From      int    `json:"from"`
	To        int    `json:"to"`
	FG        string `json:"fg,omitempty"`
	BG        string `json:"bg,omitempty"`
	Bold      bool   `json:"bold,omitempty"`
	Dim       bool   `json:"dim,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Underline bool   `json:"underline,omitempty"`
	Strike    bool   `json:"strike,omitempty"`
}

type style struct {
	fg, bg    string
	bold      bool
	dim       bool
	italic    bool
	underline bool
	strike    bool
	inverse   bool
}

func (s style) span() Span {
	fg, bg := s.fg, s.bg
	if s.inverse {
		fg, bg = bg, fg
		if fg == "" {
			fg = "bg"
		}
		if bg == "" {
			bg = "fg"
		}
	}
	return Span{FG: fg, BG: bg, Bold: s.bold, Dim: s.dim, Italic: s.italic, Underline: s.underline, Strike: s.strike}
}

type parser struct {
	buf   []byte
	u16   int
	spans []Span

	lineStart    int
	lineStartU16 int
	// A carriage return only takes effect once something is written after
	// it: a line ending in one stays visible until its replacement arrives,
	// and one followed by a line feed is just a line break.
	pendingCR bool

	style  style
	cur    Span
	styled bool
}

// Parse returns raw without its escape sequences, and the styled runs its
// colour and attribute sequences describe. Sequences that move the cursor,
// set a title and the like are dropped; an escape sequence cut off at the
// end of raw is left out, so parsing output that is still being written
// never shows half of one.
func Parse(raw string) (string, []Span) {
	p := parser{buf: make([]byte, 0, len(raw))}
	for i := 0; i < len(raw); {
		switch c := raw[i]; c {
		case 0x1b:
			n, complete := p.escape(raw[i:])
			if !complete {
				return p.result()
			}
			i += n
		case '\r':
			p.pendingCR = true
			i++
		case '\n':
			p.pendingCR = false
			p.put('\n')
			p.lineStart, p.lineStartU16 = len(p.buf), p.u16
			i++
		case '\b':
			p.backspace()
			i++
		case 0x07:
			i++
		default:
			r, size := utf8.DecodeRuneInString(raw[i:])
			if p.pendingCR {
				p.pendingCR = false
				p.truncate(p.lineStart, p.lineStartU16)
			}
			p.put(r)
			i += size
		}
	}
	return p.result()
}

func (p *parser) result() (string, []Span) {
	if len(p.spans) == 0 {
		return string(p.buf), nil
	}
	return string(p.buf), p.spans
}

func utf16Len(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

func (p *parser) put(r rune) {
	p.buf = utf8.AppendRune(p.buf, r)
	end := p.u16 + utf16Len(r)
	if p.styled {
		if n := len(p.spans); n > 0 && p.spans[n-1].To == p.u16 && sameStyle(p.spans[n-1], p.cur) {
			p.spans[n-1].To = end
		} else {
			span := p.cur
			span.From, span.To = p.u16, end
			p.spans = append(p.spans, span)
		}
	}
	p.u16 = end
}

func sameStyle(a, b Span) bool {
	a.From, a.To = b.From, b.To
	return a == b
}

func (p *parser) truncate(byteLen, u16 int) {
	p.buf = p.buf[:byteLen]
	p.u16 = u16
	for n := len(p.spans); n > 0 && p.spans[n-1].From >= u16; n-- {
		p.spans = p.spans[:n-1]
	}
	if n := len(p.spans); n > 0 && p.spans[n-1].To > u16 {
		p.spans[n-1].To = u16
	}
}

func (p *parser) backspace() {
	if p.pendingCR || len(p.buf) <= p.lineStart {
		return
	}
	r, size := utf8.DecodeLastRune(p.buf)
	p.truncate(len(p.buf)-size, p.u16-utf16Len(r))
}

func inRange(c, lo, hi byte) bool { return c >= lo && c <= hi }

// escape consumes the escape sequence s starts with and returns its length.
// complete is false when s ends before the sequence does.
func (p *parser) escape(s string) (n int, complete bool) {
	if len(s) < 2 {
		return 0, false
	}
	switch s[1] {
	case '[':
		j := 2
		for j < len(s) && inRange(s[j], 0x30, 0x3f) {
			j++
		}
		params := s[2:j]
		interStart := j
		for j < len(s) && inRange(s[j], 0x20, 0x2f) {
			j++
		}
		if j >= len(s) {
			return 0, false
		}
		if !inRange(s[j], 0x40, 0x7e) {
			return j, true
		}
		private := params != "" && inRange(params[0], '<', '?')
		if s[j] == 'm' && j == interStart && !private {
			p.sgr(params)
		}
		return j + 1, true
	case ']', 'P', 'X', '^', '_':
		for j := 2; j < len(s); j++ {
			switch s[j] {
			case 0x07:
				return j + 1, true
			case 0x1b:
				if j+1 >= len(s) {
					return 0, false
				}
				if s[j+1] == '\\' {
					return j + 2, true
				}
			}
		}
		return 0, false
	default:
		j := 1
		for j < len(s) && inRange(s[j], 0x20, 0x2f) {
			j++
		}
		if j >= len(s) {
			return 0, false
		}
		if inRange(s[j], 0x30, 0x7e) {
			return j + 1, true
		}
		return j, true
	}
}

// num reads a decimal parameter: 0 when empty, as the sequences define it,
// and -1 when it isn't a small number.
func num(s string) int {
	if len(s) > 4 {
		return -1
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if !inRange(s[i], '0', '9') {
			return -1
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func (p *parser) sgr(params string) {
	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		if strings.Contains(parts[i], ":") {
			p.sgrColon(strings.Split(parts[i], ":"))
			continue
		}
		code := num(parts[i])
		if code == 38 || code == 48 {
			colour, used := extendedColour(parts[i+1:])
			i += used
			p.setColour(code == 38, colour)
			continue
		}
		p.apply(code)
	}
	p.cur = p.style.span()
	p.styled = p.style != style{}
}

func (p *parser) setColour(foreground bool, colour string) {
	if colour == "" {
		return
	}
	if foreground {
		p.style.fg = colour
	} else {
		p.style.bg = colour
	}
}

func (p *parser) apply(code int) {
	s := &p.style
	switch {
	case code == 0:
		*s = style{}
	case code == 1:
		s.bold = true
	case code == 2:
		s.dim = true
	case code == 3:
		s.italic = true
	case code == 4:
		s.underline = true
	case code == 7:
		s.inverse = true
	case code == 9:
		s.strike = true
	case code == 22:
		s.bold, s.dim = false, false
	case code == 23:
		s.italic = false
	case code == 24:
		s.underline = false
	case code == 27:
		s.inverse = false
	case code == 29:
		s.strike = false
	case code >= 30 && code <= 37:
		s.fg = palette(code - 30)
	case code == 39:
		s.fg = ""
	case code >= 40 && code <= 47:
		s.bg = palette(code - 40)
	case code == 49:
		s.bg = ""
	case code >= 90 && code <= 97:
		s.fg = palette(code - 90 + 8)
	case code >= 100 && code <= 107:
		s.bg = palette(code - 100 + 8)
	}
}

// extendedColour reads the parameters after a 38 or 48 and reports how many
// of them it used, so the attributes that follow are still applied.
func extendedColour(rest []string) (colour string, used int) {
	if len(rest) == 0 {
		return "", 0
	}
	switch num(rest[0]) {
	case 5:
		if len(rest) < 2 {
			return "", len(rest)
		}
		return colour256(num(rest[1])), 2
	case 2:
		if len(rest) < 4 {
			return "", len(rest)
		}
		return rgb(num(rest[1]), num(rest[2]), num(rest[3])), 4
	}
	return "", 1
}

// sgrColon handles one parameter written with colon sub-parameters: an
// underline style (4:3) or a colour (38:5:n, 38:2:r:g:b, 38:2::r:g:b).
func (p *parser) sgrColon(sub []string) {
	switch code := num(sub[0]); code {
	case 4:
		p.style.underline = num(sub[1]) != 0
	case 38, 48:
		switch {
		case sub[1] == "5" && len(sub) >= 3:
			p.setColour(code == 38, colour256(num(sub[2])))
		case sub[1] == "2" && len(sub) >= 5:
			n := len(sub)
			p.setColour(code == 38, rgb(num(sub[n-3]), num(sub[n-2]), num(sub[n-1])))
		}
	}
}

func palette(n int) string { return fmt.Sprintf("ansi-%d", n) }

func rgb(r, g, b int) string {
	for _, v := range [3]int{r, g, b} {
		if v < 0 || v > 255 {
			return ""
		}
	}
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

var cubeLevels = [6]int{0, 95, 135, 175, 215, 255}

func colour256(n int) string {
	switch {
	case n < 0 || n > 255:
		return ""
	case n < 16:
		return palette(n)
	case n < 232:
		n -= 16
		return rgb(cubeLevels[n/36], cubeLevels[n/6%6], cubeLevels[n%6])
	default:
		gray := 8 + 10*(n-232)
		return rgb(gray, gray, gray)
	}
}
