// Package goli provides a button primitive for interactive UI.
package goli

import (
	"strings"

	"github.com/germtb/goli/signals"
	"github.com/germtb/gox"
)

// ButtonCornerStyle specifies the button corner appearance.
type ButtonCornerStyle string

const (
	ButtonCornerNone  ButtonCornerStyle = "none"
	ButtonCornerPill  ButtonCornerStyle = "pill"  // ▐ text ▌ - half blocks
	ButtonCornerRound ButtonCornerStyle = "round" //  text  - Nerd Font
	ButtonCornerArrow ButtonCornerStyle = "arrow" //  text  - Nerd Font
	ButtonCornerPixel ButtonCornerStyle = "pixel" // ▙ text ▟ - quadrant blocks
)

// ButtonCornerChars holds the characters for button corners.
type ButtonCornerChars struct {
	Left  rune
	Right rune
}

// ButtonCornerCharSets for different button styles.
// All use the button's background color as foreground for a shaped effect.
var ButtonCornerCharSets = map[ButtonCornerStyle]ButtonCornerChars{
	ButtonCornerPill:  {Left: '▐', Right: '▌'},           // Half blocks
	ButtonCornerRound: {Left: '\uE0B6', Right: '\uE0B4'}, // Nerd Font round
	ButtonCornerArrow: {Left: '\uE0B2', Right: '\uE0B0'}, // Nerd Font arrow
	ButtonCornerPixel: {Left: '▟', Right: '▙'},           // Quadrant blocks
}

// GetButtonCornerStyle normalizes corner prop to ButtonCornerStyle.
func GetButtonCornerStyle(corner any) ButtonCornerStyle {
	if corner == nil {
		return ButtonCornerNone
	}

	switch v := corner.(type) {
	case bool:
		if v {
			return ButtonCornerPill
		}
		return ButtonCornerNone
	case string:
		return ButtonCornerStyle(v)
	case ButtonCornerStyle:
		return v
	default:
		return ButtonCornerNone
	}
}

func init() {
	RegisterIntrinsic("button", &IntrinsicHandler{
		Measure:       measureButton,
		Layout:        layoutButton,
		Render:        RenderButtonToBuffer,
		RenderLogical: RenderButtonToLogicalBuffer,
	})
}

// ButtonOptions configures button creation.
type ButtonOptions struct {
	// OnClick is called when the button is activated (Enter/Space).
	OnClick func()
	// OnKeypress is a custom key handler (called before default handling).
	OnKeypress func(key string) bool
	// DisableFocus disables focus management registration (default: false, meaning focusable by default).
	DisableFocus bool
}

// Button represents a clickable button component.
type Button struct {
	focused    signals.Accessor[bool]
	setFocused signals.Setter[bool]

	onClick        func()
	onKeypress     func(key string) bool
	shouldRegister bool
	registered     bool
}

// NewButton creates a new button.
func NewButton(opts ButtonOptions) *Button {
	focused, setFocused := signals.CreateSignal(false)

	shouldRegister := true
	if opts.DisableFocus {
		shouldRegister = false
	}

	b := &Button{
		focused:        focused,
		setFocused:     setFocused,
		onClick:        opts.OnClick,
		onKeypress:     opts.OnKeypress,
		shouldRegister: shouldRegister,
	}

	if shouldRegister {
		Register(b)
		b.registered = true
	}

	return b
}

// Focused returns whether the button is focused.
func (b *Button) Focused() bool {
	return b.focused()
}

// Focus gives focus to this button.
func (b *Button) Focus() {
	RequestFocus(b)
}

// Blur removes focus from this button.
func (b *Button) Blur() {
	RequestBlur(b)
}

// SetFocused sets the focused state (called by focus manager).
func (b *Button) SetFocused(f bool) {
	b.setFocused(f)
}

// Dispose unregisters from the focus manager.
func (b *Button) Dispose() {
	if b.registered {
		Unregister(b)
		b.registered = false
	}
}

// HandleKey processes a key press.
// Returns true if the key was consumed.
func (b *Button) HandleKey(key string) bool {
	if !b.focused() {
		return false
	}

	// Custom handler first
	if b.onKeypress != nil {
		if b.onKeypress(key) {
			return true
		}
	}

	// Handle Enter/Space to activate button
	switch key {
	case Enter, EnterLF, Space:
		if b.onClick != nil {
			b.onClick()
		}
		return true
	}

	return false
}

// Click programmatically triggers the button's onClick handler.
func (b *Button) Click() {
	if b.onClick != nil {
		b.onClick()
	}
}

// Button measure/layout/render functions

func measureButton(node gox.VNode, ctx *LayoutContext) (int, int) {
	padding := NormalizeSpacing(node.Props["padding"])

	// Measure children content (typically just text)
	contentWidth := 0
	contentHeight := 0

	relativeChildren := FilterRelativeChildren(node)
	for _, c := range relativeChildren {
		w, h := MeasureNode(c)
		if w > contentWidth {
			contentWidth = w
		}
		if h > contentHeight {
			contentHeight = h
		}
	}

	totalWidth := contentWidth + padding.Left + padding.Right
	totalHeight := contentHeight + padding.Top + padding.Bottom

	explicitWidth := GetIntProp(node.Props, "width", -1)
	explicitHeight := GetIntProp(node.Props, "height", -1)
	minWidth := GetIntProp(node.Props, "minWidth", 0)
	minHeight := GetIntProp(node.Props, "minHeight", 0)

	finalWidth := totalWidth
	if explicitWidth >= 0 {
		finalWidth = explicitWidth
	}
	if finalWidth < minWidth {
		finalWidth = minWidth
	}

	finalHeight := totalHeight
	if explicitHeight >= 0 {
		finalHeight = explicitHeight
	}
	if finalHeight < minHeight {
		finalHeight = minHeight
	}

	return finalWidth, finalHeight
}

func layoutButton(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	padding := NormalizeSpacing(node.Props["padding"])
	margin := NormalizeSpacing(node.Props["margin"])

	// Calculate button dimensions
	measuredW, measuredH := measureButton(node, nil)
	buttonWidth := GetIntProp(node.Props, "width", -1)
	if buttonWidth < 0 {
		buttonWidth = min(measuredW, availWidth-margin.Left-margin.Right)
	}
	buttonHeight := GetIntProp(node.Props, "height", -1)
	if buttonHeight < 0 {
		buttonHeight = measuredH
	}

	// Button position (respecting margin)
	buttonX := ctx.X + margin.Left
	buttonY := ctx.Y + margin.Top

	// Inner content area
	innerX := buttonX + padding.Left
	innerY := buttonY + padding.Top
	innerWidth := buttonWidth - padding.Left - padding.Right
	innerHeight := buttonHeight - padding.Top - padding.Bottom

	// Layout children
	relativeChildren := FilterRelativeChildren(node)
	childBoxes := make([]*LayoutBox, 0, len(relativeChildren))
	childY := innerY
	for _, c := range relativeChildren {
		childBox := LayoutNode(c, LayoutContext{
			X:      innerX,
			Y:      childY,
			Width:  innerWidth,
			Height: innerHeight,
		})
		childBoxes = append(childBoxes, childBox.Box)
		childY += childBox.Box.Height
	}

	return &LayoutBox{
		X:           buttonX,
		Y:           buttonY,
		Width:       buttonWidth,
		Height:      buttonHeight,
		InnerX:      innerX,
		InnerY:      innerY,
		InnerWidth:  innerWidth,
		InnerHeight: innerHeight,
		Node:        node,
		Children:    childBoxes,
		ZIndex:      GetIntProp(node.Props, "zIndex", 0),
	}
}

// RenderButtonToBuffer renders a button to a CellBuffer.
func RenderButtonToBuffer(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	node := box.Node
	x, y, width, height := box.X, box.Y, box.Width, box.Height

	buttonPrim := node.Props["button"]
	baseStyle := GetStyle(node.Props)
	focusedStyle := getStyleProp(node.Props, "focusedStyle", Style{Inverse: true})
	cornerStyle := GetButtonCornerStyle(node.Props["corners"])

	isFocused := false
	if btn, ok := buttonPrim.(interface{ Focused() bool }); ok {
		isFocused = btn.Focused()
	}

	computedStyle := baseStyle
	if isFocused {
		computedStyle = baseStyle.Merge(focusedStyle)
	}

	chars, hasCorners := ButtonCornerCharSets[cornerStyle]

	// Fill background (excluding corner positions)
	if computedStyle.HasBackground() || isFocused {
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				// Skip corner positions
				if hasCorners && dy == 0 && (dx == 0 || dx == width-1) {
					continue
				}
				cellX, cellY := x+dx, y+dy
				if IsInClip(cellX, cellY, clip) {
					buf.Set(cellX, cellY, New(' ', computedStyle))
				}
			}
		}
	}

	// Render corners: bg color becomes fg for shaped effect
	if hasCorners {
		cornerFg := computedStyle.Background
		if cornerFg == ColorNone {
			cornerFg = ColorWhite
		}
		cornerDrawStyle := Style{Color: cornerFg}

		if IsInClip(x, y, clip) {
			buf.Set(x, y, New(chars.Left, cornerDrawStyle))
		}
		rightX := x + width - 1
		if IsInClip(rightX, y, clip) {
			buf.Set(rightX, y, New(chars.Right, cornerDrawStyle))
		}
	}

	// Render children with the computed style
	for _, childBox := range box.Children {
		renderButtonChild(childBox, buf, clip, computedStyle)
	}
}

// renderButtonChild renders a button's child with inherited style.
func renderButtonChild(box *LayoutBox, buf *CellBuffer, clip *ClipRegion, parentStyle Style) {
	if box == nil {
		return
	}

	node := box.Node
	x, y := box.X, box.Y

	// Handle text nodes with inherited style
	if IsTextNode(node) {
		text, _ := GetTextContent(node)
		lines := strings.Split(text, "\n")

		for lineIdx, line := range lines {
			lineY := y + lineIdx
			if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
				continue
			}

			charX := x
			for _, char := range line {
				if IsInClip(charX, lineY, clip) {
					buf.SetCharMerge(charX, lineY, char, parentStyle)
				}
				charX++
			}
		}
		return
	}

	// For other children, render normally
	RenderToBuffer(box, buf, clip)
}

// RenderButtonToLogicalBuffer renders a button to a LogicalBuffer.
func RenderButtonToLogicalBuffer(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	node := box.Node
	x, y, width, height := box.X, box.Y, box.Width, box.Height

	buttonPrim := node.Props["button"]
	baseStyle := GetStyle(node.Props)
	focusedStyle := getStyleProp(node.Props, "focusedStyle", Style{Inverse: true})
	cornerStyle := GetButtonCornerStyle(node.Props["corners"])

	isFocused := false
	if btn, ok := buttonPrim.(interface{ Focused() bool }); ok {
		isFocused = btn.Focused()
	}

	computedStyle := baseStyle
	if isFocused {
		computedStyle = baseStyle.Merge(focusedStyle)
	}

	chars, hasCorners := ButtonCornerCharSets[cornerStyle]

	// Fill background (excluding corner positions)
	if computedStyle.HasBackground() || isFocused {
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				// Skip corner positions
				if hasCorners && dy == 0 && (dx == 0 || dx == width-1) {
					continue
				}
				cellX, cellY := x+dx, y+dy
				if IsInClip(cellX, cellY, clip) {
					buf.Set(cellX, cellY, New(' ', computedStyle))
				}
			}
		}
	}

	// Render corners: bg color becomes fg for shaped effect
	if hasCorners {
		cornerFg := computedStyle.Background
		if cornerFg == ColorNone {
			cornerFg = ColorWhite
		}
		cornerDrawStyle := Style{Color: cornerFg}

		if IsInClip(x, y, clip) {
			buf.Set(x, y, New(chars.Left, cornerDrawStyle))
		}
		rightX := x + width - 1
		if IsInClip(rightX, y, clip) {
			buf.Set(rightX, y, New(chars.Right, cornerDrawStyle))
		}
	}

	// Render children with the computed style
	for _, childBox := range box.Children {
		renderButtonChildLogical(childBox, buf, clip, computedStyle)
	}
}

// renderButtonChildLogical renders a button's child with inherited style to LogicalBuffer.
func renderButtonChildLogical(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion, parentStyle Style) {
	if box == nil {
		return
	}

	node := box.Node
	x, y := box.X, box.Y

	// Handle text nodes with inherited style
	if IsTextNode(node) {
		text, _ := GetTextContent(node)
		lines := strings.Split(text, "\n")

		for lineIdx, line := range lines {
			lineY := y + lineIdx
			if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
				continue
			}

			charX := x
			for _, char := range line {
				if IsInClip(charX, lineY, clip) {
					buf.SetMerge(charX, lineY, New(char, parentStyle))
				}
				charX++
			}
		}
		return
	}

	// For other children, render normally
	RenderToLogicalBuffer(box, buf, clip)
}
