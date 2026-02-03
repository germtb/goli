// Package goli provides the flexbox layout engine for terminal UI.
package goli

import (
	"strings"
	"unicode/utf8"

	"github.com/germtb/gox"
)

// Direction specifies the main axis for flex layout.
type Direction string

const (
	Row    Direction = "row"
	Column Direction = "column"
)

// Justify specifies alignment along the main axis.
type Justify string

const (
	JustifyStart        Justify = "start"
	JustifyCenter       Justify = "center"
	JustifyEnd          Justify = "end"
	JustifySpaceBetween Justify = "space-between"
	JustifySpaceAround  Justify = "space-around"
)

// Align specifies alignment along the cross axis.
type Align string

const (
	AlignStart   Align = "start"
	AlignCenter  Align = "center"
	AlignEnd     Align = "end"
	AlignStretch Align = "stretch"
)

// Position specifies positioning mode.
type Position string

const (
	PositionRelative Position = "relative"
	PositionAbsolute Position = "absolute"
)

// BorderStyle specifies the border appearance.
type BorderStyle string

const (
	BorderNone    BorderStyle = "none"
	BorderSingle  BorderStyle = "single"
	BorderDouble  BorderStyle = "double"
	BorderRounded BorderStyle = "rounded"
	BorderBold    BorderStyle = "bold"
)

// Overflow specifies overflow behavior.
type Overflow string

const (
	OverflowVisible Overflow = "visible"
	OverflowHidden  Overflow = "hidden"
	OverflowScroll  Overflow = "scroll"
)

// Spacing represents padding or margin on all sides.
type Spacing struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// BorderChars holds the characters for drawing a border.
type BorderChars struct {
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	Horizontal  rune
	Vertical    rune
}

// Border character sets for different styles.
var BorderCharSets = map[BorderStyle]BorderChars{
	BorderSingle: {
		TopLeft:     '┌',
		TopRight:    '┐',
		BottomLeft:  '└',
		BottomRight: '┘',
		Horizontal:  '─',
		Vertical:    '│',
	},
	BorderDouble: {
		TopLeft:     '╔',
		TopRight:    '╗',
		BottomLeft:  '╚',
		BottomRight: '╝',
		Horizontal:  '═',
		Vertical:    '║',
	},
	BorderRounded: {
		TopLeft:     '╭',
		TopRight:    '╮',
		BottomLeft:  '╰',
		BottomRight: '╯',
		Horizontal:  '─',
		Vertical:    '│',
	},
	BorderBold: {
		TopLeft:     '┏',
		TopRight:    '┓',
		BottomLeft:  '┗',
		BottomRight: '┛',
		Horizontal:  '━',
		Vertical:    '┃',
	},
}

// NormalizeSpacing converts various spacing inputs to a Spacing struct.
func NormalizeSpacing(value any) Spacing {
	if value == nil {
		return Spacing{}
	}

	switch v := value.(type) {
	case int:
		return Spacing{Top: v, Right: v, Bottom: v, Left: v}
	case float64:
		i := int(v)
		return Spacing{Top: i, Right: i, Bottom: i, Left: i}
	case Spacing:
		return v
	case map[string]any:
		return Spacing{
			Top:    getInt(v, "top"),
			Right:  getInt(v, "right"),
			Bottom: getInt(v, "bottom"),
			Left:   getInt(v, "left"),
		}
	default:
		return Spacing{}
	}
}

// GetSpacing extracts spacing from props, supporting both base prop and directional overrides.
// For example, GetSpacing(props, "padding") reads "padding" and also
// "paddingTop", "paddingRight", "paddingBottom", "paddingLeft" as overrides.
func GetSpacing(props map[string]any, baseProp string) Spacing {
	// Start with the base prop
	spacing := NormalizeSpacing(props[baseProp])

	// Override with directional props if present
	if v, ok := props[baseProp+"Top"]; ok {
		spacing.Top = getIntFromAny(v)
	}
	if v, ok := props[baseProp+"Right"]; ok {
		spacing.Right = getIntFromAny(v)
	}
	if v, ok := props[baseProp+"Bottom"]; ok {
		spacing.Bottom = getIntFromAny(v)
	}
	if v, ok := props[baseProp+"Left"]; ok {
		spacing.Left = getIntFromAny(v)
	}

	return spacing
}

// getIntFromAny converts various numeric types to int.
func getIntFromAny(v any) int {
	switch i := v.(type) {
	case int:
		return i
	case float64:
		return int(i)
	default:
		return 0
	}
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch i := v.(type) {
		case int:
			return i
		case float64:
			return int(i)
		}
	}
	return 0
}

// GetBorderStyle normalizes border prop to BorderStyle.
func GetBorderStyle(border any) BorderStyle {
	if border == nil {
		return BorderNone
	}

	switch v := border.(type) {
	case bool:
		if v {
			return BorderSingle
		}
		return BorderNone
	case string:
		return BorderStyle(v)
	case BorderStyle:
		return v
	default:
		return BorderNone
	}
}

// GetStyle extracts a Style from props.
func GetStyle(props map[string]any) Style {
	styleVal, ok := props["style"]
	if !ok || styleVal == nil {
		return EmptyStyle
	}

	switch s := styleVal.(type) {
	case Style:
		return s
	case map[string]any:
		return mapToStyle(s)
	default:
		return EmptyStyle
	}
}

func mapToStyle(m map[string]any) Style {
	style := Style{}

	if v, ok := m["color"]; ok {
		style.Color, style.ColorRGB = toColor(v)
	}
	if v, ok := m["background"]; ok {
		style.Background, style.BackgroundRGB = toColor(v)
	}
	if v, ok := m["bold"].(bool); ok {
		style.Bold = v
	}
	if v, ok := m["dim"].(bool); ok {
		style.Dim = v
	}
	if v, ok := m["italic"].(bool); ok {
		style.Italic = v
	}
	if v, ok := m["underline"].(bool); ok {
		style.Underline = v
	}
	if v, ok := m["inverse"].(bool); ok {
		style.Inverse = v
	}
	if v, ok := m["strikethrough"].(bool); ok {
		style.Strikethrough = v
	}

	return style
}

func toColor(v any) (Color, *RGB) {
	switch c := v.(type) {
	case string:
		if color, ok := NameToColor[c]; ok {
			return color, nil
		}
		return ColorNone, nil
	case Color:
		return c, nil
	case RGB:
		return ColorNone, &c
	case *RGB:
		return ColorNone, c
	default:
		return ColorNone, nil
	}
}

// ClipRegion defines the visible area for clipping content.
type ClipRegion struct {
	MinX int // Inclusive
	MinY int // Inclusive
	MaxX int // Exclusive
	MaxY int // Exclusive
}

// LayoutBox represents a computed layout for a node.
type LayoutBox struct {
	// Position (absolute, after all calculations)
	X      int
	Y      int
	Width  int
	Height int

	// Content area (inside padding/border)
	InnerX      int
	InnerY      int
	InnerWidth  int
	InnerHeight int

	// The node this box represents
	Node gox.VNode

	// Child boxes
	Children []*LayoutBox

	// For z-index sorting
	ZIndex int
}

// LayoutContext provides the available space for layout.
type LayoutContext struct {
	X      int
	Y      int
	Width  int
	Height int
}

// LayoutResult holds the result of layout computation.
type LayoutResult struct {
	Box           *LayoutBox
	AbsoluteBoxes []*LayoutBox
}

// IsInClip checks if a position is within the clip region.
func IsInClip(x, y int, clip *ClipRegion) bool {
	if clip == nil {
		return true
	}
	return x >= clip.MinX && x < clip.MaxX && y >= clip.MinY && y < clip.MaxY
}

// IntersectClip intersects two clip regions, returning the overlapping area.
func IntersectClip(a, b *ClipRegion) *ClipRegion {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return &ClipRegion{
		MinX: max(a.MinX, b.MinX),
		MinY: max(a.MinY, b.MinY),
		MaxX: min(a.MaxX, b.MaxX),
		MaxY: min(a.MaxY, b.MaxY),
	}
}

// RuneWidth returns the display width of a string (rune count).
func RuneWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// ComputeLayout computes layout for a VNode tree.
func ComputeLayout(node gox.VNode, ctx LayoutContext) *LayoutBox {
	// First expand any functional components
	expanded := Expand(node)

	// Layout the tree
	result := layoutNode(expanded, ctx)

	// Merge absolute boxes back, sorted by z-index
	allAbsolute := collectAbsoluteBoxes(result.Box)
	allAbsolute = append(allAbsolute, result.AbsoluteBoxes...)

	// Sort by z-index
	sortByZIndex(allAbsolute)

	// Return root with absolute boxes as additional children for rendering
	newChildren := make([]*LayoutBox, len(result.Box.Children)+len(allAbsolute))
	copy(newChildren, result.Box.Children)
	copy(newChildren[len(result.Box.Children):], allAbsolute)

	return &LayoutBox{
		X:           result.Box.X,
		Y:           result.Box.Y,
		Width:       result.Box.Width,
		Height:      result.Box.Height,
		InnerX:      result.Box.InnerX,
		InnerY:      result.Box.InnerY,
		InnerWidth:  result.Box.InnerWidth,
		InnerHeight: result.Box.InnerHeight,
		Node:        result.Box.Node,
		Children:    newChildren,
		ZIndex:      result.Box.ZIndex,
	}
}

func sortByZIndex(boxes []*LayoutBox) {
	for i := 0; i < len(boxes)-1; i++ {
		for j := i + 1; j < len(boxes); j++ {
			if boxes[i].ZIndex > boxes[j].ZIndex {
				boxes[i], boxes[j] = boxes[j], boxes[i]
			}
		}
	}
}

func collectAbsoluteBoxes(box *LayoutBox) []*LayoutBox {
	var result []*LayoutBox
	for _, child := range box.Children {
		if getPosition(child.Node.Props) == PositionAbsolute {
			result = append(result, child)
		}
		result = append(result, collectAbsoluteBoxes(child)...)
	}
	return result
}

// MeasureNode measures the natural size of a node (before flex distribution).
func MeasureNode(node gox.VNode) (width, height int) {
	return measureNode(node)
}

// measureNode measures the natural size of a node (before flex distribution).
func measureNode(node gox.VNode) (width, height int) {
	// Text nodes: width = text length, height = 1
	if IsTextNode(node) {
		text, _ := GetTextContent(node)
		lines := strings.Split(text, "\n")
		maxWidth := 0
		for _, line := range lines {
			if RuneWidth(line) > maxWidth {
				maxWidth = RuneWidth(line)
			}
		}
		return maxWidth, len(lines)
	}

	typeStr, ok := TypeString(node)
	if !ok {
		return 0, 0
	}

	// Check for registered intrinsic element
	handler := GetIntrinsicHandler(typeStr)
	if handler == nil {
		panic("goli: unknown element type: " + typeStr)
	}
	if handler.Measure != nil {
		return handler.Measure(node, nil)
	}

	// Handler exists but no Measure - measure children as container
	padding := GetSpacing(node.Props, "padding")
	border := GetBorderStyle(node.Props["border"])
	borderSize := 0
	if border != BorderNone {
		borderSize = 1
	}

	direction := getDirection(node.Props)
	gap := GetIntProp(node.Props, "gap", 0)

	contentWidth := 0
	contentHeight := 0

	relativeChildren := filterRelativeChildren(node)
	childSizes := make([]struct{ w, h int }, len(relativeChildren))
	for i, c := range relativeChildren {
		w, h := measureNode(c)
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

// LayoutNode computes layout for a single node.
func LayoutNode(node gox.VNode, ctx LayoutContext) LayoutResult {
	return layoutNode(node, ctx)
}

func layoutNode(node gox.VNode, ctx LayoutContext) LayoutResult {
	var absoluteBoxes []*LayoutBox

	typeStr, ok := TypeString(node)
	if !ok {
		// Component not expanded - shouldn't happen
		return LayoutResult{Box: &LayoutBox{Node: node}}
	}

	// Handle fragments
	if typeStr == "fragment" || typeStr == gox.FragmentNodeType {
		return layoutFragment(node, ctx)
	}

	// Handle text nodes
	if IsTextNode(node) {
		text, _ := GetTextContent(node)
		lines := strings.Split(text, "\n")
		maxWidth := 0
		for _, line := range lines {
			if RuneWidth(line) > maxWidth {
				maxWidth = RuneWidth(line)
			}
		}

		w := min(maxWidth, ctx.Width)
		h := len(lines)

		return LayoutResult{
			Box: &LayoutBox{
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
			},
			AbsoluteBoxes: nil,
		}
	}

	// Check for registered intrinsic element
	handler := GetIntrinsicHandler(typeStr)
	if handler == nil {
		panic("goli: unknown element type: " + typeStr)
	}
	if handler.Layout != nil {
		box := handler.Layout(node, ctx.Width, ctx.Height, &ctx)
		return LayoutResult{
			Box:           box,
			AbsoluteBoxes: nil,
		}
	}

	// Handler exists but no Layout - treat as flex container
	padding := GetSpacing(node.Props, "padding")
	margin := GetSpacing(node.Props, "margin")
	border := GetBorderStyle(node.Props["border"])
	borderSize := 0
	if border != BorderNone {
		borderSize = 1
	}

	direction := getDirection(node.Props)
	justify := getJustify(node.Props)
	align := getAlign(node.Props)
	gap := GetIntProp(node.Props, "gap", 0)

	// Calculate box dimensions
	// Both width and height fill available space by default (block-like)
	// Use explicit width/height props to constrain size
	// Use grow property for flex children to distribute extra space
	measuredW, measuredH := measureNode(node)
	boxWidth := GetIntProp(node.Props, "width", -1)
	if boxWidth < 0 {
		// Width fills available space
		boxWidth = ctx.Width - margin.Left - margin.Right
		if boxWidth < 0 {
			boxWidth = measuredW
		}
	}
	boxHeight := GetIntProp(node.Props, "height", -1)
	if boxHeight < 0 {
		// Height fills available space
		boxHeight = ctx.Height - margin.Top - margin.Bottom
		if boxHeight < 0 {
			boxHeight = measuredH
		}
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
	relativeChildren := filterRelativeChildren(node)
	absoluteChildren := filterAbsoluteChildren(node)

	// Measure relative children
	childMeasurements := make([]childMeasurement, len(relativeChildren))
	for i, c := range relativeChildren {
		w, h := measureNode(c)
		childMeasurements[i] = childMeasurement{node: c, width: w, height: h}
	}

	// Layout flex children
	childBoxes := layoutFlexChildren(
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
		result := layoutNode(absChild, LayoutContext{
			X:      boxX + absX,
			Y:      boxY + absY,
			Width:  ctx.Width - absX,
			Height: ctx.Height - absY,
		})
		absoluteBoxes = append(absoluteBoxes, result.Box)
		absoluteBoxes = append(absoluteBoxes, result.AbsoluteBoxes...)
	}

	return LayoutResult{
		Box: &LayoutBox{
			X:           boxX,
			Y:           boxY,
			Width:       boxWidth,
			Height:      boxHeight,
			InnerX:      innerX,
			InnerY:      innerY,
			InnerWidth:  innerWidth,
			InnerHeight: innerHeight,
			Node:        node,
			Children:    childBoxes,
			ZIndex:      GetIntProp(node.Props, "zIndex", 0),
		},
		AbsoluteBoxes: absoluteBoxes,
	}
}

func layoutFragment(node gox.VNode, ctx LayoutContext) LayoutResult {
	var children []*LayoutBox
	var absoluteBoxes []*LayoutBox
	offsetY := 0

	for _, child := range node.Children {
		if getPosition(child.Props) == PositionAbsolute {
			result := layoutNode(child, ctx)
			absoluteBoxes = append(absoluteBoxes, result.Box)
			absoluteBoxes = append(absoluteBoxes, result.AbsoluteBoxes...)
		} else {
			result := layoutNode(child, LayoutContext{
				X:      ctx.X,
				Y:      ctx.Y + offsetY,
				Width:  ctx.Width,
				Height: ctx.Height - offsetY,
			})
			children = append(children, result.Box)
			absoluteBoxes = append(absoluteBoxes, result.AbsoluteBoxes...)
			margin := GetSpacing(child.Props, "margin")
			offsetY += result.Box.Height + margin.Bottom
		}
	}

	return LayoutResult{
		Box: &LayoutBox{
			X:           ctx.X,
			Y:           ctx.Y,
			Width:       ctx.Width,
			Height:      offsetY,
			InnerX:      ctx.X,
			InnerY:      ctx.Y,
			InnerWidth:  ctx.Width,
			InnerHeight: offsetY,
			Node:        node,
			Children:    children,
			ZIndex:      0,
		},
		AbsoluteBoxes: absoluteBoxes,
	}
}

// ChildMeasurement holds a measured child node.
type ChildMeasurement struct {
	Node   gox.VNode
	Width  int
	Height int
}

type childMeasurement struct {
	node   gox.VNode
	width  int
	height int
}

// LayoutFlexChildren lays out children using flexbox rules.
func LayoutFlexChildren(
	children []ChildMeasurement,
	ctx LayoutContext,
	direction Direction,
	justify Justify,
	align Align,
	gap int,
	absoluteBoxes *[]*LayoutBox,
) []*LayoutBox {
	// Convert to internal type
	internal := make([]childMeasurement, len(children))
	for i, c := range children {
		internal[i] = childMeasurement{node: c.Node, width: c.Width, height: c.Height}
	}
	return layoutFlexChildren(internal, ctx, direction, justify, align, gap, absoluteBoxes)
}

func layoutFlexChildren(
	children []childMeasurement,
	ctx LayoutContext,
	direction Direction,
	justify Justify,
	align Align,
	gap int,
	absoluteBoxes *[]*LayoutBox,
) []*LayoutBox {
	if len(children) == 0 {
		return nil
	}

	isRow := direction == Row

	// Calculate total size along main axis
	totalMainSize := 0
	for i, child := range children {
		margin := GetSpacing(child.node.Props, "margin")
		var mainMargin int
		var mainSize int
		if isRow {
			mainMargin = margin.Left + margin.Right
			mainSize = child.width
		} else {
			mainMargin = margin.Top + margin.Bottom
			mainSize = child.height
		}
		totalMainSize += mainMargin + mainSize
		if i > 0 {
			totalMainSize += gap
		}
	}

	availableMain := ctx.Width
	availableCross := ctx.Height
	if !isRow {
		availableMain = ctx.Height
		availableCross = ctx.Width
	}

	// Calculate grow values and distribute extra space
	// Children with explicit main-axis size (width for row, height for column) don't participate in grow
	totalGrow := 0
	growValues := make([]int, len(children))
	for i, child := range children {
		grow := GetIntProp(child.node.Props, "grow", 0)
		// Check if child has explicit main-axis size - if so, don't let it grow
		if isRow {
			if GetIntProp(child.node.Props, "width", -1) >= 0 {
				grow = 0
			}
		} else {
			if GetIntProp(child.node.Props, "height", -1) >= 0 {
				grow = 0
			}
		}
		growValues[i] = grow
		totalGrow += grow
	}

	// Calculate extra space for growing children
	extraSpace := 0
	if totalGrow > 0 && availableMain > totalMainSize {
		extraSpace = availableMain - totalMainSize
	}

	// Pre-calculate grow shares with remainder distribution
	// This ensures all extra space is used (no rounding loss)
	growShares := make([]int, len(children))
	if totalGrow > 0 && extraSpace > 0 {
		remainingSpace := extraSpace
		for i := range children {
			if growValues[i] > 0 {
				// Calculate this child's share
				share := (extraSpace * growValues[i]) / totalGrow
				growShares[i] = share
				remainingSpace -= share
			}
		}
		// Distribute remainder to growing children (1 extra pixel each until exhausted)
		for i := range children {
			if remainingSpace <= 0 {
				break
			}
			if growValues[i] > 0 {
				growShares[i]++
				remainingSpace--
			}
		}
	}

	// Calculate starting position and spacing based on justify
	mainPos := 0
	extraGap := 0

	switch justify {
	case JustifyStart:
		mainPos = 0
	case JustifyCenter:
		mainPos = max(0, (availableMain-totalMainSize)/2)
	case JustifyEnd:
		mainPos = max(0, availableMain-totalMainSize)
	case JustifySpaceBetween:
		if len(children) > 1 {
			extraGap = max(0, (availableMain-totalMainSize+gap*(len(children)-1))/(len(children)-1))
		}
	case JustifySpaceAround:
		if len(children) > 0 {
			totalSpace := availableMain - totalMainSize + gap*(len(children)-1)
			extraGap = totalSpace / len(children)
			mainPos = extraGap / 2
		}
	}

	// Layout each child
	var boxes []*LayoutBox

	for i, child := range children {
		margin := GetSpacing(child.node.Props, "margin")
		var childMainSize, childCrossSize int
		var mainMarginBefore, mainMarginAfter int

		if isRow {
			childMainSize = child.width
			childCrossSize = child.height
			mainMarginBefore = margin.Left
			mainMarginAfter = margin.Right
		} else {
			childMainSize = child.height
			childCrossSize = child.width
			mainMarginBefore = margin.Top
			mainMarginAfter = margin.Bottom
		}

		// Apply grow if applicable (using pre-calculated shares with remainder distribution)
		if growShares[i] > 0 {
			childMainSize += growShares[i]
		}

		// Calculate cross-axis position and size
		// Default: stretch to fill (CSS flex default is align-items: stretch)
		// Non-stretch alignments use intrinsic size
		crossPos := 0
		actualCrossSize := childCrossSize // Default to intrinsic size

		switch align {
		case AlignStart:
			crossPos = 0
			actualCrossSize = childCrossSize
		case AlignCenter:
			crossPos = max(0, (availableCross-childCrossSize)/2)
			actualCrossSize = childCrossSize
		case AlignEnd:
			crossPos = max(0, availableCross-childCrossSize)
			actualCrossSize = childCrossSize
		case AlignStretch:
			crossPos = 0
			actualCrossSize = availableCross
		default:
			// Default behavior is stretch (CSS flex default)
			actualCrossSize = availableCross
		}

		var childX, childY, childWidth, childHeight int
		if isRow {
			childX = ctx.X + mainPos
			childY = ctx.Y + crossPos
			// Add margins to available size so layoutBox can subtract them
			// (flex parent already accounted for margins in space distribution)
			childWidth = childMainSize + margin.Left + margin.Right
			childHeight = actualCrossSize + margin.Top + margin.Bottom
		} else {
			childX = ctx.X + crossPos
			childY = ctx.Y + mainPos
			// Add margins to available size so layoutBox can subtract them
			childWidth = actualCrossSize + margin.Left + margin.Right
			childHeight = childMainSize + margin.Top + margin.Bottom
		}

		result := layoutNode(child.node, LayoutContext{
			X:      childX,
			Y:      childY,
			Width:  childWidth,
			Height: childHeight,
		})

		boxes = append(boxes, result.Box)
		*absoluteBoxes = append(*absoluteBoxes, result.AbsoluteBoxes...)

		effectiveGap := gap
		if justify == JustifySpaceBetween || justify == JustifySpaceAround {
			effectiveGap = extraGap
		}
		mainPos += mainMarginBefore + childMainSize + mainMarginAfter + effectiveGap
	}

	return boxes
}

// CollectTextContent recursively collects all text content from a node.
func CollectTextContent(node gox.VNode) string {
	if IsTextNode(node) {
		text, _ := GetTextContent(node)
		return text
	}

	var result strings.Builder
	for _, child := range node.Children {
		result.WriteString(CollectTextContent(child))
	}
	return result.String()
}

// WrapText wraps text to fit within a given width.
func WrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	inputLines := strings.Split(text, "\n")
	var outputLines []string

	for _, line := range inputLines {
		if RuneWidth(line) <= maxWidth {
			outputLines = append(outputLines, line)
			continue
		}

		// Need to wrap this line
		remaining := line
		for len(remaining) > maxWidth {
			// Try to find a word boundary
			breakPoint := strings.LastIndex(remaining[:maxWidth+1], " ")

			// If no space or too early, hard wrap
			if breakPoint <= 0 || breakPoint < maxWidth/2 {
				breakPoint = maxWidth
			}

			outputLines = append(outputLines, remaining[:breakPoint])
			remaining = strings.TrimLeft(remaining[breakPoint:], " ")
		}

		if len(remaining) > 0 {
			outputLines = append(outputLines, remaining)
		}
	}

	return outputLines
}

// Helper functions

func GetIntProp(props gox.Props, key string, defaultVal int) int {
	if props == nil {
		return defaultVal
	}
	v, ok := props[key]
	if !ok {
		return defaultVal
	}
	switch i := v.(type) {
	case int:
		return i
	case float64:
		return int(i)
	default:
		return defaultVal
	}
}

// GetBoolProp gets a boolean property with a default value.
func GetBoolProp(props gox.Props, key string, defaultVal bool) bool {
	if props == nil {
		return defaultVal
	}
	v, ok := props[key]
	if !ok {
		return defaultVal
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return defaultVal
}

// GetDirection returns the flex direction from props.
func GetDirection(props gox.Props) Direction {
	return getDirection(props)
}

func getDirection(props gox.Props) Direction {
	if props == nil {
		return Column
	}
	v, ok := props["direction"]
	if !ok {
		return Column
	}
	if s, ok := v.(string); ok {
		return Direction(s)
	}
	if d, ok := v.(Direction); ok {
		return d
	}
	return Column
}

// GetJustify returns the justify-content from props.
func GetJustify(props gox.Props) Justify {
	return getJustify(props)
}

func getJustify(props gox.Props) Justify {
	if props == nil {
		return JustifyStart
	}
	v, ok := props["justify"]
	if !ok {
		return JustifyStart
	}
	if s, ok := v.(string); ok {
		return Justify(s)
	}
	if j, ok := v.(Justify); ok {
		return j
	}
	return JustifyStart
}

// GetAlign returns the align-items from props.
func GetAlign(props gox.Props) Align {
	return getAlign(props)
}

func getAlign(props gox.Props) Align {
	if props == nil {
		return AlignStretch
	}
	v, ok := props["align"]
	if !ok {
		return AlignStretch
	}
	if s, ok := v.(string); ok {
		return Align(s)
	}
	if a, ok := v.(Align); ok {
		return a
	}
	return AlignStretch
}

func getPosition(props gox.Props) Position {
	if props == nil {
		return PositionRelative
	}
	v, ok := props["position"]
	if !ok {
		return PositionRelative
	}
	if s, ok := v.(string); ok {
		return Position(s)
	}
	if p, ok := v.(Position); ok {
		return p
	}
	return PositionRelative
}

func FilterChildren(node gox.VNode, typeStr string) []gox.VNode {
	var result []gox.VNode
	for _, child := range node.Children {
		if s, ok := TypeString(child); ok && s == typeStr {
			result = append(result, child)
		}
	}
	return result
}

// FilterRelativeChildren returns children with relative positioning.
func FilterRelativeChildren(node gox.VNode) []gox.VNode {
	return filterRelativeChildren(node)
}

func filterRelativeChildren(node gox.VNode) []gox.VNode {
	var result []gox.VNode
	for _, child := range node.Children {
		if getPosition(child.Props) != PositionAbsolute {
			result = append(result, child)
		}
	}
	return result
}

// FilterAbsoluteChildren returns children with absolute positioning.
func FilterAbsoluteChildren(node gox.VNode) []gox.VNode {
	return filterAbsoluteChildren(node)
}

func filterAbsoluteChildren(node gox.VNode) []gox.VNode {
	var result []gox.VNode
	for _, child := range node.Children {
		if getPosition(child.Props) == PositionAbsolute {
			result = append(result, child)
		}
	}
	return result
}
