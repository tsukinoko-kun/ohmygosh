package ansicompiler_test

import (
	"strings"
	"testing"

	"github.com/tsukinoko-kun/ohmygosh/internal/ui/ansicompiler"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of tokens
	}{
		{
			name:     "simple text",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "text with ANSI",
			input:    "hello\x1b[31mworld\x1b[0m",
			expected: 12, // h,e,l,l,o,\x1b[31m,w,o,r,l,d,\x1b[0m
		},
		{
			name:     "cursor movement",
			input:    "\x1b[H\x1b[2;3Htest",
			expected: 6, // \x1b[H,\x1b[2;3H,t,e,s,t
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ansicompiler.Tokenize(tt.input)
			if len(tokens) != tt.expected {
				t.Errorf("Expected %d tokens, got %d", tt.expected, len(tokens))
			}
		})
	}
}

func TestCompileAnsiBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "preserve styling",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "\x1b[31mred text\x1b[0m",
		},
		{
			name:     "newline handling",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ansicompiler.CompileAnsi(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCompileAnsiCursorMovements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "overwrite with cursor home",
			input:    "hello\x1b[Hworld",
			expected: "world",
		},
		{
			name:     "cursor positioning",
			input:    "line1\nline2\x1b[1;3Hxxx",
			expected: "lixxx\nline2",
		},
		{
			name:     "cursor up movement",
			input:    "line1\nline2\x1b[A\x1b[6Gxxx",
			expected: "line1xxx\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ansicompiler.CompileAnsi(tt.input)
			// Normalize whitespace for comparison
			result = strings.TrimRight(result, " ")
			expected := strings.TrimRight(tt.expected, " ")
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	}
}

func TestCompileAnsiProgressBarExample(t *testing.T) {
	// Simulate a progress bar that overwrites itself
	input := "Progress: [    ]\x1b[6D\x1b[0K[█   ]\x1b[6D\x1b[0K[██  ]\x1b[6D\x1b[0K[███ ]\x1b[6D\x1b[0K[████]"
	result := ansicompiler.CompileAnsi(input)
	expected := "Progress: [████]"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
