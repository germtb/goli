// Package goli provides a link primitive for clickable URLs.
package goli

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/germtb/gox"
)

func init() {
	RegisterIntrinsic("link", &IntrinsicHandler{
		Measure:       measureLink,
		Layout:        layoutLink,
		Render:        RenderLinkToBuffer,
		RenderLogical: RenderLinkToLogicalBuffer,
	})
}

// LinkOptions configures link creation.
type LinkOptions struct {
	// URL is the target URL to open.
	URL string
	// OnClick is called when the link is activated (in addition to opening URL).
	OnClick func()
	// DisableFocus disables focus management registration.
	DisableFocus bool
}

// Link represents a clickable hyperlink component.
type Link struct {
	focused    Accessor[bool]
	setFocused Setter[bool]

	url            string
	onClick        func()
	shouldRegister bool
	registered     bool
}

// NewLink creates a new link.
func NewLink(opts LinkOptions) *Link {
	focused, setFocused := CreateSignal(false)

	shouldRegister := true
	if opts.DisableFocus {
		shouldRegister = false
	}

	l := &Link{
		focused:        focused,
		setFocused:     setFocused,
		url:            opts.URL,
		onClick:        opts.OnClick,
		shouldRegister: shouldRegister,
	}

	if shouldRegister {
		Register(l)
		l.registered = true
	}

	return l
}

// URL returns the link's target URL.
func (l *Link) URL() string {
	return l.url
}

// SetURL updates the link's target URL.
func (l *Link) SetURL(url string) {
	l.url = url
}

// Focused returns whether the link is focused.
func (l *Link) Focused() bool {
	return l.focused()
}

// Focus gives focus to this link.
func (l *Link) Focus() {
	RequestFocus(l)
}

// Blur removes focus from this link.
func (l *Link) Blur() {
	RequestBlur(l)
}

// SetFocused sets the focused state (called by focus manager).
func (l *Link) SetFocused(f bool) {
	l.setFocused(f)
}

// Dispose unregisters from the focus manager.
func (l *Link) Dispose() {
	if l.registered {
		Unregister(l)
		l.registered = false
	}
}

// HandleKey processes a key press.
// Returns true if the key was consumed.
func (l *Link) HandleKey(key string) bool {
	if !l.focused() {
		return false
	}

	// Handle Enter/Space to activate link
	switch key {
	case Enter, EnterLF, Space:
		l.Activate()
		return true
	}

	return false
}

// Activate opens the URL and calls the onClick handler.
func (l *Link) Activate() {
	if l.url != "" {
		OpenURL(l.url)
	}
	if l.onClick != nil {
		l.onClick()
	}
}

// OpenURL opens the given URL in the default browser.
// Works on macOS, Linux, and Windows.
func OpenURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		// Try xdg-open as fallback
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

// Link measure/layout/render functions

func measureLink(node gox.VNode, ctx *LayoutContext) (int, int) {
	// Measure text content
	text := CollectTextContent(node)
	lines := splitLines(text)

	width := 0
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}

	return width, len(lines)
}

func layoutLink(node gox.VNode, availWidth, availHeight int, ctx *LayoutContext) *LayoutBox {
	w, h := measureLink(node, ctx)

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

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

// RenderLinkToBuffer renders a link to a CellBuffer.
// Links use OSC 8 escape sequences for terminal hyperlinks.
func RenderLinkToBuffer(box *LayoutBox, buf *CellBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

	linkPrim := node.Props["url"]
	baseStyle := GetStyle(node.Props)
	focusedStyle := getStyleProp(node.Props, "focusedStyle", Style{Bold: true})

	// Default link style: blue and underlined
	if baseStyle.Color == ColorNone {
		baseStyle.Color = ColorBlue
	}
	baseStyle.Underline = true

	isFocused := false
	url := ""
	if lnk, ok := linkPrim.(interface {
		Focused() bool
		URL() string
	}); ok {
		isFocused = lnk.Focused()
		url = lnk.URL()
	}

	computedStyle := baseStyle
	if isFocused {
		computedStyle = baseStyle.Merge(focusedStyle)
	}

	// Set OSC 8 URL for the link (stored in style for ANSI output)
	computedStyle.HyperlinkURL = url

	text := CollectTextContent(node)
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		lineY := y + lineIdx
		if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
			continue
		}

		charX := x
		for _, char := range line {
			if IsInClip(charX, lineY, clip) {
				buf.Set(charX, lineY, New(char, computedStyle))
			}
			charX++
		}
	}
}

// RenderLinkToLogicalBuffer renders a link to a LogicalBuffer.
func RenderLinkToLogicalBuffer(box *LayoutBox, buf *LogicalBuffer, clip *ClipRegion) {
	node := box.Node
	x, y := box.X, box.Y

	linkPrim := node.Props["url"]
	baseStyle := GetStyle(node.Props)
	focusedStyle := getStyleProp(node.Props, "focusedStyle", Style{Bold: true})

	// Default link style: blue and underlined
	if baseStyle.Color == ColorNone {
		baseStyle.Color = ColorBlue
	}
	baseStyle.Underline = true

	isFocused := false
	url := ""
	if lnk, ok := linkPrim.(interface {
		Focused() bool
		URL() string
	}); ok {
		isFocused = lnk.Focused()
		url = lnk.URL()
	}

	computedStyle := baseStyle
	if isFocused {
		computedStyle = baseStyle.Merge(focusedStyle)
	}

	// Set OSC 8 URL for the link
	computedStyle.HyperlinkURL = url

	text := CollectTextContent(node)
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		lineY := y + lineIdx
		if clip != nil && (lineY < clip.MinY || lineY >= clip.MaxY) {
			continue
		}

		charX := x
		for _, char := range line {
			if IsInClip(charX, lineY, clip) {
				buf.Set(charX, lineY, New(char, computedStyle))
			}
			charX++
		}
	}
}
