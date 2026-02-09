package goli

import (
	"strings"
	"testing"

	"github.com/germtb/gox"
)

func textNode(text string) gox.VNode {
	return gox.VNode{
		Type:  "text",
		Props: gox.Props{},
		Children: []gox.VNode{
			CreateTextNode(text),
		},
	}
}

func styledTextNode(text string, style Style) gox.VNode {
	return gox.VNode{
		Type:  "text",
		Props: gox.Props{"style": style},
		Children: []gox.VNode{
			CreateTextNode(text),
		},
	}
}

func boxNode(props gox.Props, children ...gox.VNode) gox.VNode {
	return gox.VNode{
		Type:     "box",
		Props:    props,
		Children: children,
	}
}

func TestSprint(t *testing.T) {
	node := boxNode(
		gox.Props{"width": 10, "height": 1},
		textNode("Hello"),
	)

	result := sprintWith(node, PrintOptions{Width: 10, Height: 1})

	// Should contain "Hello" followed by spaces to fill width
	if !strings.Contains(result, "Hello") {
		t.Errorf("Sprint output should contain 'Hello', got: %q", result)
	}
	// Should end with a newline
	if !strings.HasSuffix(result, "\n") {
		t.Errorf("Sprint output should end with newline, got: %q", result)
	}
}

func TestSprint_WithStyles(t *testing.T) {
	node := boxNode(
		gox.Props{"width": 10, "height": 1},
		styledTextNode("Bold", Style{Bold: true}),
	)

	result := sprintWith(node, PrintOptions{Width: 10, Height: 1})

	// Should contain ANSI bold code
	if !strings.Contains(result, boldStr) {
		t.Errorf("Sprint output should contain bold ANSI code, got: %q", result)
	}
	if !strings.Contains(result, "Bold") {
		t.Errorf("Sprint output should contain 'Bold', got: %q", result)
	}
	// Should contain reset
	if !strings.Contains(result, resetStr) {
		t.Errorf("Sprint output should contain reset ANSI code, got: %q", result)
	}
}

func TestSprint_WideCharacters(t *testing.T) {
	node := boxNode(
		gox.Props{"width": 10, "height": 1},
		textNode("Hi🌐"),
	)

	result := sprintWith(node, PrintOptions{Width: 10, Height: 1})

	if !strings.Contains(result, "Hi🌐") {
		t.Errorf("Sprint output should contain 'Hi🌐', got: %q", result)
	}
}

func TestSprint_MultiLine(t *testing.T) {
	node := boxNode(
		gox.Props{"width": 10, "height": 3, "direction": "column"},
		textNode("Line1"),
		textNode("Line2"),
		textNode("Line3"),
	)

	result := sprintWith(node, PrintOptions{Width: 10, Height: 3})

	if !strings.Contains(result, "Line1") {
		t.Errorf("Sprint output should contain 'Line1', got: %q", result)
	}
	if !strings.Contains(result, "Line2") {
		t.Errorf("Sprint output should contain 'Line2', got: %q", result)
	}
	if !strings.Contains(result, "Line3") {
		t.Errorf("Sprint output should contain 'Line3', got: %q", result)
	}

	// Should have newlines between lines (not \r\n, and no cursor positioning)
	if strings.Contains(result, "\r\n") {
		t.Errorf("Sprint output should use \\n not \\r\\n, got: %q", result)
	}
	if strings.Contains(result, MoveCursor(0, 0)) {
		t.Errorf("Sprint output should not contain cursor positioning, got: %q", result)
	}
}

func TestSprint_TrimsEmptyRows(t *testing.T) {
	// Create a box with height 10 but only 1 line of content
	node := boxNode(
		gox.Props{"width": 10, "height": 10},
		textNode("Hi"),
	)

	result := sprintWith(node, PrintOptions{Width: 10, Height: 10})

	// Strip the trailing newline added by Fprint
	trimmed := strings.TrimSuffix(result, "\n")

	// Should NOT have 9 trailing blank lines - only the content row
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 1 {
		t.Errorf("Sprint should trim empty rows, got %d lines: %q", len(lines), result)
	}
}

func TestFprint_CustomDimensions(t *testing.T) {
	node := boxNode(
		gox.Props{"width": 20, "height": 2},
		textNode("Custom"),
	)

	var sb strings.Builder
	Fprint(&sb, node, PrintOptions{Width: 20, Height: 2})
	result := sb.String()

	if !strings.Contains(result, "Custom") {
		t.Errorf("Fprint output should contain 'Custom', got: %q", result)
	}
}

func TestSprint_TallContent_NotTruncated(t *testing.T) {
	// Create 50 lines of content — should all render even if terminal height is small
	children := make([]gox.VNode, 50)
	for i := range children {
		children[i] = textNode("Line")
	}

	node := boxNode(
		gox.Props{"width": 10, "direction": "column"},
		children...,
	)

	// Use a small terminal height to simulate the old truncation issue
	result := sprintWith(node, PrintOptions{Width: 10, Height: 10})

	trimmed := strings.TrimSuffix(result, "\n")
	lines := strings.Split(trimmed, "\n")

	// All 50 lines should be present
	if len(lines) < 50 {
		t.Errorf("expected at least 50 lines in output, got %d", len(lines))
	}
}

// sprintWith is a test helper that renders with explicit options.
func sprintWith(node gox.VNode, opts PrintOptions) string {
	var sb strings.Builder
	Fprint(&sb, node, opts)
	return sb.String()
}
