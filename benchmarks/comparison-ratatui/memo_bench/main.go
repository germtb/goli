// Benchmark comparing naive vs memoized components
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

const (
	Rows       = 50
	Cols       = 200
	TotalCells = Rows * Cols
	Iterations = 50
)


// --- Naive approach: recreates all VNodes every frame ---

type CellPropsNaive struct {
	Index       int
	IsHighlight bool
}

func CellNaive(props CellPropsNaive, children ...gox.VNode) gox.VNode {
	style := map[string]any{"color": "white"}
	char := "·"
	if props.IsHighlight {
		style["color"] = "cyan"
		style["bold"] = true
		char = "█"
	}
	return gox.Element("text", gox.Props{"style": style}, gox.Text(char))
}

func generateGridNaive(highlight int) gox.VNode {
	var rowNodes []gox.VNode
	for r := 0; r < Rows; r++ {
		var cells []gox.VNode
		for c := 0; c < Cols; c++ {
			idx := r*Cols + c
			cells = append(cells, CellNaive(CellPropsNaive{
				Index:       idx,
				IsHighlight: idx == highlight,
			}))
		}
		rowNodes = append(rowNodes, gox.Element("box", gox.Props{"direction": "row"}, cells...))
	}
	return gox.Element("box", gox.Props{"direction": "column"}, rowNodes...)
}

// --- Memoized approach: skips unchanged cells ---

type CellPropsMemo struct {
	Key         int
	Index       int
	IsHighlight bool
}

func (p CellPropsMemo) GetKey() int { return p.Key }

// MemoizedCell uses shallow equality (==) - fast like React.memo
var MemoizedCell = goli.Memo(
	func(props CellPropsMemo, children ...gox.VNode) gox.VNode {
		style := map[string]any{"color": "white"}
		char := "·"
		if props.IsHighlight {
			style["color"] = "cyan"
			style["bold"] = true
			char = "█"
		}
		return gox.Element("text", gox.Props{"style": style}, gox.Text(char))
	},
	goli.ShallowEquals[CellPropsMemo],
)

func generateGridMemo(highlight int) gox.VNode {
	var rowNodes []gox.VNode
	for r := 0; r < Rows; r++ {
		var cells []gox.VNode
		for c := 0; c < Cols; c++ {
			idx := r*Cols + c
			cells = append(cells, MemoizedCell(CellPropsMemo{
				Key:         idx,
				Index:       idx,
				IsHighlight: idx == highlight,
			}))
		}
		rowNodes = append(rowNodes, gox.Element("box", gox.Props{"direction": "row"}, cells...))
	}
	return gox.Element("box", gox.Props{"direction": "column"}, rowNodes...)
}


func benchmark(name string, fn func()) time.Duration {
	// Warmup
	for i := 0; i < 3; i++ {
		fn()
	}

	start := time.Now()
	for i := 0; i < Iterations; i++ {
		fn()
	}
	elapsed := time.Since(start)
	avg := elapsed / Iterations
	fps := float64(time.Second) / float64(avg)
	fmt.Printf("%-35s avg: %8.3fms  (%6.0f FPS)\n", name, float64(avg.Microseconds())/1000, fps)
	return avg
}

func main() {
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println("Memoization Benchmark: goli.Memo() vs Naive")
	fmt.Printf("Grid: %d rows × %d cols = %d cells\n", Rows, Cols, TotalCells)
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println()

	// --- VNode Creation Only ---
	fmt.Println("VNode Creation (no layout/render):")
	fmt.Println(strings.Repeat("-", 65))

	frameNum := 0
	benchmark("Naive: Create all VNodes", func() {
		generateGridNaive(frameNum % TotalCells)
		frameNum++
	})

	frameNum = 0
	// Prime the memo cache
	generateGridMemo(0)
	goli.BeginRender()

	benchmark("Memo (int key)", func() {
		goli.BeginRender()
		generateGridMemo(frameNum % TotalCells)
		frameNum++
	})

	fmt.Println()

	// --- Full Pipeline ---
	fmt.Println("Full Pipeline (create + layout + render + diff):")
	fmt.Println(strings.Repeat("-", 65))

	var output strings.Builder

	r1 := goli.NewRenderer(goli.Options{Width: Cols, Height: Rows, Output: &output})
	frameNum = 0
	benchmark("Naive: Full pipeline", func() {
		tree := generateGridNaive(frameNum % TotalCells)
		r1.Render(tree)
		frameNum++
		output.Reset()
	})

	r2 := goli.NewRenderer(goli.Options{Width: Cols, Height: Rows, Output: &output})
	frameNum = 0
	// Prime
	r2.Render(generateGridMemo(0))
	output.Reset()

	benchmark("Memo (int key)", func() {
		tree := generateGridMemo(frameNum % TotalCells)
		r2.Render(tree)
		frameNum++
		output.Reset()
	})

	fmt.Println()
	fmt.Println(strings.Repeat("=", 65))
}
