package goli

import "github.com/germtb/gox"

func init() {
	RegisterIntrinsic("spacer", &IntrinsicHandler{
		Measure:       measureSpacer,
		Layout:        layoutSpacer,
		Render:        renderSpacer,
		RenderLogical: renderSpacerLogical,
	})
}

func measureSpacer(node gox.VNode, ctx *LayoutContext) (int, int) {
	w := GetIntProp(node.Props, "width", 0)
	h := GetIntProp(node.Props, "height", 0)
	return w, h
}

func layoutSpacer(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	w := GetIntProp(node.Props, "width", 0)
	h := GetIntProp(node.Props, "height", 0)

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
	}
}

// renderSpacer is a no-op — spacers are invisible.
func renderSpacer(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {}

// renderSpacerLogical is a no-op — spacers are invisible.
func renderSpacerLogical(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {}
