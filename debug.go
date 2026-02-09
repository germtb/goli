package goli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// DebugLayout prints the layout tree to stdout for debugging.
func DebugLayout(box *LayoutBox) {
	FprintLayout(os.Stdout, box)
}

// SprintLayout returns the layout tree as a string for debugging.
func SprintLayout(box *LayoutBox) string {
	var sb strings.Builder
	FprintLayout(&sb, box)
	return sb.String()
}

// FprintLayout writes the layout tree to the given writer for debugging.
func FprintLayout(w io.Writer, box *LayoutBox) {
	fprintLayoutIndent(w, box, 0)
}

func fprintLayoutIndent(w io.Writer, box *LayoutBox, depth int) {
	indent := strings.Repeat("  ", depth)

	// Determine node type name
	nodeType := "unknown"
	if box.Node.Type != nil {
		if s, ok := TypeString(box.Node); ok {
			nodeType = s
		} else {
			nodeType = fmt.Sprintf("%T", box.Node.Type)
		}
	}

	// Position and dimensions
	line := fmt.Sprintf("%s%s x=%d y=%d w=%d h=%d", indent, nodeType, box.X, box.Y, box.Width, box.Height)

	// Show inner dimensions when they differ from outer
	if box.InnerX != box.X || box.InnerY != box.Y || box.InnerWidth != box.Width || box.InnerHeight != box.Height {
		line += fmt.Sprintf(" inner(x=%d y=%d w=%d h=%d)", box.InnerX, box.InnerY, box.InnerWidth, box.InnerHeight)
	}

	fmt.Fprintln(w, line)

	for _, child := range box.Children {
		fprintLayoutIndent(w, child, depth+1)
	}
}
