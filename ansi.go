// Package ansi provides ANSI escape code generation for terminal output.
package goli

import (
	"strconv"
	"strings"
)

const (
	ESC = "\x1b"
	CSI = ESC + "["
)

// Pre-computed ANSI escape sequences
const (
	csiStr    = "\x1b["
	resetStr  = "\x1b[0m"
	boldStr   = "\x1b[1m"
	dimStr    = "\x1b[2m"
	italicStr = "\x1b[3m"
	underStr  = "\x1b[4m"
	invStr    = "\x1b[7m"
	strikeStr = "\x1b[9m"
)

// MoveCursor returns the ANSI code to move the cursor to (x, y).
// ANSI uses 1-based coordinates.
func MoveCursor(x, y int) string {
	return csiStr + strconv.Itoa(y+1) + ";" + strconv.Itoa(x+1) + "H"
}

// HideCursor returns the ANSI code to hide the cursor.
func HideCursor() string {
	return CSI + "?25l"
}

// ShowCursor returns the ANSI code to show the cursor.
func ShowCursor() string {
	return CSI + "?25h"
}

// ClearScreen returns the ANSI code to clear the screen.
func ClearScreen() string {
	return CSI + "2J" + CSI + "H"
}

// Foreground color ANSI codes indexed by Color
var fgCodes = [...]string{
	ColorNone:    "",
	ColorDefault: "\x1b[39m",
	ColorBlack:   "\x1b[30m",
	ColorRed:     "\x1b[31m",
	ColorGreen:   "\x1b[32m",
	ColorYellow:  "\x1b[33m",
	ColorBlue:    "\x1b[34m",
	ColorMagenta: "\x1b[35m",
	ColorCyan:    "\x1b[36m",
	ColorWhite:   "\x1b[37m",
}

// Background color ANSI codes indexed by Color
var bgCodes = [...]string{
	ColorNone:    "",
	ColorDefault: "\x1b[49m",
	ColorBlack:   "\x1b[40m",
	ColorRed:     "\x1b[41m",
	ColorGreen:   "\x1b[42m",
	ColorYellow:  "\x1b[43m",
	ColorBlue:    "\x1b[44m",
	ColorMagenta: "\x1b[45m",
	ColorCyan:    "\x1b[46m",
	ColorWhite:   "\x1b[47m",
}

// ColorToAnsi converts a Color to ANSI escape code.
func ColorToAnsi(color Color, rgb *RGB, isFg bool) string {
	// Handle RGB first
	if rgb != nil {
		if isFg {
			return csiStr + "38;2;" + strconv.Itoa(int(rgb.R)) + ";" + strconv.Itoa(int(rgb.G)) + ";" + strconv.Itoa(int(rgb.B)) + "m"
		}
		return csiStr + "48;2;" + strconv.Itoa(int(rgb.R)) + ";" + strconv.Itoa(int(rgb.G)) + ";" + strconv.Itoa(int(rgb.B)) + "m"
	}

	// Use pre-computed codes for named colors
	if int(color) < len(fgCodes) {
		if isFg {
			return fgCodes[color]
		}
		return bgCodes[color]
	}
	return ""
}

// StyleToAnsi generates ANSI codes for a style, writing directly to builder.
func StyleToAnsi(style Style, sb *strings.Builder) {
	if style.Bold {
		sb.WriteString(boldStr)
	}
	if style.Dim {
		sb.WriteString(dimStr)
	}
	if style.Italic {
		sb.WriteString(italicStr)
	}
	if style.Underline {
		sb.WriteString(underStr)
	}
	if style.Inverse {
		sb.WriteString(invStr)
	}
	if style.Strikethrough {
		sb.WriteString(strikeStr)
	}
	if style.Color != ColorNone || style.ColorRGB != nil {
		sb.WriteString(ColorToAnsi(style.Color, style.ColorRGB, true))
	}
	if style.Background != ColorNone || style.BackgroundRGB != nil {
		sb.WriteString(ColorToAnsi(style.Background, style.BackgroundRGB, false))
	}
}

// CellRun represents a run of consecutive cells.
type CellRun struct {
	X     int
	Y     int
	Cells []Cell
}

// RunToAnsi renders a run of cells to ANSI, writing directly to builder.
func RunToAnsi(run CellRun, sb *strings.Builder) {
	sb.WriteString(MoveCursor(run.X, run.Y))

	var currentStyle *Style

	for _, c := range run.Cells {
		styleChanged := currentStyle == nil || !currentStyle.Equal(c.Style)

		if styleChanged {
			sb.WriteString(resetStr)
			StyleToAnsi(c.Style, sb)
			styleCopy := c.Style
			currentStyle = &styleCopy
		}

		sb.WriteRune(c.Char)
	}
}

// RunsToAnsi renders all runs to a single ANSI string.
func RunsToAnsi(runs []CellRun) string {
	if len(runs) == 0 {
		return resetStr
	}

	// Pre-allocate: estimate ~20 bytes per cell average
	totalCells := 0
	for _, run := range runs {
		totalCells += len(run.Cells)
	}

	var sb strings.Builder
	sb.Grow(totalCells*20 + len(runs)*15)

	for _, run := range runs {
		RunToAnsi(run, &sb)
	}

	sb.WriteString(resetStr)
	return sb.String()
}

// RunsToAnsiBuilder renders all runs to the provided strings.Builder.
// This avoids allocation when the caller manages the builder.
func RunsToAnsiBuilder(runs []CellRun, sb *strings.Builder) {
	if len(runs) == 0 {
		sb.WriteString(resetStr)
		return
	}

	for _, run := range runs {
		RunToAnsi(run, sb)
	}

	sb.WriteString(resetStr)
}
