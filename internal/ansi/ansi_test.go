package ansi

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantText  string
		wantSpans []Span
	}{
		{
			name:     "plain text passes through",
			raw:      "hello\nworld\n",
			wantText: "hello\nworld\n",
		},
		{
			name:      "foreground colour until reset",
			raw:       "\x1b[31mred\x1b[0m plain",
			wantText:  "red plain",
			wantSpans: []Span{{From: 0, To: 3, FG: "ansi-1"}},
		},
		{
			name:      "empty parameter list resets",
			raw:       "\x1b[32mok\x1b[m done",
			wantText:  "ok done",
			wantSpans: []Span{{From: 0, To: 2, FG: "ansi-2"}},
		},
		{
			name:      "bright foreground",
			raw:       "\x1b[91mx",
			wantText:  "x",
			wantSpans: []Span{{From: 0, To: 1, FG: "ansi-9"}},
		},
		{
			name:      "background and bright background",
			raw:       "\x1b[44ma\x1b[102mb",
			wantText:  "ab",
			wantSpans: []Span{{From: 0, To: 1, BG: "ansi-4"}, {From: 1, To: 2, BG: "ansi-10"}},
		},
		{
			name:      "several attributes in one sequence",
			raw:       "\x1b[1;4;32mok",
			wantText:  "ok",
			wantSpans: []Span{{From: 0, To: 2, FG: "ansi-2", Bold: true, Underline: true}},
		},
		{
			name:      "dim italic strike",
			raw:       "\x1b[2;3;9mx",
			wantText:  "x",
			wantSpans: []Span{{From: 0, To: 1, Dim: true, Italic: true, Strike: true}},
		},
		{
			name:      "22 turns bold and dim off",
			raw:       "\x1b[1;2mA\x1b[22mB",
			wantText:  "AB",
			wantSpans: []Span{{From: 0, To: 1, Bold: true, Dim: true}},
		},
		{
			name:     "individual attribute resets",
			raw:      "\x1b[3;4;9ma\x1b[23mb\x1b[24mc\x1b[29md",
			wantText: "abcd",
			wantSpans: []Span{
				{From: 0, To: 1, Italic: true, Underline: true, Strike: true},
				{From: 1, To: 2, Underline: true, Strike: true},
				{From: 2, To: 3, Strike: true},
			},
		},
		{
			name:      "39 and 49 restore default colours",
			raw:       "\x1b[31;42ma\x1b[39mb\x1b[49mc",
			wantText:  "abc",
			wantSpans: []Span{{From: 0, To: 1, FG: "ansi-1", BG: "ansi-2"}, {From: 1, To: 2, BG: "ansi-2"}},
		},
		{
			name:      "256-colour index below 16 maps to the palette",
			raw:       "\x1b[38;5;9mx",
			wantText:  "x",
			wantSpans: []Span{{From: 0, To: 1, FG: "ansi-9"}},
		},
		{
			name:      "256-colour cube",
			raw:       "\x1b[38;5;196ma\x1b[48;5;67mb",
			wantText:  "ab",
			wantSpans: []Span{{From: 0, To: 1, FG: "#ff0000"}, {From: 1, To: 2, FG: "#ff0000", BG: "#5f87af"}},
		},
		{
			name:      "256-colour grayscale ramp",
			raw:       "\x1b[38;5;244mx",
			wantText:  "x",
			wantSpans: []Span{{From: 0, To: 1, FG: "#808080"}},
		},
		{
			name:      "truecolor foreground and background",
			raw:       "\x1b[38;2;1;2;3;48;2;255;128;0mx",
			wantText:  "x",
			wantSpans: []Span{{From: 0, To: 1, FG: "#010203", BG: "#ff8000"}},
		},
		{
			name:      "attributes after an extended colour still apply",
			raw:       "\x1b[38;5;196;1mx",
			wantText:  "x",
			wantSpans: []Span{{From: 0, To: 1, FG: "#ff0000", Bold: true}},
		},
		{
			name:      "colon-separated truecolor",
			raw:       "\x1b[38:2::255:128:0ma\x1b[0m\x1b[38:2:1:2:3mb\x1b[0m\x1b[38:5:196mc",
			wantText:  "abc",
			wantSpans: []Span{{From: 0, To: 1, FG: "#ff8000"}, {From: 1, To: 2, FG: "#010203"}, {From: 2, To: 3, FG: "#ff0000"}},
		},
		{
			name:      "colon underline styles",
			raw:       "\x1b[4:3ma\x1b[4:0mb",
			wantText:  "ab",
			wantSpans: []Span{{From: 0, To: 1, Underline: true}},
		},
		{
			name:      "inverse swaps with the default colours",
			raw:       "\x1b[7mx\x1b[27my",
			wantText:  "xy",
			wantSpans: []Span{{From: 0, To: 1, FG: "bg", BG: "fg"}},
		},
		{
			name:      "inverse swaps explicit colours",
			raw:       "\x1b[31;7ma\x1b[44mb",
			wantText:  "ab",
			wantSpans: []Span{{From: 0, To: 1, FG: "bg", BG: "ansi-1"}, {From: 1, To: 2, FG: "ansi-4", BG: "ansi-1"}},
		},
		{
			name:     "unknown SGR codes are ignored",
			raw:      "\x1b[5;53;73mx",
			wantText: "x",
		},
		{
			name:      "the same style set twice is one span",
			raw:       "\x1b[31ma\x1b[31mb",
			wantText:  "ab",
			wantSpans: []Span{{From: 0, To: 2, FG: "ansi-1"}},
		},
		{
			name:      "a span runs across line breaks",
			raw:       "\x1b[31ma\nb\x1b[0m",
			wantText:  "a\nb",
			wantSpans: []Span{{From: 0, To: 3, FG: "ansi-1"}},
		},
		{
			name:     "non-SGR control sequences are removed",
			raw:      "a\x1b[2Kb\x1b[1;1Hc\x1b[?25l\x1b[>4;2md\x1b[0 qe",
			wantText: "abcde",
		},
		{
			name:     "OSC ended by BEL is removed",
			raw:      "\x1b]0;window title\x07x",
			wantText: "x",
		},
		{
			name:     "OSC ended by ST is removed",
			raw:      "\x1b]8;;http://example.com\x1b\\link\x1b]8;;\x1b\\",
			wantText: "link",
		},
		{
			name:     "other string sequences are removed",
			raw:      "a\x1bPq#0;2;0;0;0\x1b\\b\x1b_app\x1b\\c",
			wantText: "abc",
		},
		{
			name:     "two-character and charset escapes are removed",
			raw:      "\x1b(Bx\x1b=y\x1b7z",
			wantText: "xyz",
		},
		{
			name:     "bell is dropped",
			raw:      "a\x07b",
			wantText: "ab",
		},
		{
			name:     "CRLF becomes LF",
			raw:      "one\r\ntwo\r\n",
			wantText: "one\ntwo\n",
		},
		{
			name:     "a carriage return restarts the line",
			raw:      "10%\r20%\r100%\ndone",
			wantText: "100%\ndone",
		},
		{
			name:     "a carriage return only rewrites its own line",
			raw:      "keep\nold\rnew",
			wantText: "keep\nnew",
		},
		{
			name:     "a trailing carriage return keeps the line until more arrives",
			raw:      "50%\r",
			wantText: "50%",
		},
		{
			name:     "an erase after a carriage return still rewrites",
			raw:      "50%\r\x1b[K75%",
			wantText: "75%",
		},
		{
			name:      "a rewrite drops the spans of the text it replaced",
			raw:       "\x1b[31mold\r\x1b[32mnew",
			wantText:  "new",
			wantSpans: []Span{{From: 0, To: 3, FG: "ansi-2"}},
		},
		{
			name:      "a rewrite clips a span that started on an earlier line",
			raw:       "\x1b[31ma\nbc\r\x1b[0md",
			wantText:  "a\nd",
			wantSpans: []Span{{From: 0, To: 2, FG: "ansi-1"}},
		},
		{
			name:     "backspace erases the previous character",
			raw:      "ab\bc",
			wantText: "ac",
		},
		{
			name:     "backspace stops at the start of the line",
			raw:      "a\n\b\bb",
			wantText: "a\nb",
		},
		{
			name:      "backspace shrinks the span it erased from",
			raw:       "\x1b[31mab\b\x1b[0mc",
			wantText:  "ac",
			wantSpans: []Span{{From: 0, To: 1, FG: "ansi-1"}},
		},
		{
			name:     "a lone escape at the end is held back",
			raw:      "ab\x1b",
			wantText: "ab",
		},
		{
			name:     "an unfinished control sequence at the end is held back",
			raw:      "ab\x1b[38;5",
			wantText: "ab",
		},
		{
			name:     "an unfinished OSC at the end is held back",
			raw:      "ab\x1b]0;tit",
			wantText: "ab",
		},
		{
			name:      "offsets count UTF-16 units",
			raw:       "é😀\x1b[31mx😀\x1b[0my",
			wantText:  "é😀x😀y",
			wantSpans: []Span{{From: 3, To: 6, FG: "ansi-1"}},
		},
		{
			name:      "an invalid byte becomes one replacement character",
			raw:       "\xffa\x1b[31mb",
			wantText:  "�ab",
			wantSpans: []Span{{From: 2, To: 3, FG: "ansi-1"}},
		},
		{
			name:     "backspace over a surrogate pair",
			raw:      "a😀\bb\x1b[1mc",
			wantText: "abc",
			wantSpans: []Span{
				{From: 2, To: 3, Bold: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotSpans := Parse(tt.raw)
			if gotText != tt.wantText {
				t.Errorf("text = %q, want %q", gotText, tt.wantText)
			}
			if !reflect.DeepEqual(gotSpans, tt.wantSpans) {
				t.Errorf("spans = %+v, want %+v", gotSpans, tt.wantSpans)
			}
		})
	}
}
