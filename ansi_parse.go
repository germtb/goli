package goli

import (
	"strings"
)

// ContainsAnsi returns true if the string contains ANSI escape sequences.
func ContainsAnsi(s string) bool {
	return strings.Contains(s, "\x1b[")
}

// StripAnsi removes ANSI escape sequences from a string,
// returning only the visible text content.
func StripAnsi(s string) string {
	if !ContainsAnsi(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// CSI sequence: skip ESC[ then params until final byte (0x40-0x7E)
			i += 2
			for i < len(s) && !(s[i] >= 0x40 && s[i] <= 0x7E) {
				i++
			}
			if i < len(s) {
				i++ // skip final byte
			}
		} else if s[i] == '\x1b' {
			// Other escape: skip ESC + next byte
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// AnsiSegment represents a piece of text with associated style from ANSI codes.
type AnsiSegment struct {
	Text  string
	Style Style
}

// ParseAnsiLine parses a line containing ANSI escape codes into styled segments.
// The baseStyle is the element's own style; ANSI styles are merged on top.
func ParseAnsiLine(line string, baseStyle Style) []AnsiSegment {
	if !ContainsAnsi(line) {
		return []AnsiSegment{{Text: line, Style: baseStyle}}
	}

	var segments []AnsiSegment
	current := baseStyle
	var text strings.Builder
	i := 0

	for i < len(line) {
		if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
			// Flush accumulated text
			if text.Len() > 0 {
				segments = append(segments, AnsiSegment{Text: text.String(), Style: current})
				text.Reset()
			}

			// Parse CSI sequence
			i += 2 // skip ESC[
			paramStart := i
			for i < len(line) && !(line[i] >= 0x40 && line[i] <= 0x7E) {
				i++
			}
			if i < len(line) {
				if line[i] == 'm' {
					// SGR sequence — apply style changes
					applySGR(line[paramStart:i], &current, baseStyle)
				}
				i++ // skip final byte
			}
		} else if line[i] == '\x1b' {
			// Non-CSI escape: skip
			i += 2
		} else {
			text.WriteByte(line[i])
			i++
		}
	}

	if text.Len() > 0 {
		segments = append(segments, AnsiSegment{Text: text.String(), Style: current})
	}

	return segments
}

// applySGR applies SGR (Select Graphic Rendition) parameters to a style.
func applySGR(paramStr string, style *Style, baseStyle Style) {
	if paramStr == "" {
		// ESC[m is equivalent to ESC[0m (reset)
		*style = baseStyle
		return
	}

	params := parseSGRParams(paramStr)
	i := 0
	for i < len(params) {
		p := params[i]
		switch {
		case p == 0:
			*style = baseStyle
		case p == 1:
			style.Bold = true
		case p == 2:
			style.Dim = true
		case p == 3:
			style.Italic = true
		case p == 4:
			style.Underline = true
		case p == 7:
			style.Inverse = true
		case p == 9:
			style.Strikethrough = true
		case p == 22:
			style.Bold = false
			style.Dim = false
		case p == 23:
			style.Italic = false
		case p == 24:
			style.Underline = false
		case p == 27:
			style.Inverse = false
		case p == 29:
			style.Strikethrough = false

		// Foreground colors 30-37
		case p >= 30 && p <= 37:
			style.Color = Color(ColorBlack + Color(p-30))
			style.ColorRGB = nil
		case p == 39:
			style.Color = baseStyle.Color
			style.ColorRGB = baseStyle.ColorRGB

		// Background colors 40-47
		case p >= 40 && p <= 47:
			style.Background = Color(ColorBlack + Color(p-40))
			style.BackgroundRGB = nil
		case p == 49:
			style.Background = baseStyle.Background
			style.BackgroundRGB = baseStyle.BackgroundRGB

		// Bright foreground 90-97
		case p >= 90 && p <= 97:
			style.Color = Color(ColorBrightBlack + Color(p-90))
			style.ColorRGB = nil

		// Bright background 100-107
		case p >= 100 && p <= 107:
			style.Background = Color(ColorBrightBlack + Color(p-100))
			style.BackgroundRGB = nil

		// Extended foreground: 38;5;N (256-color) or 38;2;R;G;B
		case p == 38:
			if i+1 < len(params) && params[i+1] == 5 && i+2 < len(params) {
				style.Color, style.ColorRGB = color256toGoli(params[i+2])
				i += 2
			} else if i+1 < len(params) && params[i+1] == 2 && i+4 < len(params) {
				style.ColorRGB = &RGB{R: uint8(params[i+2]), G: uint8(params[i+3]), B: uint8(params[i+4])}
				style.Color = ColorNone
				i += 4
			}

		// Extended background: 48;5;N or 48;2;R;G;B
		case p == 48:
			if i+1 < len(params) && params[i+1] == 5 && i+2 < len(params) {
				style.Background, style.BackgroundRGB = color256toGoli(params[i+2])
				i += 2
			} else if i+1 < len(params) && params[i+1] == 2 && i+4 < len(params) {
				style.BackgroundRGB = &RGB{R: uint8(params[i+2]), G: uint8(params[i+3]), B: uint8(params[i+4])}
				style.Background = ColorNone
				i += 4
			}
		}
		i++
	}
}

// parseSGRParams splits a semicolon-separated parameter string into integers.
func parseSGRParams(s string) []int {
	var params []int
	n := 0
	hasDigit := false
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
			hasDigit = true
		} else if s[i] == ';' {
			params = append(params, n)
			n = 0
			hasDigit = false
		}
	}
	if hasDigit {
		params = append(params, n)
	}
	return params
}

// color256toGoli maps a 256-color index to a goli Color and optional RGB.
func color256toGoli(n int) (Color, *RGB) {
	switch {
	case n >= 0 && n <= 7:
		return Color(ColorBlack + Color(n)), nil
	case n >= 8 && n <= 15:
		return Color(ColorBrightBlack + Color(n-8)), nil
	case n >= 16 && n <= 231:
		// 6x6x6 color cube
		n -= 16
		b := n % 6
		g := (n / 6) % 6
		r := n / 36
		return ColorNone, &RGB{R: uint8(r * 51), G: uint8(g * 51), B: uint8(b * 51)}
	case n >= 232 && n <= 255:
		// Grayscale
		v := uint8((n-232)*10 + 8)
		return ColorNone, &RGB{R: v, G: v, B: v}
	}
	return ColorNone, nil
}
