package goli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/germtb/gox"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxWidth int
		expected []string
	}{
		// Basic ASCII wrapping
		{
			name:     "short line fits",
			text:     "hello",
			maxWidth: 10,
			expected: []string{"hello"},
		},
		{
			name:     "exact fit",
			text:     "hello",
			maxWidth: 5,
			expected: []string{"hello"},
		},
		{
			name:     "wrap at word boundary",
			text:     "hello world",
			maxWidth: 7,
			expected: []string{"hello", "world"},
		},
		{
			name:     "hard wrap no spaces",
			text:     "abcdefghij",
			maxWidth: 5,
			expected: []string{"abcde", "fghij"},
		},
		{
			name:     "multiple wraps",
			text:     "one two three four",
			maxWidth: 9,
			expected: []string{"one two", "three", "four"},
		},

		// Newline handling
		{
			name:     "preserves existing newlines",
			text:     "line1\nline2",
			maxWidth: 10,
			expected: []string{"line1", "line2"},
		},
		{
			name:     "wraps long lines but preserves short ones",
			text:     "short\nthis line is too long",
			maxWidth: 10,
			expected: []string{"short", "this line", "is too", "long"},
		},

		// Wide characters (CJK, each 2 display columns)
		{
			name:     "CJK fits exactly",
			text:     "日本",
			maxWidth: 4,
			expected: []string{"日本"},
		},
		{
			name:     "CJK wraps correctly",
			text:     "日本語テスト",
			maxWidth: 6,
			expected: []string{"日本語", "テスト"},
		},
		{
			name:     "CJK won't split mid-character",
			text:     "日本語",
			maxWidth: 5, // can only fit 2 chars (4 cols), 3rd would exceed
			expected: []string{"日本", "語"},
		},

		// Emoji (each 2 display columns)
		{
			name:     "emoji wrapping",
			text:     "🌐🎉✨🚀",
			maxWidth: 4, // fits 2 emoji per line
			expected: []string{"🌐🎉", "✨🚀"},
		},
		{
			name:     "emoji won't fit on odd width",
			text:     "🌐🎉",
			maxWidth: 3, // fits 1 emoji (2 cols), 2nd would exceed
			expected: []string{"🌐", "🎉"},
		},

		// Mixed ASCII + wide characters
		{
			name:     "mixed ascii and CJK breaks at space",
			text:     "hi 日本",
			maxWidth: 5, // prefers word boundary: "hi" then "日本" (4 cols)
			expected: []string{"hi", "日本"},
		},
		{
			name:     "mixed ascii and CJK no space",
			text:     "hi日本語",
			maxWidth: 5, // "hi日" = 5 cols, then "本語" = 4 cols
			expected: []string{"hi日", "本語"},
		},
		{
			name:     "ascii word then emoji",
			text:     "hello 🌐🎉",
			maxWidth: 8, // "hello " = 6, "🌐" = 2, total 8
			expected: []string{"hello", "🌐🎉"},
		},

		// Edge cases
		{
			name:     "zero width returns as-is",
			text:     "hello",
			maxWidth: 0,
			expected: []string{"hello"},
		},
		{
			name:     "negative width returns as-is",
			text:     "hello",
			maxWidth: -1,
			expected: []string{"hello"},
		},
		{
			name:     "empty string",
			text:     "",
			maxWidth: 10,
			expected: []string{""},
		},
		{
			name:     "width 1 hard wraps ASCII",
			text:     "abc",
			maxWidth: 1,
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapText(tt.text, tt.maxWidth)
			if len(result) != len(tt.expected) {
				t.Fatalf("WrapText(%q, %d) returned %d lines, want %d\ngot:  %v\nwant: %v",
					tt.text, tt.maxWidth, len(result), len(tt.expected), result, tt.expected)
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("WrapText(%q, %d) line %d = %q, want %q\nfull result: %v",
						tt.text, tt.maxWidth, i, result[i], tt.expected[i], result)
				}
			}
			// Verify no line exceeds maxWidth in display columns
			if tt.maxWidth > 0 {
				for i, line := range result {
					w := RuneWidth(line)
					if w > tt.maxWidth {
						t.Errorf("WrapText(%q, %d) line %d (%q) has display width %d, exceeds max %d",
							tt.text, tt.maxWidth, i, line, w, tt.maxWidth)
					}
				}
			}
		})
	}
}

func TestWrapText_PreservesContent(t *testing.T) {
	// Property: joining wrapped lines should give back the original words
	inputs := []string{
		"the quick brown fox jumps over the lazy dog",
		"日本語のテスト文字列",
		"hello 🌐 world 🎉 test",
	}

	for _, input := range inputs {
		for _, width := range []int{3, 5, 8, 10, 20} {
			t.Run(fmt.Sprintf("w=%d/%q", width, input[:min(len(input), 20)]), func(t *testing.T) {
				result := WrapText(input, width)
				rejoined := strings.Join(result, " ")
				// All original non-space characters should appear
				origChars := strings.ReplaceAll(input, " ", "")
				resultChars := strings.ReplaceAll(rejoined, " ", "")
				if origChars != resultChars {
					t.Errorf("content lost: original chars %q, after wrap+join %q", origChars, resultChars)
				}
			})
		}
	}
}

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

func TestMeasureNode_Fragment(t *testing.T) {
	// gox.Map() returns __fragment__ nodes; measureNode must handle them
	fragment := gox.VNode{
		Type: gox.FragmentNodeType,
		Children: []gox.VNode{
			CreateTextNode("Hello"),
			CreateTextNode("World!"),
		},
	}

	w, h := measureNode(fragment)
	if w != 6 {
		t.Errorf("fragment width = %d, want 6 (max child width)", w)
	}
	if h != 2 {
		t.Errorf("fragment height = %d, want 2 (sum of children)", h)
	}
}

func TestComputeLayout_Fragment(t *testing.T) {
	// Fragment children should be laid out inline within a parent box
	node := gox.VNode{
		Type:  "box",
		Props: gox.Props{"width": 20, "height": 5, "direction": "column"},
		Children: []gox.VNode{
			{
				Type: gox.FragmentNodeType,
				Children: []gox.VNode{
					CreateTextNode("AAA"),
					CreateTextNode("BBB"),
				},
			},
			CreateTextNode("CCC"),
		},
	}

	box := ComputeLayout(node, LayoutContext{X: 0, Y: 0, Width: 20, Height: 5})
	buf := NewCellBuffer(20, 5)
	RenderToBuffer(box, buf, nil)
	output := buf.ToDebugString()

	if !strings.Contains(output, "AAA") {
		t.Errorf("should contain AAA from fragment child, got:\n%s", output)
	}
	if !strings.Contains(output, "BBB") {
		t.Errorf("should contain BBB from fragment child, got:\n%s", output)
	}
	if !strings.Contains(output, "CCC") {
		t.Errorf("should contain CCC after fragment, got:\n%s", output)
	}
}
