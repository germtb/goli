// Package goli provides VNode helper functions.
package goli

import (
	"github.com/germtb/gox"
)

// VNode is an alias for gox.VNode - no wrapper needed.
type VNode = gox.VNode

// Props is an alias for gox.Props.
type Props = gox.Props

// IsTextNode returns true if this is a text node.
func IsTextNode(v gox.VNode) bool {
	s, ok := v.Type.(string)
	return ok && s == gox.TextNodeType
}

// GetTextContent returns the text content if this is a text node.
func GetTextContent(v gox.VNode) (string, bool) {
	if !IsTextNode(v) {
		return "", false
	}
	if content, ok := v.Props["content"].(string); ok {
		return content, true
	}
	if text, ok := v.Props["text"].(string); ok {
		return text, true
	}
	return "", false
}

// TypeString returns the type as a string (for intrinsic elements).
func TypeString(v gox.VNode) (string, bool) {
	s, ok := v.Type.(string)
	return s, ok
}

// CreateTextNode creates a text node.
func CreateTextNode(text string) gox.VNode {
	return gox.VNode{
		Type:     gox.TextNodeType,
		Props:    gox.Props{"text": text, "content": text},
		Children: nil,
	}
}

// Expand recursively expands functional components into their rendered output.
func Expand(v gox.VNode) gox.VNode {
	// If it's a text node or intrinsic element, just expand children
	if _, ok := TypeString(v); ok {
		if len(v.Children) == 0 {
			return v
		}

		expandedChildren := make([]gox.VNode, len(v.Children))
		for i, child := range v.Children {
			expandedChildren[i] = Expand(child)
		}

		return gox.VNode{
			Type:     v.Type,
			Props:    v.Props,
			Children: expandedChildren,
		}
	}

	// It's a functional component
	if comp, ok := v.Type.(gox.Component); ok {
		// Create props with children
		props := gox.Props{}
		for k, val := range v.Props {
			props[k] = val
		}
		props["children"] = v.Children

		result := comp(props)
		return Expand(result)
	}

	return v
}
