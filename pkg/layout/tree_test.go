package layout

import (
	"testing"
)

func TestCollapseTextWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"pure newlines", "\n\n\n", ""},
		{"pure crlf", "\r\n\r\n", ""},
		{"single word", "hello", "hello"},
		{"multiple words", "hello   world", "hello world"},
		{"leading space preserved", " hello world", " hello world"},
		{"trailing space preserved", "hello world ", "hello world "},
		{"leading and trailing", " hello world ", " hello world "},
		{"only spaces", "   ", " "},
		{"tabs collapsed", "hello\t\tworld", "hello world"},
		{"mixed whitespace", "  hello  \t world  ", " hello world "},
		{"newline in middle collapses", "hello\nworld", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collapseTextWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("collapseTextWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
