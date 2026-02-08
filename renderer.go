// Package goli provides the main rendering orchestrator for terminal UI.
package goli

import (
	"io"
	"os"
	"strings"

	"github.com/germtb/gox"
)

// RendererInterface defines the common interface for all renderers.
type RendererInterface interface {
	Render(root gox.VNode)
}

// Options configures the renderer and app.
type Options struct {
	Width           int
	Height          int
	Output          io.Writer
	Pipeline        bool // Force pipeline renderer (auto-detected if not set)
	DisableThrottle bool // Disable frame rate limiting (for tests)
	OnRender        func()
	OnError         func(error)
}

// PipelineThreshold is the minimum cell count where the pipeline renderer helps.
// Below this, goroutine/channel overhead outweighs the parallelization benefit.
const PipelineThreshold = 3000 // ~80x40 or 60x50

// NewAuto creates the optimal renderer based on grid size.
// Uses pipeline renderer for larger grids (>3000 cells) and sequential for smaller ones.
func NewAuto(opts Options) RendererInterface {
	cells := opts.Width * opts.Height
	if opts.Pipeline || cells >= PipelineThreshold {
		return NewPipeline(opts)
	}
	return NewRenderer(opts)
}

// Renderer is the main orchestrator that ties everything together.
// Uses LogicalBuffer for content storage, transforms to visual rows for output.
type Renderer struct {
	width, height  int
	currentLogical *LogicalBuffer
	nextLogical    *LogicalBuffer
	currentVisual  *CellBuffer
	nextVisual     *CellBuffer
	output         io.Writer
	isFirstRender  bool
}

// NewRenderer creates a new renderer.
func NewRenderer(opts Options) *Renderer {
	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	return &Renderer{
		width:          opts.Width,
		height:         opts.Height,
		currentLogical: NewLogicalBuffer(opts.Height),
		nextLogical:    NewLogicalBuffer(opts.Height),
		currentVisual:  NewCellBuffer(opts.Width, opts.Height),
		nextVisual:     NewCellBuffer(opts.Width, opts.Height),
		output:         output,
		isFirstRender:  true,
	}
}

// Render renders a gox VNode tree to the terminal.
func (r *Renderer) Render(root gox.VNode) {
	// Increment memo generation for cache management
	BeginRender()

	// Clear next logical buffer
	r.nextLogical.Clear()

	// Compute layout
	ctx := LayoutContext{
		X:      0,
		Y:      0,
		Width:  r.width,
		Height: r.height,
	}
	layoutBox := ComputeLayout(root, ctx)

	// Render to logical buffer
	RenderToLogicalBuffer(layoutBox, r.nextLogical, nil)

	// Get actual content height (may exceed terminal height)
	contentHeight := r.nextLogical.Height()
	if layoutBox.Height > contentHeight {
		contentHeight = layoutBox.Height
	}

	// Clear next visual buffer (Clear() already sets all cells to EmptyCell)
	r.nextVisual.Clear()

	// Resize visual buffers if content exceeds current size
	if contentHeight > r.nextVisual.Height() {
		r.nextVisual = NewCellBuffer(r.width, contentHeight)
		r.currentVisual = NewCellBuffer(r.width, contentHeight)
	}

	// Convert logical to visual rows
	visualRows := r.nextLogical.ToVisualRows(r.width)

	// Copy all visual rows (no clipping at terminal height)
	for vy := 0; vy < len(visualRows.Rows); vy++ {
		row := visualRows.Rows[vy]
		for x := 0; x < len(row); x++ {
			r.nextVisual.Set(x, vy, row[x])
		}
	}

	// Diff and output
	if r.isFirstRender {
		io.WriteString(r.output, ClearScreen())
		r.isFirstRender = false
	}

	// Check if content exceeds terminal height - use sequential output for overflow
	if contentHeight > r.height {
		// Overflow mode: output entire buffer sequentially with newlines
		// ANSI cursor positioning doesn't work beyond terminal height
		ansiOutput := BufferToSequentialAnsi(r.nextVisual)
		io.WriteString(r.output, ansiOutput)
	} else {
		// Normal mode: use diff-based updates with cursor positioning
		changes := DiffBuffers(r.currentVisual, r.nextVisual)

		if len(changes) > 0 {
			runs := FindRuns(changes)
			ansiOutput := RunsToAnsi(runs)
			io.WriteString(r.output, ansiOutput)
		}
	}

	// Swap buffers
	r.currentLogical, r.nextLogical = r.nextLogical, r.currentLogical
	r.currentVisual, r.nextVisual = r.nextVisual, r.currentVisual
}

// Resize resizes the renderer.
func (r *Renderer) Resize(width, height int) {
	r.width = width
	r.height = height
	r.currentLogical = NewLogicalBuffer(height)
	r.nextLogical = NewLogicalBuffer(height)
	r.currentVisual = NewCellBuffer(width, height)
	r.nextVisual = NewCellBuffer(width, height)
	r.isFirstRender = true
}

// CurrentBuffer returns the current visual buffer (for testing).
func (r *Renderer) CurrentBuffer() *CellBuffer {
	return r.currentVisual
}

// Width returns the terminal width.
func (r *Renderer) Width() int {
	return r.width
}

// Height returns the terminal height.
func (r *Renderer) Height() int {
	return r.height
}

// PipelineRenderer uses a 4-stage concurrent pipeline for rendering.
// Each stage runs in its own goroutine:
//  1. Layout: VNode → LayoutBox
//  2. Buffer: LayoutBox → CellBuffer
//  3. Diff: CellBuffer → []CellChange → []CellRun → ANSI string
//  4. Output: ANSI string → io.Writer
type PipelineRenderer struct {
	width, height int
	output        io.Writer

	// Channels connecting pipeline stages
	layoutIn chan gox.VNode
	bufferIn chan *LayoutBox
	diffIn   chan *CellBuffer
	outputIn chan string

	// Stop signal
	stop chan struct{}
	done chan struct{}

	// Previous buffer for diffing (owned by diff stage)
	prevBuffer *CellBuffer
}

// NewPipeline creates a new pipelined renderer.
func NewPipeline(opts Options) *PipelineRenderer {
	output := opts.Output
	if output == nil {
		panic("PipelineRenderer requires an output writer")
	}

	p := &PipelineRenderer{
		width:      opts.Width,
		height:     opts.Height,
		output:     output,
		layoutIn:   make(chan gox.VNode, 2),
		bufferIn:   make(chan *LayoutBox, 2),
		diffIn:     make(chan *CellBuffer, 2),
		outputIn:   make(chan string, 2),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		prevBuffer: nil,
	}

	// Start pipeline stages
	go p.layoutStage()
	go p.bufferStage()
	go p.diffStage()
	go p.outputStage()

	return p
}

// layoutStage: VNode → LayoutBox
func (p *PipelineRenderer) layoutStage() {
	ctx := LayoutContext{
		X:      0,
		Y:      0,
		Width:  p.width,
		Height: p.height,
	}

	for {
		select {
		case <-p.stop:
			close(p.bufferIn)
			return
		case node := <-p.layoutIn:
			// Check for empty VNode (used as nil marker)
			if node.Type == nil {
				continue
			}
			layoutBox := ComputeLayout(node, ctx)
			p.bufferIn <- layoutBox
		}
	}
}

// bufferStage: LayoutBox → CellBuffer
// Uses a rotating pool of 4 buffers to avoid per-frame allocations.
// Pool size 4 ensures no buffer is reused while still referenced:
//   - 2 in channel capacity
//   - 1 being filled
//   - 1 held as prevBuffer by diffStage
func (p *PipelineRenderer) bufferStage() {
	const poolSize = 4

	// Pre-allocate buffer pool
	logicalPool := make([]*LogicalBuffer, poolSize)
	visualPool := make([]*CellBuffer, poolSize)
	for i := 0; i < poolSize; i++ {
		logicalPool[i] = NewLogicalBuffer(p.height)
		visualPool[i] = NewCellBuffer(p.width, p.height)
	}
	poolIdx := 0

	for {
		select {
		case <-p.stop:
			close(p.diffIn)
			return
		case layoutBox, ok := <-p.bufferIn:
			if !ok {
				close(p.diffIn)
				return
			}
			if layoutBox == nil {
				continue
			}

			// Get next buffer from pool (rotating)
			logicalBuf := logicalPool[poolIdx]
			visualBuf := visualPool[poolIdx]
			poolIdx = (poolIdx + 1) % poolSize

			// Clear and reuse
			logicalBuf.Clear()
			visualBuf.Clear()

			// Render to logical buffer
			RenderToLogicalBuffer(layoutBox, logicalBuf, nil)

			// Convert logical to visual
			visualRows := logicalBuf.ToVisualRows(p.width)
			for vy := 0; vy < len(visualRows.Rows) && vy < p.height; vy++ {
				row := visualRows.Rows[vy]
				for x := 0; x < len(row); x++ {
					visualBuf.Set(x, vy, row[x])
				}
			}

			p.diffIn <- visualBuf
		}
	}
}

// diffStage: CellBuffer → ANSI string
// Uses pre-allocated slices for diff results.
func (p *PipelineRenderer) diffStage() {
	isFirst := true

	// Pre-allocate reusable slices for diff results
	// Estimate: 20% of cells change per frame on average
	estimatedChanges := (p.width * p.height) / 5
	if estimatedChanges < 64 {
		estimatedChanges = 64
	}
	changes := make([]CellChange, 0, estimatedChanges)
	runs := make([]CellRun, 0, estimatedChanges/4)

	// Pre-allocate string builder
	var sb strings.Builder
	sb.Grow(estimatedChanges * 20)

	for {
		select {
		case <-p.stop:
			close(p.outputIn)
			return
		case currentBuf, ok := <-p.diffIn:
			if !ok {
				close(p.outputIn)
				return
			}
			if currentBuf == nil {
				continue
			}

			// Clear and reuse slices
			changes = changes[:0]
			runs = runs[:0]
			sb.Reset()

			if isFirst || p.prevBuffer == nil {
				// First frame: clear screen and output everything
				sb.WriteString(ClearScreen())
				// Create a blank buffer to diff against (only on first frame)
				blankBuf := NewCellBuffer(p.width, p.height)
				changes = DiffBuffersInto(blankBuf, currentBuf, changes)
				if len(changes) > 0 {
					runs = FindRunsInto(changes, runs)
					RunsToAnsiBuilder(runs, &sb)
				}
				isFirst = false
			} else {
				// Subsequent frames: diff against previous
				changes = DiffBuffersInto(p.prevBuffer, currentBuf, changes)
				if len(changes) > 0 {
					runs = FindRunsInto(changes, runs)
					RunsToAnsiBuilder(runs, &sb)
				}
			}

			// Keep current buffer for next diff
			p.prevBuffer = currentBuf

			if sb.Len() > 0 {
				p.outputIn <- sb.String()
			}
		}
	}
}

// outputStage: ANSI string → io.Writer
func (p *PipelineRenderer) outputStage() {
	for {
		select {
		case <-p.stop:
			close(p.done)
			return
		case ansiStr, ok := <-p.outputIn:
			if !ok {
				close(p.done)
				return
			}
			io.WriteString(p.output, ansiStr)
		}
	}
}

// Render submits a frame to the pipeline (non-blocking if pipeline has capacity).
func (p *PipelineRenderer) Render(root gox.VNode) {
	select {
	case p.layoutIn <- root:
	default:
		// Pipeline full - drop frame (could also block here)
	}
}

// RenderBlocking submits a frame and waits until it enters the pipeline.
func (p *PipelineRenderer) RenderBlocking(root gox.VNode) {
	p.layoutIn <- root
}

// Stop shuts down the pipeline gracefully.
func (p *PipelineRenderer) Stop() {
	close(p.stop)
	<-p.done
}
