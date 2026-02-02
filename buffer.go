// Package goli provides buffer implementations for terminal rendering.
package goli

import "strings"

// MaxBufferHeight is the maximum height a LogicalBuffer can auto-grow to.
// This prevents runaway memory usage from unbounded growth.
// 10,000 lines is generous for most TUI applications.
const MaxBufferHeight = 10000

// CellBuffer is a fixed-size 2D grid of cells representing the terminal screen.
// This is the core data structure for diffing.
type CellBuffer struct {
	width, height int
	cells         []Cell
}

// NewCellBuffer creates a new buffer filled with empty cells.
func NewCellBuffer(width, height int) *CellBuffer {
	cells := make([]Cell, width*height)
	for i := range cells {
		cells[i] = EmptyCell
	}
	return &CellBuffer{
		width:  width,
		height: height,
		cells:  cells,
	}
}

func (b *CellBuffer) index(x, y int) int {
	return y*b.width + x
}

func (b *CellBuffer) inBounds(x, y int) bool {
	return x >= 0 && x < b.width && y >= 0 && y < b.height
}

// Width returns the buffer width.
func (b *CellBuffer) Width() int { return b.width }

// Height returns the buffer height.
func (b *CellBuffer) Height() int { return b.height }

// Get returns the cell at (x, y), or EmptyCell if out of bounds.
func (b *CellBuffer) Get(x, y int) Cell {
	if !b.inBounds(x, y) {
		return EmptyCell
	}
	return b.cells[b.index(x, y)]
}

// Set sets the cell at (x, y). Does nothing if out of bounds.
func (b *CellBuffer) Set(x, y int, c Cell) {
	if !b.inBounds(x, y) {
		return
	}
	b.cells[b.index(x, y)] = c
}

// SetChar sets a character with style at (x, y).
func (b *CellBuffer) SetChar(x, y int, char rune, style Style) {
	b.Set(x, y, New(char, style))
}

// SetCharMerge sets a character, merging style with existing cell.
// Preserves background if the new style doesn't specify one.
func (b *CellBuffer) SetCharMerge(x, y int, char rune, style Style) {
	if !b.inBounds(x, y) {
		return
	}
	existing := b.Get(x, y)
	mergedStyle := existing.Style.Merge(style)
	// Preserve background if new style doesn't have one
	if !style.HasBackground() && existing.Style.HasBackground() {
		mergedStyle.Background = existing.Style.Background
		mergedStyle.BackgroundRGB = existing.Style.BackgroundRGB
	}
	b.Set(x, y, New(char, mergedStyle))
}

// WriteString writes a string starting at (x, y), going right.
// Text is clipped at buffer edge. Returns number of characters written.
func (b *CellBuffer) WriteString(x, y int, text string, style Style) int {
	if y < 0 || y >= b.height {
		return 0
	}

	written := 0
	col := x
	for _, char := range text {
		if col < 0 {
			col++
			continue
		}
		if col >= b.width {
			break
		}
		b.SetChar(col, y, char, style)
		written++
		col++
	}
	return written
}

// Clear clears the entire buffer with empty cells.
func (b *CellBuffer) Clear() {
	for i := range b.cells {
		b.cells[i] = EmptyCell
	}
}

// ToDebugString returns a debug string representation (characters only).
func (b *CellBuffer) ToDebugString() string {
	var sb strings.Builder
	for y := 0; y < b.height; y++ {
		if y > 0 {
			sb.WriteRune('\n')
		}
		for x := 0; x < b.width; x++ {
			sb.WriteRune(b.Get(x, y).Char)
		}
	}
	return sb.String()
}

// LogicalRow is a variable-length array of cells.
type LogicalRow struct {
	Cells []Cell
}

// LogicalBuffer stores content as logical rows with arbitrary length.
// Terminal wrapping is handled at render time, not storage time.
type LogicalBuffer struct {
	rows   []LogicalRow
	height int
}

// NewLogicalBuffer creates a new logical buffer with the given height.
func NewLogicalBuffer(height int) *LogicalBuffer {
	rows := make([]LogicalRow, height)
	for i := range rows {
		rows[i] = LogicalRow{Cells: nil}
	}
	return &LogicalBuffer{
		rows:   rows,
		height: height,
	}
}

// Height returns the number of logical rows.
func (b *LogicalBuffer) Height() int { return b.height }

// Get returns the cell at logical position (x, y).
// Returns EmptyCell if out of bounds.
func (b *LogicalBuffer) Get(x, y int) Cell {
	if y < 0 || y >= b.height {
		return EmptyCell
	}
	row := b.rows[y]
	if x < 0 || x >= len(row.Cells) {
		return EmptyCell
	}
	return row.Cells[x]
}

// Set sets the cell at logical position (x, y).
// Extends the row if needed. Grows the buffer if y exceeds current height.
// Will not grow beyond MaxBufferHeight.
func (b *LogicalBuffer) Set(x, y int, c Cell) {
	if x < 0 || y < 0 || y >= MaxBufferHeight {
		return
	}
	// Grow buffer if needed
	for y >= b.height {
		b.rows = append(b.rows, LogicalRow{Cells: nil})
		b.height++
	}
	row := &b.rows[y]

	// Extend row with empty cells if needed
	for len(row.Cells) <= x {
		row.Cells = append(row.Cells, EmptyCell)
	}
	row.Cells[x] = c
}

// SetMerge sets a cell, merging style with existing cell.
// Preserves background color if the new style doesn't specify one.
// Grows the buffer if y exceeds current height. Will not grow beyond MaxBufferHeight.
func (b *LogicalBuffer) SetMerge(x, y int, c Cell) {
	if x < 0 || y < 0 || y >= MaxBufferHeight {
		return
	}
	// Grow buffer if needed
	for y >= b.height {
		b.rows = append(b.rows, LogicalRow{Cells: nil})
		b.height++
	}
	row := &b.rows[y]

	// Extend row with empty cells if needed
	for len(row.Cells) <= x {
		row.Cells = append(row.Cells, EmptyCell)
	}

	existing := row.Cells[x]
	mergedStyle := existing.Style.Merge(c.Style)
	// Preserve background if new style doesn't have one
	if !c.Style.HasBackground() && existing.Style.HasBackground() {
		mergedStyle.Background = existing.Style.Background
		mergedStyle.BackgroundRGB = existing.Style.BackgroundRGB
	}
	row.Cells[x] = New(c.Char, mergedStyle)
}

// RowLength returns the length of a logical row.
func (b *LogicalBuffer) RowLength(y int) int {
	if y < 0 || y >= b.height {
		return 0
	}
	return len(b.rows[y].Cells)
}

// GetRow returns a logical row.
func (b *LogicalBuffer) GetRow(y int) *LogicalRow {
	if y < 0 || y >= b.height {
		return nil
	}
	return &b.rows[y]
}

// WriteString writes a string starting at (x, y).
// The row extends as needed - no clipping.
func (b *LogicalBuffer) WriteString(x, y int, text string, style Style) {
	if y < 0 || y >= b.height {
		return
	}
	col := x
	for _, char := range text {
		b.Set(col, y, New(char, style))
		col++
	}
}

// ClearRow clears a row.
func (b *LogicalBuffer) ClearRow(y int) {
	if y < 0 || y >= b.height {
		return
	}
	b.rows[y] = LogicalRow{Cells: nil}
}

// Clear clears the entire buffer.
func (b *LogicalBuffer) Clear() {
	for y := 0; y < b.height; y++ {
		b.rows[y] = LogicalRow{Cells: nil}
	}
}

// VisualRows holds the result of transforming logical rows to visual rows.
type VisualRows struct {
	Rows            [][]Cell // Visual rows
	LogicalToVisual []int    // LogicalToVisual[logicalY] = first visual row index
}

// ToVisualRows transforms logical rows to visual rows based on terminal width.
func (b *LogicalBuffer) ToVisualRows(terminalWidth int) VisualRows {
	visualRows := make([][]Cell, 0)
	logicalToVisual := make([]int, b.height)

	for y := 0; y < b.height; y++ {
		logicalToVisual[y] = len(visualRows)

		row := b.rows[y]
		if len(row.Cells) == 0 {
			// Empty logical row = one empty visual row
			visualRows = append(visualRows, []Cell{})
		} else {
			// Split into chunks of terminalWidth
			for i := 0; i < len(row.Cells); i += terminalWidth {
				end := i + terminalWidth
				if end > len(row.Cells) {
					end = len(row.Cells)
				}
				chunk := make([]Cell, end-i)
				copy(chunk, row.Cells[i:end])
				visualRows = append(visualRows, chunk)
			}
		}
	}

	return VisualRows{
		Rows:            visualRows,
		LogicalToVisual: logicalToVisual,
	}
}
