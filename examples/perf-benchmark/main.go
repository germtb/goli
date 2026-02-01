// Performance benchmark - measures actual render pipeline timing
//
// Run with: go run ./examples/perf-benchmark/
package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

// Configuration
const (
	WarmupIterations    = 10
	BenchmarkIterations = 100
	Rows                = 50
	Cols                = 80
)

// BenchResult holds benchmark timing results
type BenchResult struct {
	Avg float64
	Min float64
	Max float64
}

// Generate test VNode tree
func generateTree(frame int) gox.VNode {
	rows := make([]gox.VNode, Rows)

	for i := 0; i < Rows; i++ {
		// Add some variation based on frame and row
		char := string(rune('A' + (i+frame)%26))
		content := fmt.Sprintf("Row %03d: %s", i, strings.Repeat(char, Cols-12))

		var bg string
		if i == Rows/2 {
			bg = "blue"
		}
		color := "green"
		if i%2 != 0 {
			color = "yellow"
		}

		rows[i] = gox.Element("text", gox.Props{
			"style": map[string]any{
				"color":      color,
				"background": bg,
			},
		}, gox.Text(content))
	}

	return gox.Element("box", gox.Props{
		"direction": "column",
		"width":     Cols,
		"height":    Rows,
	}, rows...)
}

// Benchmark helper
func benchmark(name string, iterations int, fn func()) BenchResult {
	times := make([]float64, 0, iterations)

	// Warmup
	for i := 0; i < WarmupIterations; i++ {
		fn()
	}

	// Benchmark
	for i := 0; i < iterations; i++ {
		start := time.Now()
		fn()
		times = append(times, float64(time.Since(start).Nanoseconds())/1e6)
	}

	var sum float64
	min := times[0]
	max := times[0]
	for _, t := range times {
		sum += t
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}

	return BenchResult{
		Avg: sum / float64(len(times)),
		Min: min,
		Max: max,
	}
}

func formatResult(name string, result BenchResult) string {
	return fmt.Sprintf("%-35s avg: %.3fms  min: %.3fms  max: %.3fms", name, result.Avg, result.Min, result.Max)
}

func main() {
	fmt.Println(strings.Repeat("=", 85))
	fmt.Println("goli Performance Benchmark")
	fmt.Printf("Config: %d rows × %d cols, %d iterations\n", Rows, Cols, BenchmarkIterations)
	fmt.Println(strings.Repeat("=", 85))
	fmt.Println()

	// 1. VNode tree creation
	treeCreation := benchmark("VNode tree creation", BenchmarkIterations, func() {
		generateTree(0)
	})
	fmt.Println(formatResult("1. VNode tree creation", treeCreation))

	// 2. Layout computation
	tree := generateTree(0)
	layoutComputation := benchmark("Layout computation", BenchmarkIterations, func() {
		goli.ComputeLayout(tree, goli.LayoutContext{X: 0, Y: 0, Width: Cols, Height: Rows})
	})
	fmt.Println(formatResult("2. Layout computation", layoutComputation))

	// 3. Render to buffer
	layoutResult := goli.ComputeLayout(tree, goli.LayoutContext{X: 0, Y: 0, Width: Cols, Height: Rows})
	renderToBufferResult := benchmark("Render to buffer", BenchmarkIterations, func() {
		buf := goli.NewCellBuffer(Cols, Rows)
		goli.RenderToBuffer(layoutResult, buf, nil)
	})
	fmt.Println(formatResult("3. Render to buffer", renderToBufferResult))

	// 4. Buffer diffing (no changes)
	buf1 := goli.NewCellBuffer(Cols, Rows)
	goli.RenderToBuffer(layoutResult, buf1, nil)
	buf2 := goli.NewCellBuffer(Cols, Rows)
	goli.RenderToBuffer(layoutResult, buf2, nil) // Same content

	diffNoChanges := benchmark("Diff (no changes)", BenchmarkIterations, func() {
		goli.DiffBuffers(buf1, buf2)
	})
	fmt.Println(formatResult("4. Diff (no changes)", diffNoChanges))

	// 5. Buffer diffing (all changes)
	buf3 := goli.NewCellBuffer(Cols, Rows)
	tree2 := generateTree(1) // Different frame
	layout2 := goli.ComputeLayout(tree2, goli.LayoutContext{X: 0, Y: 0, Width: Cols, Height: Rows})
	goli.RenderToBuffer(layout2, buf3, nil)

	diffAllChanges := benchmark("Diff (all changes)", BenchmarkIterations, func() {
		goli.DiffBuffers(buf1, buf3)
	})
	fmt.Println(formatResult("5. Diff (all changes)", diffAllChanges))

	// 6. Buffer diffing (partial changes - 10%)
	buf4 := goli.NewCellBuffer(Cols, Rows)
	goli.RenderToBuffer(layoutResult, buf4, nil) // Start with same content
	for i := 0; i < Rows*Cols/10; i++ {
		x := i % Cols
		y := (i / Cols) % Rows
		buf4.WriteString(x, y, "X", goli.Style{})
	}

	diffPartialChanges := benchmark("Diff (10% changes)", BenchmarkIterations, func() {
		goli.DiffBuffers(buf1, buf4)
	})
	fmt.Println(formatResult("6. Diff (10% changes)", diffPartialChanges))

	// 7. Find runs
	changes := goli.DiffBuffers(buf1, buf3)
	findRunsResult := benchmark("Find runs", BenchmarkIterations, func() {
		goli.FindRuns(changes)
	})
	fmt.Println(formatResult("7. Find runs", findRunsResult))

	// 8. ANSI generation
	runs := goli.FindRuns(changes)
	var ansiOutput string
	ansiGeneration := benchmark("ANSI generation", BenchmarkIterations, func() {
		ansiOutput = goli.RunsToAnsi(runs)
	})
	fmt.Println(formatResult("8. ANSI generation", ansiGeneration))

	// 9. Full render pipeline
	var output strings.Builder
	r := goli.NewRenderer(goli.Options{
		Width:  Cols,
		Height: Rows,
		Output: &output,
	})

	frameNum := 0
	fullPipeline := benchmark("Full render pipeline", BenchmarkIterations, func() {
		tree := generateTree(frameNum)
		r.Render(tree)
		frameNum++
		output.Reset()
	})
	fmt.Println(formatResult("9. Full render pipeline", fullPipeline))

	// 10. Full pipeline with alternating frames (simulates real usage)
	frameNum = 0
	fullPipelineAlternating := benchmark("Full pipeline (alternating)", BenchmarkIterations, func() {
		tree := generateTree(frameNum % 2)
		r.Render(tree)
		frameNum++
		output.Reset()
	})
	fmt.Println(formatResult("10. Full pipeline (alternating)", fullPipelineAlternating))

	fmt.Println()
	fmt.Println(strings.Repeat("=", 85))

	// Summary
	totalPipeline := treeCreation.Avg + layoutComputation.Avg + renderToBufferResult.Avg +
		diffPartialChanges.Avg + findRunsResult.Avg + ansiGeneration.Avg
	theoreticalFps := 1000 / totalPipeline

	fmt.Println("Summary:")
	fmt.Printf("  Total pipeline (10%% changes): %.3fms = %.0f FPS\n", totalPipeline, theoreticalFps)
	fmt.Printf("  Full render:                  %.3fms = %.0f FPS\n", fullPipeline.Avg, 1000/fullPipeline.Avg)
	fmt.Println()

	// Identify bottlenecks
	type step struct {
		name string
		time float64
	}
	steps := []step{
		{"VNode creation", treeCreation.Avg},
		{"Layout", layoutComputation.Avg},
		{"Render to buffer", renderToBufferResult.Avg},
		{"Diff", diffPartialChanges.Avg},
		{"Find runs", findRunsResult.Avg},
		{"ANSI generation", ansiGeneration.Avg},
	}

	sort.Slice(steps, func(i, j int) bool {
		return steps[i].time > steps[j].time
	})

	fmt.Println("All stages (sorted by time):")
	for _, s := range steps {
		fmt.Printf("  %-25s %.3fms\n", s.name, s.time)
	}

	fmt.Println()
	fmt.Printf("ANSI output size: %d bytes\n", len(ansiOutput))
	fmt.Println(strings.Repeat("=", 85))
}
