// Package goli provides intrinsic element handlers for box and text.
package goli

import (
	"strings"

	"github.com/germtb/gox"
)

func init() {
	// Register core intrinsic elements
	RegisterIntrinsic("box", &IntrinsicHandler{
		Measure:       measureBox,
		Layout:        layoutBox,
		Render:        renderBox,
		RenderLogical: renderBoxLogical,
	})

	RegisterIntrinsic("text", &IntrinsicHandler{
		Measure:       measureText,
		Layout:        layoutText,
		Render:        renderText,
		RenderLogical: renderTextLogical,
	})

	RegisterIntrinsic("input", &IntrinsicHandler{
		Measure:       measureInput,
		Layout:        layoutInput,
		Render:        RenderInputToBuffer,
		RenderLogical: RenderInputToLogicalBuffer,
	})

	RegisterIntrinsic("select", &IntrinsicHandler{
		Measure:       measureSelect,
		Layout:        layoutSelect,
		Render:        RenderSelectToBuffer,
		RenderLogical: RenderSelectToLogicalBuffer,
	})
}

// Box handlers

func measureBox(node gox.VNode, ctx *LayoutContext) (int, int) {
	padding := NormalizeSpacing(node.Props["padding"])
	border := GetBorderStyle(node.Props["border"])
	borderSize := 0
	if border != BorderNone {
		borderSize = 1
	}

	direction := GetDirection(node.Props)
	gap := GetIntProp(node.Props, "gap", 0)

	contentWidth := 0
	contentHeight := 0

	relativeChildren := FilterRelativeChildren(node)
	childSizes := make([]struct{ w, h int }, len(relativeChildren))
	for i, c := range relativeChildren {
		w, h := MeasureNode(c)
		childSizes[i] = struct{ w, h int }{w, h}
	}

	if direction == Row {
		for i, size := range childSizes {
			contentWidth += size.w
			if i > 0 {
				contentWidth += gap
			}
			if size.h > contentHeight {
				contentHeight = size.h
			}
		}
	} else {
		for i, size := range childSizes {
			contentHeight += size.h
			if i > 0 {
				contentHeight += gap
			}
			if size.w > contentWidth {
				contentWidth = size.w
			}
		}
	}

	totalWidth := contentWidth + padding.Left + padding.Right + borderSize*2
	totalHeight := contentHeight + padding.Top + padding.Bottom + borderSize*2

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

func layoutBox(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	var absoluteBoxes []*LayoutBox

	padding := NormalizeSpacing(node.Props["padding"])
	margin := NormalizeSpacing(node.Props["margin"])
	border := GetBorderStyle(node.Props["border"])
	borderSize := 0
	if border != BorderNone {
		borderSize = 1
	}

	direction := GetDirection(node.Props)
	justify := GetJustify(node.Props)
	align := GetAlign(node.Props)
	gap := GetIntProp(node.Props, "gap", 0)

	// Calculate box dimensions
	measuredW, measuredH := measureBox(node, nil)
	boxWidth := GetIntProp(node.Props, "width", -1)
	if boxWidth < 0 {
		boxWidth = min(measuredW, availWidth-margin.Left-margin.Right)
	}
	boxHeight := GetIntProp(node.Props, "height", -1)
	if boxHeight < 0 {
		boxHeight = measuredH
	}

	// Box position (respecting margin)
	boxX := ctx.X + margin.Left
	boxY := ctx.Y + margin.Top

	// Inner content area (inside border and padding)
	innerX := boxX + borderSize + padding.Left
	innerY := boxY + borderSize + padding.Top
	innerWidth := boxWidth - borderSize*2 - padding.Left - padding.Right
	innerHeight := boxHeight - borderSize*2 - padding.Top - padding.Bottom

	// Separate relative and absolute children
	relativeChildren := FilterRelativeChildren(node)
	absoluteChildren := FilterAbsoluteChildren(node)

	// Measure relative children
	childMeasurements := make([]ChildMeasurement, len(relativeChildren))
	for i, c := range relativeChildren {
		w, h := MeasureNode(c)
		childMeasurements[i] = ChildMeasurement{Node: c, Width: w, Height: h}
	}

	// Layout flex children
	childBoxes := LayoutFlexChildren(
		childMeasurements,
		LayoutContext{X: innerX, Y: innerY, Width: innerWidth, Height: innerHeight},
		direction,
		justify,
		align,
		gap,
		&absoluteBoxes,
	)

	// Layout absolute children
	for _, absChild := range absoluteChildren {
		absX := GetIntProp(absChild.Props, "x", 0)
		absY := GetIntProp(absChild.Props, "y", 0)
		result := LayoutNode(absChild, LayoutContext{
			X:      boxX + absX,
			Y:      boxY + absY,
			Width:  availWidth - absX,
			Height: availHeight - absY,
		})
		absoluteBoxes = append(absoluteBoxes, result.Box)
		absoluteBoxes = append(absoluteBoxes, result.AbsoluteBoxes...)
	}

	// Merge absolute boxes into children for rendering
	allChildren := make([]*LayoutBox, len(childBoxes)+len(absoluteBoxes))
	copy(allChildren, childBoxes)
	copy(allChildren[len(childBoxes):], absoluteBoxes)

	return &LayoutBox{
		X:           boxX,
		Y:           boxY,
		Width:       boxWidth,
		Height:      boxHeight,
		InnerX:      innerX,
		InnerY:      innerY,
		InnerWidth:  innerWidth,
		InnerHeight: innerHeight,
		Node:        node,
		Children:    allChildren,
		ZIndex:      GetIntProp(node.Props, "zIndex", 0),
	}
}

func renderBox(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	node := box.Node
	x, y, width, height := box.X, box.Y, box.Width, box.Height

	style := GetStyle(node.Props)
	borderStyle := GetBorderStyle(node.Props["border"])
	overflow := GetOverflow(node.Props)

	// Fill background if set
	if style.HasBackground() {
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				cellX, cellY := x+dx, y+dy
				if IsInClip(cellX, cellY, clip) {
					buf.Set(cellX, cellY, New(' ', Style{Background: style.Background, BackgroundRGB: style.BackgroundRGB}))
				}
			}
		}
	}

	// Draw border
	if borderStyle != BorderNone {
		chars := BorderCharSets[borderStyle]
		borderColor := style.Color

		// Top border
		if IsInClip(x, y, clip) {
			buf.SetCharMerge(x, y, chars.TopLeft, Style{Color: borderColor})
		}
		for dx := 1; dx < width-1; dx++ {
			if IsInClip(x+dx, y, clip) {
				buf.SetCharMerge(x+dx, y, chars.Horizontal, Style{Color: borderColor})
			}
		}
		if IsInClip(x+width-1, y, clip) {
			buf.SetCharMerge(x+width-1, y, chars.TopRight, Style{Color: borderColor})
		}

		// Side borders
		for dy := 1; dy < height-1; dy++ {
			if IsInClip(x, y+dy, clip) {
				buf.SetCharMerge(x, y+dy, chars.Vertical, Style{Color: borderColor})
			}
			if IsInClip(x+width-1, y+dy, clip) {
				buf.SetCharMerge(x+width-1, y+dy, chars.Vertical, Style{Color: borderColor})
			}
		}

		// Bottom border
		if IsInClip(x, y+height-1, clip) {
			buf.SetCharMerge(x, y+height-1, chars.BottomLeft, Style{Color: borderColor})
		}
		for dx := 1; dx < width-1; dx++ {
			if IsInClip(x+dx, y+height-1, clip) {
				buf.SetCharMerge(x+dx, y+height-1, chars.Horizontal, Style{Color: borderColor})
			}
		}
		if IsInClip(x+width-1, y+height-1, clip) {
			buf.SetCharMerge(x+width-1, y+height-1, chars.BottomRight, Style{Color: borderColor})
		}
	}

	// Calculate clip region for children
	childClip := clip
	if overflow == OverflowHidden || overflow == OverflowScroll {
		newClip := &ClipRegion{
			MinX: box.InnerX,
			MinY: box.InnerY,
			MaxX: box.InnerX + box.InnerWidth,
			MaxY: box.InnerY + box.InnerHeight,
		}
		childClip = IntersectClip(clip, newClip)
	}

	// Render children
	for _, childBox := range box.Children {
		RenderToBuffer(childBox, buf, childClip)
	}
}

func renderBoxLogical(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	node := box.Node
	x, y, width, height := box.X, box.Y, box.Width, box.Height

	style := GetStyle(node.Props)
	borderStyle := GetBorderStyle(node.Props["border"])
	overflow := GetOverflow(node.Props)

	// Fill background if set
	if style.HasBackground() {
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				cellX, cellY := x+dx, y+dy
				if IsInClip(cellX, cellY, clip) {
					buf.Set(cellX, cellY, New(' ', Style{Background: style.Background, BackgroundRGB: style.BackgroundRGB}))
				}
			}
		}
	}

	// Draw border
	if borderStyle != BorderNone {
		chars := BorderCharSets[borderStyle]
		borderColor := style.Color

		// Top border
		if IsInClip(x, y, clip) {
			buf.SetMerge(x, y, New(chars.TopLeft, Style{Color: borderColor}))
		}
		for dx := 1; dx < width-1; dx++ {
			if IsInClip(x+dx, y, clip) {
				buf.SetMerge(x+dx, y, New(chars.Horizontal, Style{Color: borderColor}))
			}
		}
		if IsInClip(x+width-1, y, clip) {
			buf.SetMerge(x+width-1, y, New(chars.TopRight, Style{Color: borderColor}))
		}

		// Side borders
		for dy := 1; dy < height-1; dy++ {
			if IsInClip(x, y+dy, clip) {
				buf.SetMerge(x, y+dy, New(chars.Vertical, Style{Color: borderColor}))
			}
			if IsInClip(x+width-1, y+dy, clip) {
				buf.SetMerge(x+width-1, y+dy, New(chars.Vertical, Style{Color: borderColor}))
			}
		}

		// Bottom border
		if IsInClip(x, y+height-1, clip) {
			buf.SetMerge(x, y+height-1, New(chars.BottomLeft, Style{Color: borderColor}))
		}
		for dx := 1; dx < width-1; dx++ {
			if IsInClip(x+dx, y+height-1, clip) {
				buf.SetMerge(x+dx, y+height-1, New(chars.Horizontal, Style{Color: borderColor}))
			}
		}
		if IsInClip(x+width-1, y+height-1, clip) {
			buf.SetMerge(x+width-1, y+height-1, New(chars.BottomRight, Style{Color: borderColor}))
		}
	}

	// Calculate clip region for children
	childClip := clip
	if overflow == OverflowHidden || overflow == OverflowScroll {
		newClip := &ClipRegion{
			MinX: box.InnerX,
			MinY: box.InnerY,
			MaxX: box.InnerX + box.InnerWidth,
			MaxY: box.InnerY + box.InnerHeight,
		}
		childClip = IntersectClip(clip, newClip)
	}

	// Render children
	for _, childBox := range box.Children {
		RenderToLogicalBuffer(childBox, buf, childClip)
	}
}

// Text handlers

func measureText(node gox.VNode, ctx *LayoutContext) (int, int) {
	text := CollectTextContent(node)
	lines := strings.Split(text, "\n")
	maxWidth := 0
	for _, line := range lines {
		if RuneWidth(line) > maxWidth {
			maxWidth = RuneWidth(line)
		}
	}
	return maxWidth, len(lines)
}

func layoutText(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	text := CollectTextContent(node)
	shouldWrap := GetBoolProp(node.Props, "wrap", false)

	var lines []string
	if shouldWrap {
		lines = WrapText(text, availWidth)
	} else {
		lines = strings.Split(text, "\n")
	}

	maxWidth := 0
	for _, line := range lines {
		if RuneWidth(line) > maxWidth {
			maxWidth = RuneWidth(line)
		}
	}

	w := min(maxWidth, availWidth)
	h := len(lines)

	// Create synthetic text node with wrapped content
	syntheticNode := CreateTextNode(strings.Join(lines, "\n"))
	syntheticNode.Props["style"] = node.Props["style"]

	return &LayoutBox{
		X:           ctx.X,
		Y:           ctx.Y,
		Width:       w,
		Height:      h,
		InnerX:      ctx.X,
		InnerY:      ctx.Y,
		InnerWidth:  w,
		InnerHeight: h,
		Node:        syntheticNode,
		Children:    nil,
		ZIndex:      GetIntProp(node.Props, "zIndex", 0),
	}
}

func renderText(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

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
			charX++
		}
	}
}

func renderTextLogical(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

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
			charX++
		}
	}
}

// Input handlers

func measureInput(node gox.VNode, ctx *LayoutContext) (int, int) {
	explicitWidth := GetIntProp(node.Props, "width", -1)
	explicitHeight := GetIntProp(node.Props, "height", -1)

	// Get display value from input
	inputPrim := node.Props["input"]
	displayValue := ""
	if inputer, ok := inputPrim.(interface{ DisplayValue() string }); ok {
		displayValue = inputer.DisplayValue()
	}

	lines := strings.Split(displayValue, "\n")
	maxWidth := 0
	for _, line := range lines {
		if RuneWidth(line) > maxWidth {
			maxWidth = RuneWidth(line)
		}
	}

	// Add 1 for cursor
	w := maxWidth + 1
	h := len(lines)

	if explicitWidth >= 0 {
		w = explicitWidth
	}
	if explicitHeight >= 0 {
		h = explicitHeight
	}

	return w, h
}

func layoutInput(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	w, h := measureInput(node, ctx)

	return &LayoutBox{
		X:           ctx.X,
		Y:           ctx.Y,
		Width:       w,
		Height:      h,
		InnerX:      ctx.X,
		InnerY:      ctx.Y,
		InnerWidth:  w,
		InnerHeight: h,
		Node:        node,
		Children:    nil,
		ZIndex:      GetIntProp(node.Props, "zIndex", 0),
	}
}

// Select handlers

func measureSelect(node gox.VNode, ctx *LayoutContext) (int, int) {
	pointerWidth := GetIntProp(node.Props, "pointerWidth", 2)
	optionChildren := FilterChildren(node, "option")

	maxOptionWidth := 0
	for _, opt := range optionChildren {
		optText := CollectTextContent(opt)
		if len(optText) > maxOptionWidth {
			maxOptionWidth = len(optText)
		}
	}

	return pointerWidth + maxOptionWidth, len(optionChildren)
}

func layoutSelect(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	w, h := measureSelect(node, ctx)

	// Auto-register options from children (doesn't trigger re-renders)
	selectPrim := node.Props["select"]
	optionChildren := FilterChildren(node, "option")

	if sel, ok := selectPrim.(interface {
		ClearOptions()
		SetOptionCount(int)
		RegisterOptionAny(int, any)
	}); ok {
		sel.ClearOptions()
		sel.SetOptionCount(len(optionChildren))
		for idx, opt := range optionChildren {
			if val, ok := opt.Props["value"]; ok {
				sel.RegisterOptionAny(idx, val)
			}
		}
	}

	return &LayoutBox{
		X:           ctx.X,
		Y:           ctx.Y,
		Width:       w,
		Height:      h,
		InnerX:      ctx.X,
		InnerY:      ctx.Y,
		InnerWidth:  w,
		InnerHeight: h,
		Node:        node,
		Children:    nil,
		ZIndex:      GetIntProp(node.Props, "zIndex", 0),
	}
}
