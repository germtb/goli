// Package goli provides buffer rendering functions.
package goli

import (
	"strings"

	"github.com/germtb/gox"
	"github.com/mattn/go-runewidth"
)

// RenderToBuffer renders a LayoutBox tree to a CellBuffer.
func RenderToBuffer(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	if box == nil {
		return
	}

	node := box.Node
	x, y := box.X, box.Y

	// Handle text nodes
	if IsTextNode(node) {
		style := GetStyle(node.Props)
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
					buf.SetCharMerge(charX, lineY, char, style)
				}
				charX += runewidth.RuneWidth(char)
			}
		}
		return
	}

	typeStr, ok := TypeString(node)
	if !ok {
		return
	}

	// Skip fragments, just render children
	if typeStr == "__fragment__" {
		for _, childBox := range box.Children {
			RenderToBuffer(childBox, buf, clip)
		}
		return
	}

	// Check for registered intrinsic element
	handler := GetIntrinsicHandler(typeStr)
	if handler == nil {
		panic("goli: unknown element type: " + typeStr)
	}
	if handler.Render != nil {
		handler.Render(box, buf, clip)
	}
}

// RenderToLogicalBuffer renders a LayoutBox tree to a LogicalBuffer.
func RenderToLogicalBuffer(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	if box == nil {
		return
	}

	node := box.Node
	x, y := box.X, box.Y

	// Handle text nodes
	if IsTextNode(node) {
		style := GetStyle(node.Props)
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
					buf.SetMerge(charX, lineY, New(char, style))
				}
				charX += runewidth.RuneWidth(char)
			}
		}
		return
	}

	typeStr, ok := TypeString(node)
	if !ok {
		return
	}

	// Skip fragments
	if typeStr == "__fragment__" {
		for _, childBox := range box.Children {
			RenderToLogicalBuffer(childBox, buf, clip)
		}
		return
	}

	// Check for registered intrinsic element
	handler := GetIntrinsicHandler(typeStr)
	if handler == nil {
		panic("goli: unknown element type: " + typeStr)
	}
	if handler.RenderLogical != nil {
		handler.RenderLogical(box, buf, clip)
	}
}

func RenderInputToBuffer(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	node := box.Node
	x, y, width, height := box.X, box.Y, box.Width, box.Height

	inputPrim := node.Props["input"]
	baseStyle := GetStyle(node.Props)
	if baseStyle.Color == ColorNone {
		baseStyle.Color = ColorWhite
	}

	cursorStyle := getStyleProp(node.Props, "cursorStyle", Style{Background: ColorWhite, Color: ColorBlack})
	placeholderStyle := getStyleProp(node.Props, "placeholderStyle", Style{Dim: true})

	displayValue := ""
	cursorPos := 0
	isFocused := false
	isPlaceholder := false

	if inp, ok := inputPrim.(interface {
		DisplayValue() string
		CursorPos() int
		Focused() bool
		ShowingPlaceholder() bool
	}); ok {
		displayValue = inp.DisplayValue()
		cursorPos = inp.CursorPos()
		isFocused = inp.Focused()
		isPlaceholder = inp.ShowingPlaceholder()
	}

	textStyle := baseStyle
	if isPlaceholder {
		textStyle = baseStyle.Merge(placeholderStyle)
	}

	lines := strings.Split(displayValue, "\n")
	charPos := 0

	// Calculate vertical scroll offset to keep cursor line visible
	cursorLine := 0
	tempPos := 0
	for i, line := range lines {
		if cursorPos >= tempPos && cursorPos <= tempPos+len(line) {
			cursorLine = i
			break
		}
		tempPos += len(line) + 1
	}
	scrollY := 0
	if cursorLine >= height {
		scrollY = cursorLine - height + 1
	}

	for lineIdx := 0; lineIdx < height; lineIdx++ {
		lineY := y + lineIdx
		srcLineIdx := lineIdx + scrollY

		if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
			continue
		}

		if srcLineIdx < len(lines) {
			line := lines[srcLineIdx]
			lineRunes := []rune(line)

			// Calculate charPos for this line
			lineCharPos := 0
			for i := 0; i < srcLineIdx; i++ {
				lineCharPos += len(lines[i]) + 1
			}

			cursorOnThisLine := isFocused && cursorPos >= lineCharPos && cursorPos <= lineCharPos+len(lineRunes)
			cursorColOnLine := cursorPos - lineCharPos

			// Calculate horizontal scroll offset to keep cursor visible
			scrollX := 0
			if cursorOnThisLine && cursorColOnLine >= width {
				scrollX = cursorColOnLine - width + 1
			}

			for i := 0; i < width; i++ {
				charX := x + i
				if !IsInClip(charX, lineY, clip) {
					continue
				}

				srcIdx := i + scrollX
				var char rune = ' '
				if srcIdx < len(lineRunes) {
					char = lineRunes[srcIdx]
				}

				if cursorOnThisLine && srcIdx == cursorColOnLine {
					buf.Set(charX, lineY, New(char, cursorStyle))
				} else if srcIdx < len(lineRunes) {
					buf.SetCharMerge(charX, lineY, char, textStyle)
				} else {
					buf.SetCharMerge(charX, lineY, ' ', textStyle)
				}
			}

			charPos = lineCharPos + len(line) + 1
		} else {
			for i := 0; i < width; i++ {
				charX := x + i
				if IsInClip(charX, lineY, clip) {
					buf.SetCharMerge(charX, lineY, ' ', EmptyStyle)
				}
			}
		}
	}
	_ = charPos // silence unused variable warning
}

func RenderInputToLogicalBuffer(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	node := box.Node
	x, y, width, height := box.X, box.Y, box.Width, box.Height

	inputPrim := node.Props["input"]
	baseStyle := GetStyle(node.Props)
	if baseStyle.Color == ColorNone {
		baseStyle.Color = ColorWhite
	}

	cursorStyle := getStyleProp(node.Props, "cursorStyle", Style{Background: ColorWhite, Color: ColorBlack})
	placeholderStyle := getStyleProp(node.Props, "placeholderStyle", Style{Dim: true})

	displayValue := ""
	cursorPos := 0
	isFocused := false
	isPlaceholder := false

	if inp, ok := inputPrim.(interface {
		DisplayValue() string
		CursorPos() int
		Focused() bool
		ShowingPlaceholder() bool
	}); ok {
		displayValue = inp.DisplayValue()
		cursorPos = inp.CursorPos()
		isFocused = inp.Focused()
		isPlaceholder = inp.ShowingPlaceholder()
	}

	textStyle := baseStyle
	if isPlaceholder {
		textStyle = baseStyle.Merge(placeholderStyle)
	}

	lines := strings.Split(displayValue, "\n")
	charPos := 0

	// Calculate vertical scroll offset to keep cursor line visible
	cursorLine := 0
	tempPos := 0
	for i, line := range lines {
		if cursorPos >= tempPos && cursorPos <= tempPos+len(line) {
			cursorLine = i
			break
		}
		tempPos += len(line) + 1
	}
	scrollY := 0
	if cursorLine >= height {
		scrollY = cursorLine - height + 1
	}

	for lineIdx := 0; lineIdx < height; lineIdx++ {
		lineY := y + lineIdx
		srcLineIdx := lineIdx + scrollY

		if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
			continue
		}

		if srcLineIdx < len(lines) {
			line := lines[srcLineIdx]
			lineRunes := []rune(line)

			// Calculate charPos for this line
			lineCharPos := 0
			for i := 0; i < srcLineIdx; i++ {
				lineCharPos += len(lines[i]) + 1
			}

			cursorOnThisLine := isFocused && cursorPos >= lineCharPos && cursorPos <= lineCharPos+len(lineRunes)
			cursorColOnLine := cursorPos - lineCharPos

			// Calculate horizontal scroll offset to keep cursor visible
			scrollX := 0
			if cursorOnThisLine && cursorColOnLine >= width {
				scrollX = cursorColOnLine - width + 1
			}

			for i := 0; i < width; i++ {
				charX := x + i
				if !IsInClip(charX, lineY, clip) {
					continue
				}

				srcIdx := i + scrollX
				var char rune = ' '
				if srcIdx < len(lineRunes) {
					char = lineRunes[srcIdx]
				}

				if cursorOnThisLine && srcIdx == cursorColOnLine {
					buf.Set(charX, lineY, New(char, cursorStyle))
				} else {
					buf.SetMerge(charX, lineY, New(char, textStyle))
				}
			}

			charPos = lineCharPos + len(line) + 1
		} else {
			for i := 0; i < width; i++ {
				charX := x + i
				if IsInClip(charX, lineY, clip) {
					buf.SetMerge(charX, lineY, New(' ', EmptyStyle))
				}
			}
		}
	}
	_ = charPos // silence unused variable warning
}

func RenderSelectToBuffer(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

	selectPrim := node.Props["select"]
	pointerWidth := GetIntProp(node.Props, "pointerWidth", 2)
	baseOptionStyle := getStyleProp(node.Props, "optionStyle", EmptyStyle)
	selectedStyle := getStyleProp(node.Props, "selectedStyle", EmptyStyle)

	optionChildren := FilterChildren(node, "option")

	for idx, opt := range optionChildren {
		optY := y + idx
		if clip != nil && (optY < clip.MinY || optY >= clip.MaxY) {
			continue
		}

		isSelected := false
		if sel, ok := selectPrim.(interface{ IsSelectedIndex(int) bool }); ok {
			isSelected = sel.IsSelectedIndex(idx)
		}

		computedStyle := baseOptionStyle.Merge(GetStyle(opt.Props))
		if isSelected {
			computedStyle = computedStyle.Merge(selectedStyle)
		}

		// Render pointer (iterate by runes, not bytes)
		pointerRunes := []rune(strings.Repeat(" ", pointerWidth))
		if isSelected {
			if pointer := node.Props["pointer"]; pointer != nil {
				if pnode, ok := pointer.(gox.VNode); ok {
					pointerRunes = []rune(CollectTextContent(pnode))
				}
			}
		}

		for i := 0; i < pointerWidth && i < len(pointerRunes); i++ {
			charX := x + i
			if IsInClip(charX, optY, clip) {
				buf.SetCharMerge(charX, optY, pointerRunes[i], EmptyStyle)
			}
		}

		// Render option text
		optText := CollectTextContent(opt)
		charX := x + pointerWidth
		for _, char := range optText {
			if IsInClip(charX, optY, clip) {
				buf.SetCharMerge(charX, optY, char, computedStyle)
			}
			charX += runewidth.RuneWidth(char)
		}
	}
}

func RenderSelectToLogicalBuffer(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

	selectPrim := node.Props["select"]
	pointerWidth := GetIntProp(node.Props, "pointerWidth", 2)
	baseOptionStyle := getStyleProp(node.Props, "optionStyle", EmptyStyle)
	selectedStyle := getStyleProp(node.Props, "selectedStyle", EmptyStyle)

	optionChildren := FilterChildren(node, "option")

	for idx, opt := range optionChildren {
		optY := y + idx
		if clip != nil && (optY < clip.MinY || optY >= clip.MaxY) {
			continue
		}

		isSelected := false
		if sel, ok := selectPrim.(interface{ IsSelectedIndex(int) bool }); ok {
			isSelected = sel.IsSelectedIndex(idx)
		}

		computedStyle := baseOptionStyle.Merge(GetStyle(opt.Props))
		if isSelected {
			computedStyle = computedStyle.Merge(selectedStyle)
		}

		// Render pointer (iterate by runes, not bytes)
		pointerRunes := []rune(strings.Repeat(" ", pointerWidth))
		if isSelected {
			if pointer := node.Props["pointer"]; pointer != nil {
				if pnode, ok := pointer.(gox.VNode); ok {
					pointerRunes = []rune(CollectTextContent(pnode))
				}
			}
		}

		for i := 0; i < pointerWidth && i < len(pointerRunes); i++ {
			charX := x + i
			if IsInClip(charX, optY, clip) {
				buf.SetMerge(charX, optY, New(pointerRunes[i], EmptyStyle))
			}
		}

		// Render option text
		optText := CollectTextContent(opt)
		charX := x + pointerWidth
		for _, char := range optText {
			if IsInClip(charX, optY, clip) {
				buf.SetMerge(charX, optY, New(char, computedStyle))
			}
			charX += runewidth.RuneWidth(char)
		}
	}
}

// GetOverflow returns the overflow mode from props.
func GetOverflow(props map[string]any) Overflow {
	if props == nil {
		return OverflowVisible
	}
	v, ok := props["overflow"]
	if !ok {
		return OverflowVisible
	}
	if s, ok := v.(string); ok {
		return Overflow(s)
	}
	if o, ok := v.(Overflow); ok {
		return o
	}
	return OverflowVisible
}

func getStyleProp(props map[string]any, key string, defaultStyle Style) Style {
	if props == nil {
		return defaultStyle
	}
	v, ok := props[key]
	if !ok {
		return defaultStyle
	}
	switch s := v.(type) {
	case Style:
		return s
	case map[string]any:
		return mapToStyle(s)
	default:
		return defaultStyle
	}
}
