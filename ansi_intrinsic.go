package goli

import (
	"strings"

	"github.com/germtb/gox"
	"github.com/mattn/go-runewidth"
)

func init() {
	RegisterIntrinsic("ansi", &IntrinsicHandler{
		Measure:       measureAnsi,
		Layout:        layoutAnsi,
		Render:        renderAnsi,
		RenderLogical: renderAnsiLogical,
	})
}

func measureAnsi(node gox.VNode, ctx *LayoutContext) (int, int) {
	text := CollectTextContent(node)
	lines := strings.Split(text, "\n")
	maxWidth := 0
	for _, line := range lines {
		w := RuneWidth(line) // RuneWidth already strips ANSI
		if w > maxWidth {
			maxWidth = w
		}
	}
	margin := GetSpacing(node.Props, "margin")
	return maxWidth + margin.Left + margin.Right, len(lines) + margin.Top + margin.Bottom
}

func layoutAnsi(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	text := CollectTextContent(node)
	shouldWrap := GetBoolProp(node.Props, "wrap", false)
	margin := GetSpacing(node.Props, "margin")

	contentWidth := availWidth - margin.Left - margin.Right
	if contentWidth < 0 {
		contentWidth = 0
	}

	var lines []string
	if shouldWrap {
		lines = WrapText(text, contentWidth)
	} else {
		lines = strings.Split(text, "\n")
	}

	maxWidth := 0
	for _, line := range lines {
		w := RuneWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}

	w := min(maxWidth, contentWidth)
	h := len(lines)

	// Create a synthetic "ansi" node (NOT a text node, so the render
	// dispatch reaches the ansi handler instead of the text path).
	syntheticNode := gox.VNode{
		Type:  "ansi",
		Props: gox.Props{"content": strings.Join(lines, "\n")},
	}
	if v, ok := node.Props["style"]; ok {
		syntheticNode.Props["style"] = v
	}
	for _, key := range styleAttributeKeys {
		if v, ok := node.Props[key]; ok {
			syntheticNode.Props[key] = v
		}
	}

	boxX := ctx.X + margin.Left
	boxY := ctx.Y + margin.Top

	return &LayoutBox{
		X:           boxX,
		Y:           boxY,
		Width:       w,
		Height:      h,
		InnerX:      boxX,
		InnerY:      boxY,
		InnerWidth:  w,
		InnerHeight: h,
		Node:        syntheticNode,
		Children:    nil,
		ZIndex:      GetIntProp(node.Props, "zIndex", 0),
	}
}

func getAnsiContent(node gox.VNode) string {
	if s, ok := node.Props["content"].(string); ok {
		return s
	}
	return CollectTextContent(node)
}

func renderAnsi(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

	baseStyle := GetStyle(node.Props)
	text := getAnsiContent(node)
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		lineY := y + lineIdx
		if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
			continue
		}

		segments := ParseAnsiLine(line, baseStyle)
		charX := x
		for _, seg := range segments {
			for _, char := range seg.Text {
				if IsInClip(charX, lineY, clip) {
					buf.SetCharMerge(charX, lineY, char, seg.Style)
				}
				charX += runewidth.RuneWidth(char)
			}
		}
	}
}

func renderAnsiLogical(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

	baseStyle := GetStyle(node.Props)
	text := getAnsiContent(node)
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		lineY := y + lineIdx
		if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
			continue
		}

		segments := ParseAnsiLine(line, baseStyle)
		charX := x
		for _, seg := range segments {
			for _, char := range seg.Text {
				if IsInClip(charX, lineY, clip) {
					buf.SetMerge(charX, lineY, New(char, seg.Style))
				}
				charX += runewidth.RuneWidth(char)
			}
		}
	}
}
