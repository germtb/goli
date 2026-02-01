// Package cell provides the fundamental Cell type representing a terminal "pixel".
// Each Cell holds a character and its styling attributes.
package goli

// Color represents terminal colors using a compact uint8 representation.
// Values 0-9 are named colors, 10+ reserved for future use.
// RGB colors use a separate type.
type Color uint8

const (
	ColorNone    Color = iota // No color set (transparent)
	ColorDefault              // Terminal default
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

// NameToColor converts a string color name to Color
var NameToColor = map[string]Color{
	"default": ColorDefault,
	"black":   ColorBlack,
	"red":     ColorRed,
	"green":   ColorGreen,
	"yellow":  ColorYellow,
	"blue":    ColorBlue,
	"magenta": ColorMagenta,
	"cyan":    ColorCyan,
	"white":   ColorWhite,
}

// RGB represents a 24-bit true color.
// When used, the Color field should be set to a special marker.
type RGB struct {
	R, G, B uint8
}

// Style holds text styling attributes.
// Uses compact representation: 2 bytes for colors, 1 byte for flags, plus optional RGB.
type Style struct {
	Color         Color
	Background    Color
	Bold          bool
	Dim           bool
	Italic        bool
	Underline     bool
	Inverse       bool
	Strikethrough bool
	// RGB colors (only used when Color/Background need 24-bit)
	ColorRGB      *RGB
	BackgroundRGB *RGB
}

// Cell represents a single "pixel" in the terminal.
// It holds a character and its styling attributes.
type Cell struct {
	Char  rune
	Style Style
}

// EmptyStyle is a Style with no attributes set.
var EmptyStyle = Style{}

// EmptyCell is a Cell with a space character and no styling.
var EmptyCell = Cell{Char: ' ', Style: EmptyStyle}

// New creates a new Cell with the given character and style.
func New(char rune, style Style) Cell {
	return Cell{Char: char, Style: style}
}

// Equal returns true if two Cells are identical.
func (a Cell) Equal(b Cell) bool {
	if a.Char != b.Char {
		return false
	}
	return a.Style.Equal(b.Style)
}

// Equal returns true if two Styles are identical.
func (a Style) Equal(b Style) bool {
	// Compare simple fields first (most likely to differ, fastest to check)
	if a.Color != b.Color || a.Background != b.Background {
		return false
	}
	if a.Bold != b.Bold || a.Dim != b.Dim || a.Italic != b.Italic ||
		a.Underline != b.Underline || a.Inverse != b.Inverse ||
		a.Strikethrough != b.Strikethrough {
		return false
	}
	// Compare RGB if present
	if !rgbEqual(a.ColorRGB, b.ColorRGB) {
		return false
	}
	return rgbEqual(a.BackgroundRGB, b.BackgroundRGB)
}

func rgbEqual(a, b *RGB) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.R == b.R && a.G == b.G && a.B == b.B
}

// HasColor returns true if the style has a foreground color set.
func (s Style) HasColor() bool {
	return s.Color != ColorNone || s.ColorRGB != nil
}

// HasBackground returns true if the style has a background color set.
func (s Style) HasBackground() bool {
	return s.Background != ColorNone || s.BackgroundRGB != nil
}

// Merge creates a new Style by combining two styles.
// The overlay style takes precedence for non-zero values.
func (base Style) Merge(overlay Style) Style {
	result := base

	if overlay.Color != ColorNone {
		result.Color = overlay.Color
		result.ColorRGB = overlay.ColorRGB
	}
	if overlay.Background != ColorNone {
		result.Background = overlay.Background
		result.BackgroundRGB = overlay.BackgroundRGB
	}
	if overlay.Bold {
		result.Bold = true
	}
	if overlay.Dim {
		result.Dim = true
	}
	if overlay.Italic {
		result.Italic = true
	}
	if overlay.Underline {
		result.Underline = true
	}
	if overlay.Inverse {
		result.Inverse = true
	}
	if overlay.Strikethrough {
		result.Strikethrough = true
	}

	return result
}
