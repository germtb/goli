// Scaling test - see how performance changes with grid size
//
// Run with: go run ./examples/perf-scaling/
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

func main() {
	fmt.Println("=== Scaling Test ===")
	fmt.Println()

	for _, size := range []int{50, 100, 200, 300, 400} {
		rows := size
		cols := size
		iterations := 20

		generateTree := func(frame int) gox.VNode {
			children := make([]gox.VNode, rows)
			for i := 0; i < rows; i++ {
				char := string(rune('A' + (i+frame)%26))
				color := "green"
				if i%2 != 0 {
					color = "yellow"
				}
				children[i] = gox.Element("text", gox.Props{
					"style": map[string]any{"color": color},
				}, gox.Text(strings.Repeat(char, cols)))
			}
			return gox.Element("box", gox.Props{
				"direction": "column",
				"width":     cols,
				"height":    rows,
			}, children...)
		}

		var output strings.Builder
		r := goli.NewAuto(goli.Options{
			Width:  cols,
			Height: rows,
			Output: &output,
		})

		// Warmup
		for i := 0; i < 3; i++ {
			r.Render(generateTree(i))
			output.Reset()
		}

		// Benchmark
		start := time.Now()
		for i := 0; i < iterations; i++ {
			r.Render(generateTree(i))
			output.Reset()
		}
		elapsed := time.Since(start)
		perFrame := float64(elapsed.Nanoseconds()) / float64(iterations) / 1e6
		fps := 1000 / perFrame
		cells := rows * cols

		fmt.Printf("%dx%d (%d cells): %.2fms/frame = %.0f FPS\n", size, size, cells, perFrame, fps)
	}

	fmt.Println()
	fmt.Println("=== Terminal Output Test ===")
	fmt.Println()

	// Test with actual terminal output
	const (
		realRows       = 24
		realCols       = 80
		realIterations = 100
	)

	var totalOutputBytes int

	generateRealTree := func(frame int) gox.VNode {
		children := make([]gox.VNode, realRows)
		for i := 0; i < realRows; i++ {
			// Simulate realistic content with some changes per frame
			changing := (i+frame)%5 == 0
			char := "█"
			color := "green"
			if changing {
				char = string(rune('A' + frame%26))
				color = "cyan"
			} else if i%2 != 0 {
				color = "white"
			}
			children[i] = gox.Element("text", gox.Props{
				"style": map[string]any{"color": color},
			}, gox.Text(strings.Repeat(char, realCols)))
		}
		return gox.Element("box", gox.Props{
			"direction": "column",
			"width":     realCols,
			"height":    realRows,
		}, children...)
	}

	var realOutput strings.Builder
	realRenderer := goli.NewAuto(goli.Options{
		Width:  realCols,
		Height: realRows,
		Output: &realOutput,
	})

	// Warmup
	for i := 0; i < 5; i++ {
		realRenderer.Render(generateRealTree(i))
		realOutput.Reset()
	}
	totalOutputBytes = 0

	// Benchmark with output
	realStart := time.Now()
	for i := 0; i < realIterations; i++ {
		realRenderer.Render(generateRealTree(i))
		totalOutputBytes += realOutput.Len()
		realOutput.Reset()
	}
	realElapsed := time.Since(realStart)
	realPerFrame := float64(realElapsed.Nanoseconds()) / float64(realIterations) / 1e6

	fmt.Printf("Terminal size (%dx%d):\n", realCols, realRows)
	fmt.Printf("  Per frame: %.3fms = %.0f FPS\n", realPerFrame, 1000/realPerFrame)
	fmt.Printf("  Avg output: %d bytes/frame\n", totalOutputBytes/realIterations)
}
