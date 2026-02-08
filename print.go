package goli

import (
	"io"
	"os"
	"strings"

	"github.com/germtb/gox"
)

// PrintOptions configures dimensions for Fprint.
type PrintOptions struct {
	Width  int // 0 = auto-detect terminal width (default 80)
	Height int // 0 = auto-detect terminal height (default 24)
}

// Print renders a VNode tree to stdout with ANSI styling.
func Print(node gox.VNode) {
	Fprint(os.Stdout, node, PrintOptions{})
}

// Sprint renders a VNode tree to a string with ANSI styling.
// Width/height auto-detected from terminal (falls back to 80x24).
func Sprint(node gox.VNode) string {
	var sb strings.Builder
	Fprint(&sb, node, PrintOptions{})
	return sb.String()
}

// Fprint renders a VNode tree to a writer with ANSI styling.
func Fprint(w io.Writer, node gox.VNode, opts PrintOptions) {
	width := opts.Width
	height := opts.Height

	if width == 0 || height == 0 {
		tw, th, err := GetSize(Stdout())
		if err == nil {
			if width == 0 {
				width = tw
			}
			if height == 0 {
				height = th
			}
		}
	}
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	// Expand functional components
	expanded := Expand(node)

	// Compute layout
	layoutBox := ComputeLayout(expanded, LayoutContext{
		X:      0,
		Y:      0,
		Width:  width,
		Height: height,
	})

	// Use content height from layout, capped to available height
	contentHeight := min(layoutBox.Height, height)
	if contentHeight <= 0 {
		return
	}

	// Render to buffer
	buf := NewCellBuffer(width, contentHeight)
	RenderToBuffer(layoutBox, buf, nil)

	// Find last non-empty row
	lastRow := 0
	for y := contentHeight - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			c := buf.Get(x, y)
			if c.Char != ' ' || c.Style != EmptyStyle {
				lastRow = y
				goto found
			}
		}
	}
found:

	// Convert to ANSI and write
	output := bufferToAnsiLines(buf, lastRow)
	io.WriteString(w, output)
	io.WriteString(w, "\n")
}
