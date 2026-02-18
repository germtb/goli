package goli

import (
	"strings"
	"testing"

	"github.com/germtb/gox"
)

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\x1b[32mhello\x1b[0m", "hello"},
		{"\x1b[1;31mERROR\x1b[0m: something", "ERROR: something"},
		{"\x1b[38;5;196mred\x1b[0m", "red"},
		{"\x1b[38;2;255;0;0mrgb\x1b[0m", "rgb"},
		{"no escape codes here", "no escape codes here"},
		{"", ""},
	}

	for _, tt := range tests {
		got := StripAnsi(tt.input)
		if got != tt.expected {
			t.Errorf("StripAnsi(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestRuneWidthWithAnsi(t *testing.T) {
	// "hello" is 5 chars wide regardless of ANSI codes
	plain := RuneWidth("hello")
	ansi := RuneWidth("\x1b[32mhello\x1b[0m")
	if plain != ansi {
		t.Errorf("RuneWidth with ANSI = %d, without = %d, should be equal", ansi, plain)
	}
	if plain != 5 {
		t.Errorf("RuneWidth('hello') = %d, want 5", plain)
	}
}

func TestParseAnsiLine(t *testing.T) {
	base := Style{}

	// No ANSI — single segment
	segs := ParseAnsiLine("hello", base)
	if len(segs) != 1 || segs[0].Text != "hello" {
		t.Errorf("plain text: got %d segments, text=%q", len(segs), segs[0].Text)
	}

	// Green text + reset
	segs = ParseAnsiLine("\x1b[32mhello\x1b[0m world", base)
	if len(segs) != 2 {
		t.Fatalf("green+reset: got %d segments, want 2", len(segs))
	}
	if segs[0].Text != "hello" {
		t.Errorf("seg[0].Text = %q, want 'hello'", segs[0].Text)
	}
	if segs[0].Style.Color != ColorGreen {
		t.Errorf("seg[0].Color = %d, want ColorGreen(%d)", segs[0].Style.Color, ColorGreen)
	}
	if segs[1].Text != " world" {
		t.Errorf("seg[1].Text = %q, want ' world'", segs[1].Text)
	}
	if segs[1].Style.Color != base.Color {
		t.Errorf("seg[1].Color = %d, want base(%d)", segs[1].Style.Color, base.Color)
	}

	// Bold
	segs = ParseAnsiLine("\x1b[1mbold\x1b[0m", base)
	if len(segs) != 1 {
		t.Fatalf("bold: got %d segments, want 1", len(segs))
	}
	if !segs[0].Style.Bold {
		t.Error("expected Bold=true")
	}

	// Combined: bold red
	segs = ParseAnsiLine("\x1b[1;31mtext\x1b[0m", base)
	if len(segs) != 1 {
		t.Fatalf("bold+red: got %d segments", len(segs))
	}
	if !segs[0].Style.Bold || segs[0].Style.Color != ColorRed {
		t.Errorf("expected Bold+Red, got Bold=%v Color=%d", segs[0].Style.Bold, segs[0].Style.Color)
	}
}

func TestContainsAnsi(t *testing.T) {
	if ContainsAnsi("hello") {
		t.Error("plain text should not contain ANSI")
	}
	if !ContainsAnsi("\x1b[32mhello\x1b[0m") {
		t.Error("colored text should contain ANSI")
	}
}

func ansiNode(text string) gox.VNode {
	return gox.VNode{
		Type:  "ansi",
		Props: gox.Props{},
		Children: []gox.VNode{
			CreateTextNode(text),
		},
	}
}

func TestAnsiElement_PlainText(t *testing.T) {
	node := boxNode(
		gox.Props{"width": 20, "height": 1},
		ansiNode("hello"),
	)

	result := sprintWith(node, PrintOptions{Width: 20, Height: 1})
	if !strings.Contains(result, "hello") {
		t.Errorf("ansi element should render plain text, got: %q", result)
	}
}

func TestAnsiElement_ColoredText(t *testing.T) {
	// Green "hello" via ANSI
	node := boxNode(
		gox.Props{"width": 20, "height": 1},
		ansiNode("\x1b[32mhello\x1b[0m"),
	)

	result := sprintWith(node, PrintOptions{Width: 20, Height: 1})

	// Should contain "hello" with green ANSI code
	if !strings.Contains(result, "hello") {
		t.Errorf("ansi element should render text content, got: %q", result)
	}
	// The green color should be output as ANSI (goli renders Style.Color as ANSI)
	if !strings.Contains(result, "\x1b[32m") {
		t.Errorf("ansi element should preserve green color, got: %q", result)
	}
}

func TestAnsiElement_CorrectWidth(t *testing.T) {
	// "\x1b[32mhi\x1b[0m" is 2 visible chars wide, not 10+
	node := ansiNode("\x1b[32mhi\x1b[0m")
	w, h := measureAnsi(node, nil)
	if w != 2 {
		t.Errorf("ansi element width should be 2, got %d", w)
	}
	if h != 1 {
		t.Errorf("ansi element height should be 1, got %d", h)
	}
}

func TestAnsiElement_MultipleColors(t *testing.T) {
	// Red "err" + reset + " ok"
	node := boxNode(
		gox.Props{"width": 20, "height": 1},
		ansiNode("\x1b[31merr\x1b[0m ok"),
	)

	result := sprintWith(node, PrintOptions{Width: 20, Height: 1})
	if !strings.Contains(result, "err") || !strings.Contains(result, "ok") {
		t.Errorf("ansi element should render all segments, got: %q", result)
	}
}

func TestAnsiElement_BoldAndColor(t *testing.T) {
	node := boxNode(
		gox.Props{"width": 20, "height": 1},
		ansiNode("\x1b[1;31mERROR\x1b[0m"),
	)

	result := sprintWith(node, PrintOptions{Width: 20, Height: 1})
	if !strings.Contains(result, "ERROR") {
		t.Errorf("ansi element should render bold+color text, got: %q", result)
	}
	// Should have bold
	if !strings.Contains(result, boldStr) {
		t.Errorf("ansi element should output bold ANSI, got: %q", result)
	}
}
