// Pipeline benchmark - compare sequential vs pipelined rendering
//
// Run with: go run ./examples/perf-pipeline/
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

const (
	Rows         = 100
	Cols         = 100
	Frames       = 200
	WarmupFrames = 20
)

func generateTree(frame int) gox.VNode {
	children := make([]gox.VNode, Rows)
	for i := 0; i < Rows; i++ {
		char := string(rune('A' + (i+frame)%26))
		color := "green"
		if i%2 != 0 {
			color = "yellow"
		}
		children[i] = gox.Element("text", gox.Props{
			"style": map[string]any{"color": color},
		}, gox.Text(strings.Repeat(char, Cols)))
	}
	return gox.Element("box", gox.Props{
		"direction": "column",
		"width":     Cols,
		"height":    Rows,
	}, children...)
}

func main() {
	fmt.Println("Pipeline vs Sequential Renderer Benchmark")
	fmt.Printf("Config: %dx%d grid, %d frames\n", Rows, Cols, Frames)
	fmt.Println(strings.Repeat("=", 50))

	// Sequential renderer benchmark
	var seqOutput strings.Builder
	seqRenderer := goli.NewRenderer(goli.Options{
		Width:  Cols,
		Height: Rows,
		Output: &seqOutput,
	})

	// Warmup
	for i := 0; i < WarmupFrames; i++ {
		seqRenderer.Render(generateTree(i))
		seqOutput.Reset()
	}

	seqStart := time.Now()
	for i := 0; i < Frames; i++ {
		seqRenderer.Render(generateTree(i))
		seqOutput.Reset()
	}
	seqDuration := time.Since(seqStart)
	seqFps := float64(Frames) / seqDuration.Seconds()

	fmt.Printf("\nSequential Renderer:\n")
	fmt.Printf("  Total time: %v\n", seqDuration)
	fmt.Printf("  Per frame:  %.3fms\n", float64(seqDuration.Nanoseconds())/float64(Frames)/1e6)
	fmt.Printf("  Throughput: %.0f FPS\n", seqFps)

	// Pipeline renderer benchmark
	var pipeOutput strings.Builder
	pipeRenderer := goli.NewPipeline(goli.Options{
		Width:  Cols,
		Height: Rows,
		Output: &pipeOutput,
	})

	// Warmup
	for i := 0; i < WarmupFrames; i++ {
		pipeRenderer.RenderBlocking(generateTree(i))
	}
	pipeOutput.Reset()

	pipeStart := time.Now()
	for i := 0; i < Frames; i++ {
		pipeRenderer.RenderBlocking(generateTree(i))
	}
	// Wait for pipeline to drain
	time.Sleep(10 * time.Millisecond)
	pipeDuration := time.Since(pipeStart)
	pipeFps := float64(Frames) / pipeDuration.Seconds()

	pipeRenderer.Stop()

	fmt.Printf("\nPipeline Renderer:\n")
	fmt.Printf("  Total time: %v\n", pipeDuration)
	fmt.Printf("  Per frame:  %.3fms\n", float64(pipeDuration.Nanoseconds())/float64(Frames)/1e6)
	fmt.Printf("  Throughput: %.0f FPS\n", pipeFps)

	fmt.Println()
	fmt.Printf("Speedup: %.2fx\n", pipeFps/seqFps)
	fmt.Println(strings.Repeat("=", 50))
}
