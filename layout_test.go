package goli

import (
	"testing"
)

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "ASCII text",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "single emoji",
			input:    "🌐",
			expected: 2,
		},
		{
			name:     "emoji with text",
			input:    "🌐 hello",
			expected: 8, // 2 (emoji) + 1 (space) + 5 (hello)
		},
		{
			name:     "multiple emojis",
			input:    "🌐🎉✨",
			expected: 6, // 2 + 2 + 2
		},
		{
			name:     "emoji in middle",
			input:    "a 🌐 b",
			expected: 6, // 1 + 1 + 2 + 1 + 1
		},
		{
			name:     "CJK characters",
			input:    "日本語",
			expected: 6, // 2 + 2 + 2
		},
		{
			name:     "mixed content",
			input:    "Hello 世界 🌍",
			expected: 13, // 5 + 1 + 4 + 1 + 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RuneWidth(tt.input)
			if result != tt.expected {
				t.Errorf("RuneWidth(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}
