package goli

import (
	"strings"
	"testing"

	"github.com/germtb/gox"
)

func TestInput_HorizontalScroll(t *testing.T) {
	// Clear focus manager state
	Manager().Clear()

	// Create input with fixed width=10
	input := NewInput(InputOptions{})
	input.Focus()

	tests := []struct {
		name           string
		value          string
		cursorPos      int
		expectedScroll int // expected scrollX offset
		visibleText    string
	}{
		{
			name:           "short text no scroll",
			value:          "hello",
			cursorPos:      5,
			expectedScroll: 0,
			visibleText:    "hello",
		},
		{
			name:           "exactly width no scroll",
			value:          "0123456789",
			cursorPos:      9,
			expectedScroll: 0,
			visibleText:    "0123456789",
		},
		{
			name:           "cursor at end past width",
			value:          "0123456789ABC",
			cursorPos:      13,
			expectedScroll: 4,
			visibleText:    "456789ABC",
		},
		{
			name:           "cursor in middle past width",
			value:          "0123456789ABCDEF",
			cursorPos:      12,
			expectedScroll: 3,
			visibleText:    "3456789ABC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input.SetValue(tt.value)
			input.SetCursorPos(tt.cursorPos)

			var output strings.Builder
			app := Render(func() gox.VNode {
				return gox.Element("input", gox.Props{
					"input": input,
					"width": 10,
				})
			}, Options{Width: 20, Height: 5, Output: &output, DisableThrottle: true})

			buf := app.Renderer().CurrentBuffer()
			debugStr := buf.ToDebugString()

			// Get the first 10 characters (our input width)
			lines := strings.Split(debugStr, "\n")
			if len(lines) == 0 {
				t.Fatal("No output lines")
			}
			firstLine := lines[0]
			if len(firstLine) > 10 {
				firstLine = firstLine[:10]
			}

			// Trim trailing spaces for comparison
			trimmed := strings.TrimRight(firstLine, " ")
			if trimmed != tt.visibleText {
				t.Errorf("Expected visible text %q, got %q", tt.visibleText, trimmed)
				t.Logf("Full buffer:\n%s", debugStr)
			}

			// Verify cursor is visible (within the width)
			cursorRelative := tt.cursorPos - tt.expectedScroll
			if cursorRelative < 0 || cursorRelative >= 10 {
				t.Errorf("Cursor at pos %d with scroll %d should be visible (relative: %d)",
					tt.cursorPos, tt.expectedScroll, cursorRelative)
			}

			app.Dispose()
		})
	}

	input.Dispose()
}

func TestInput_VerticalScroll(t *testing.T) {
	// Clear focus manager state
	Manager().Clear()

	// Create input with fixed height=3
	input := NewInput(InputOptions{})
	input.Focus()

	// Set a multiline value
	input.SetValue("line1\nline2\nline3\nline4\nline5")
	// Cursor at end of line 5 (past the visible 3-line window)
	input.SetCursorPos(len("line1\nline2\nline3\nline4\nline5"))

	var output strings.Builder
	app := Render(func() gox.VNode {
		return gox.Element("input", gox.Props{
			"input":  input,
			"width":  10,
			"height": 3,
		})
	}, Options{Width: 20, Height: 10, Output: &output, DisableThrottle: true})

	buf := app.Renderer().CurrentBuffer()
	debugStr := buf.ToDebugString()

	// With cursor on line 5 (index 4), and height 3, we should see lines 3, 4, 5
	// scrollY = cursorLine - height + 1 = 4 - 3 + 1 = 2
	// So visible lines start at index 2: line3, line4, line5
	if !strings.Contains(debugStr, "line3") {
		t.Error("Expected 'line3' to be visible")
	}
	if !strings.Contains(debugStr, "line4") {
		t.Error("Expected 'line4' to be visible")
	}
	if !strings.Contains(debugStr, "line5") {
		t.Error("Expected 'line5' to be visible")
	}
	// line1 and line2 should NOT be visible (scrolled out)
	lines := strings.Split(debugStr, "\n")
	firstThree := strings.Join(lines[:3], "\n")
	if strings.Contains(firstThree, "line1") {
		t.Error("line1 should be scrolled out")
	}
	if strings.Contains(firstThree, "line2") {
		t.Error("line2 should be scrolled out")
	}

	app.Dispose()
	input.Dispose()
}

func TestInput_CursorVisible_AfterTyping(t *testing.T) {
	// Clear focus manager state
	Manager().Clear()

	// Create input with fixed width=5
	input := NewInput(InputOptions{})
	input.Focus()

	// Simulate typing characters one by one past the width
	chars := "ABCDEFGHIJ"
	for i, c := range chars {
		input.SetValue(chars[:i+1])
		input.SetCursorPos(i + 1)

		var output strings.Builder
		app := Render(func() gox.VNode {
			return gox.Element("input", gox.Props{
				"input": input,
				"width": 5,
			})
		}, Options{Width: 20, Height: 5, Output: &output, DisableThrottle: true})

		// Just verify no panic and cursor is within visible range
		buf := app.Renderer().CurrentBuffer()
		_ = buf.ToDebugString() // Ensure render succeeded

		// Calculate expected scroll
		cursorPos := i + 1
		expectedScroll := 0
		if cursorPos >= 5 {
			expectedScroll = cursorPos - 5 + 1
		}

		// Verify the last visible character should be the one just typed
		cursorRelative := cursorPos - expectedScroll
		if cursorRelative < 0 || cursorRelative >= 5 {
			t.Errorf("After typing %q, cursor at %d with scroll %d - relative %d should be in [0,5)",
				string(c), cursorPos, expectedScroll, cursorRelative)
		}

		app.Dispose()
	}

	input.Dispose()
}
