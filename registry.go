// Package goli provides intrinsic element registration.
package goli

import (
	"sync"

	"github.com/germtb/gox"
)

// IntrinsicLayoutFunc handles layout for an intrinsic element type.
// It receives the node, available width/height, and layout context.
// Returns the computed LayoutBox for this element.
type IntrinsicLayoutFunc func(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox

// IntrinsicMeasureFunc measures the intrinsic size of an element.
// Returns (width, height).
type IntrinsicMeasureFunc func(node gox.VNode, ctx *LayoutContext) (int, int)

// IntrinsicRenderFunc renders an element to a CellBuffer.
type IntrinsicRenderFunc func(box *LayoutBox, buf *CellBuffer, clip *ClipRegion)

// IntrinsicRenderLogicalFunc renders an element to a LogicalBuffer.
type IntrinsicRenderLogicalFunc func(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion)

// IntrinsicHandler defines how to layout and render an intrinsic element type.
type IntrinsicHandler struct {
	// Layout computes the layout for this element type.
	// If nil, default box layout is used.
	Layout IntrinsicLayoutFunc

	// Measure returns the intrinsic size of this element.
	// If nil, size is determined by props or children.
	Measure IntrinsicMeasureFunc

	// Render draws this element to a CellBuffer.
	// If nil, children are rendered with default box behavior.
	Render IntrinsicRenderFunc

	// RenderLogical draws this element to a LogicalBuffer.
	// If nil, children are rendered with default box behavior.
	RenderLogical IntrinsicRenderLogicalFunc
}

var (
	intrinsicRegistry = make(map[string]*IntrinsicHandler)
	registryMu        sync.RWMutex
)

// RegisterIntrinsic registers a handler for an intrinsic element type.
// This should be called from init() functions in component packages.
// The name corresponds to the JSX element name (e.g., "input", "select").
func RegisterIntrinsic(name string, handler *IntrinsicHandler) {
	registryMu.Lock()
	defer registryMu.Unlock()
	intrinsicRegistry[name] = handler
}

// GetIntrinsicHandler returns the handler for an intrinsic element type.
// Returns nil if no handler is registered.
func GetIntrinsicHandler(name string) *IntrinsicHandler {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return intrinsicRegistry[name]
}

// HasIntrinsicHandler returns true if a handler is registered for the given type.
func HasIntrinsicHandler(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := intrinsicRegistry[name]
	return ok
}
