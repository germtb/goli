// Package ansi provides ANSI escape code generation for terminal output.
package goli

import (
	"strconv"
	"strings"
)

const (
	ESC = "\x1b"
	CSI = ESC + "["
	OSC = ESC + "]"
	ST  = ESC + "\\" // String Terminator
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
	// OSC 8 hyperlink end
	hyperlinkEnd = "\x1b]8;;\x1b\\"
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
	ColorNone:          "",
	ColorDefault:       "\x1b[39m",
	ColorBlack:         "\x1b[30m",
	ColorRed:           "\x1b[31m",
	ColorGreen:         "\x1b[32m",
	ColorYellow:        "\x1b[33m",
	ColorBlue:          "\x1b[34m",
	ColorMagenta:       "\x1b[35m",
	ColorCyan:          "\x1b[36m",
	ColorWhite:         "\x1b[37m",
	ColorBrightBlack:   "\x1b[90m",
	ColorBrightRed:     "\x1b[91m",
	ColorBrightGreen:   "\x1b[92m",
	ColorBrightYellow:  "\x1b[93m",
	ColorBrightBlue:    "\x1b[94m",
	ColorBrightMagenta: "\x1b[95m",
	ColorBrightCyan:    "\x1b[96m",
	ColorBrightWhite:   "\x1b[97m",
}

// Background color ANSI codes indexed by Color
var bgCodes = [...]string{
	ColorNone:          "",
	ColorDefault:       "\x1b[49m",
	ColorBlack:         "\x1b[40m",
	ColorRed:           "\x1b[41m",
	ColorGreen:         "\x1b[42m",
	ColorYellow:        "\x1b[43m",
	ColorBlue:          "\x1b[44m",
	ColorMagenta:       "\x1b[45m",
	ColorCyan:          "\x1b[46m",
	ColorWhite:         "\x1b[47m",
	ColorBrightBlack:   "\x1b[100m",
	ColorBrightRed:     "\x1b[101m",
	ColorBrightGreen:   "\x1b[102m",
	ColorBrightYellow:  "\x1b[103m",
	ColorBrightBlue:    "\x1b[104m",
	ColorBrightMagenta: "\x1b[105m",
	ColorBrightCyan:    "\x1b[106m",
	ColorBrightWhite:   "\x1b[107m",
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

// HyperlinkStart returns the OSC 8 sequence to start a hyperlink.
func HyperlinkStart(url string) string {
	return "\x1b]8;;" + url + "\x1b\\"
}

// HyperlinkEnd returns the OSC 8 sequence to end a hyperlink.
func HyperlinkEnd() string {
	return hyperlinkEnd
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
	currentHyperlink := ""

	for _, c := range run.Cells {
		styleChanged := currentStyle == nil || !currentStyle.Equal(c.Style)
		hyperlinkChanged := c.Style.HyperlinkURL != currentHyperlink

		// If style changed, we need to reset and reapply everything
		if styleChanged {
			// End current hyperlink before reset (if any)
			if currentHyperlink != "" {
				sb.WriteString(hyperlinkEnd)
			}
			sb.WriteString(resetStr)
			StyleToAnsi(c.Style, sb)
			// Apply new hyperlink after style (if any)
			if c.Style.HyperlinkURL != "" {
				sb.WriteString(HyperlinkStart(c.Style.HyperlinkURL))
			}
			currentHyperlink = c.Style.HyperlinkURL
			styleCopy := c.Style
			currentStyle = &styleCopy
		} else if hyperlinkChanged {
			// Style same but hyperlink changed - just update hyperlink
			if currentHyperlink != "" {
				sb.WriteString(hyperlinkEnd)
			}
			if c.Style.HyperlinkURL != "" {
				sb.WriteString(HyperlinkStart(c.Style.HyperlinkURL))
			}
			currentHyperlink = c.Style.HyperlinkURL
		}

		sb.WriteRune(c.Char)
	}

	// End any open hyperlink
	if currentHyperlink != "" {
		sb.WriteString(hyperlinkEnd)
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

// bufferToAnsiLines renders a CellBuffer to ANSI output suitable for printing.
// Unlike BufferToSequentialAnsi, it uses no cursor positioning and \n line separators.
// Only outputs rows 0..maxRow (inclusive).
func bufferToAnsiLines(buf *CellBuffer, maxRow int) string {
	var sb strings.Builder
	sb.Grow(buf.Width() * (maxRow + 1) * 15)

	var currentStyle *Style
	currentHyperlink := ""

	for y := 0; y <= maxRow; y++ {
		if y > 0 {
			if currentStyle != nil {
				sb.WriteString(resetStr)
				currentStyle = nil
			}
			if currentHyperlink != "" {
				sb.WriteString(hyperlinkEnd)
				currentHyperlink = ""
			}
			sb.WriteByte('\n')
		}

		for x := 0; x < buf.Width(); x++ {
			c := buf.Get(x, y)

			styleChanged := currentStyle == nil || !currentStyle.Equal(c.Style)
			hyperlinkChanged := c.Style.HyperlinkURL != currentHyperlink

			if styleChanged {
				if currentHyperlink != "" {
					sb.WriteString(hyperlinkEnd)
				}
				sb.WriteString(resetStr)
				StyleToAnsi(c.Style, &sb)
				if c.Style.HyperlinkURL != "" {
					sb.WriteString(HyperlinkStart(c.Style.HyperlinkURL))
				}
				currentHyperlink = c.Style.HyperlinkURL
				styleCopy := c.Style
				currentStyle = &styleCopy
			} else if hyperlinkChanged {
				if currentHyperlink != "" {
					sb.WriteString(hyperlinkEnd)
				}
				if c.Style.HyperlinkURL != "" {
					sb.WriteString(HyperlinkStart(c.Style.HyperlinkURL))
				}
				currentHyperlink = c.Style.HyperlinkURL
			}

			sb.WriteRune(c.Char)
		}
	}

	if currentHyperlink != "" {
		sb.WriteString(hyperlinkEnd)
	}
	sb.WriteString(resetStr)

	return sb.String()
}

// BufferToSequentialAnsi renders a CellBuffer line-by-line with newlines.
// This is used for overflow content where ANSI cursor positioning doesn't work.
// Outputs from cursor position (0,0) downward, using newlines to advance rows.
func BufferToSequentialAnsi(buf *CellBuffer) string {
	var sb strings.Builder
	// Estimate ~15 bytes per cell
	sb.Grow(buf.Width() * buf.Height() * 15)

	// Move to home position first
	sb.WriteString(MoveCursor(0, 0))

	var currentStyle *Style
	currentHyperlink := ""

	for y := 0; y < buf.Height(); y++ {
		if y > 0 {
			// End any styles before newline, then newline, then continue
			if currentStyle != nil {
				sb.WriteString(resetStr)
				currentStyle = nil
			}
			if currentHyperlink != "" {
				sb.WriteString(hyperlinkEnd)
				currentHyperlink = ""
			}
			sb.WriteString("\r\n")
		}

		for x := 0; x < buf.Width(); x++ {
			c := buf.Get(x, y)

			styleChanged := currentStyle == nil || !currentStyle.Equal(c.Style)
			hyperlinkChanged := c.Style.HyperlinkURL != currentHyperlink

			// If style changed, we need to reset and reapply everything
			if styleChanged {
				// End current hyperlink before reset (if any)
				if currentHyperlink != "" {
					sb.WriteString(hyperlinkEnd)
				}
				sb.WriteString(resetStr)
				StyleToAnsi(c.Style, &sb)
				// Apply new hyperlink after style (if any)
				if c.Style.HyperlinkURL != "" {
					sb.WriteString(HyperlinkStart(c.Style.HyperlinkURL))
				}
				currentHyperlink = c.Style.HyperlinkURL
				styleCopy := c.Style
				currentStyle = &styleCopy
			} else if hyperlinkChanged {
				// Style same but hyperlink changed - just update hyperlink
				if currentHyperlink != "" {
					sb.WriteString(hyperlinkEnd)
				}
				if c.Style.HyperlinkURL != "" {
					sb.WriteString(HyperlinkStart(c.Style.HyperlinkURL))
				}
				currentHyperlink = c.Style.HyperlinkURL
			}

			sb.WriteRune(c.Char)
		}
	}

	// End any open hyperlink and reset
	if currentHyperlink != "" {
		sb.WriteString(hyperlinkEnd)
	}
	sb.WriteString(resetStr)

	return sb.String()
}
