// Package goli provides the diff engine for comparing cell buffers.
package goli

import (
	"sort"
)

// CellChange represents a change at a specific position.
type CellChange struct {
	X    int
	Y    int
	Cell Cell
}

// DiffBuffers computes the diff between two buffers.
// Returns an array of cell changes needed to transform `from` into `to`.
func DiffBuffers(from, to *CellBuffer) []CellChange {
	// Get minimum dimensions
	width := min(from.Width(), to.Width())
	height := min(from.Height(), to.Height())

	// Pre-allocate with estimated capacity (assume ~20% of cells change)
	estimated := (to.Width() * to.Height()) / 5
	if estimated < 64 {
		estimated = 64
	}
	changes := make([]CellChange, 0, estimated)

	// Compare overlapping region
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fromCell := from.Get(x, y)
			toCell := to.Get(x, y)

			if !fromCell.Equal(toCell) {
				changes = append(changes, CellChange{X: x, Y: y, Cell: toCell})
			}
		}
	}

	// Handle new rows (if `to` is taller)
	for y := height; y < to.Height(); y++ {
		for x := 0; x < to.Width(); x++ {
			changes = append(changes, CellChange{X: x, Y: y, Cell: to.Get(x, y)})
		}
	}

	// Handle new columns (if `to` is wider) - only for existing rows
	for y := 0; y < height; y++ {
		for x := width; x < to.Width(); x++ {
			changes = append(changes, CellChange{X: x, Y: y, Cell: to.Get(x, y)})
		}
	}

	return changes
}

// GroupChangesByRow groups changes by row for more efficient cursor movement.
func GroupChangesByRow(changes []CellChange) map[int][]CellChange {
	byRow := make(map[int][]CellChange)

	for _, change := range changes {
		byRow[change.Y] = append(byRow[change.Y], change)
	}

	// Sort each row by x coordinate
	for _, row := range byRow {
		sort.Slice(row, func(i, j int) bool {
			return row[i].X < row[j].X
		})
	}

	return byRow
}

// FindRuns detects consecutive runs in changes for efficient output.
// A run is a sequence of consecutive x positions.
func FindRuns(changes []CellChange) []CellRun {
	if len(changes) == 0 {
		return nil
	}

	byRow := GroupChangesByRow(changes)

	// Pre-allocate runs slice (estimate: changes / 4 runs on average)
	runs := make([]CellRun, 0, len(changes)/4+1)

	// Get sorted row keys for deterministic order
	rows := make([]int, 0, len(byRow))
	for y := range byRow {
		rows = append(rows, y)
	}
	sort.Ints(rows)

	for _, y := range rows {
		rowChanges := byRow[y]
		var currentRun *CellRun

		for _, change := range rowChanges {
			if currentRun != nil && change.X == currentRun.X+len(currentRun.Cells) {
				// Consecutive: extend the run
				currentRun.Cells = append(currentRun.Cells, change.Cell)
			} else {
				// Start a new run
				if currentRun != nil {
					runs = append(runs, *currentRun)
				}
				// Pre-allocate cells slice for the run
				cells := make([]Cell, 1, 16)
				cells[0] = change.Cell
				currentRun = &CellRun{X: change.X, Y: y, Cells: cells}
			}
		}

		if currentRun != nil {
			runs = append(runs, *currentRun)
		}
	}

	return runs
}

// DiffBuffersInto computes the diff between two buffers, appending to the provided slice.
// This avoids allocation when the caller pre-allocates the result slice.
func DiffBuffersInto(from, to *CellBuffer, result []CellChange) []CellChange {
	width := min(from.Width(), to.Width())
	height := min(from.Height(), to.Height())

	// Compare overlapping region
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fromCell := from.Get(x, y)
			toCell := to.Get(x, y)

			if !fromCell.Equal(toCell) {
				result = append(result, CellChange{X: x, Y: y, Cell: toCell})
			}
		}
	}

	// Handle new rows (if `to` is taller)
	for y := height; y < to.Height(); y++ {
		for x := 0; x < to.Width(); x++ {
			result = append(result, CellChange{X: x, Y: y, Cell: to.Get(x, y)})
		}
	}

	// Handle new columns (if `to` is wider) - only for existing rows
	for y := 0; y < height; y++ {
		for x := width; x < to.Width(); x++ {
			result = append(result, CellChange{X: x, Y: y, Cell: to.Get(x, y)})
		}
	}

	return result
}

// FindRunsInto detects consecutive runs in changes, appending to the provided slice.
// This avoids allocation when the caller pre-allocates the result slice.
func FindRunsInto(changes []CellChange, result []CellRun) []CellRun {
	if len(changes) == 0 {
		return result
	}

	byRow := GroupChangesByRow(changes)

	// Get sorted row keys for deterministic order
	rows := make([]int, 0, len(byRow))
	for y := range byRow {
		rows = append(rows, y)
	}
	sort.Ints(rows)

	for _, y := range rows {
		rowChanges := byRow[y]
		var currentRun *CellRun

		for _, change := range rowChanges {
			if currentRun != nil && change.X == currentRun.X+len(currentRun.Cells) {
				// Consecutive: extend the run
				currentRun.Cells = append(currentRun.Cells, change.Cell)
			} else {
				// Start a new run
				if currentRun != nil {
					result = append(result, *currentRun)
				}
				// Pre-allocate cells slice for the run
				cells := make([]Cell, 1, 16)
				cells[0] = change.Cell
				currentRun = &CellRun{X: change.X, Y: y, Cells: cells}
			}
		}

		if currentRun != nil {
			result = append(result, *currentRun)
		}
	}

	return result
}
