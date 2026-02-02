// Package main provides CPU usage benchmarking for goli applications.
//
// Run with: go run ./examples/perf-cpu
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

// Simulates a complex UI similar to nerdtree
func ComplexUI() gox.VNode {
	// Create a tree-like structure
	var items []gox.VNode
	for i := 0; i < 20; i++ {
		style := map[string]any{"color": "white"}
		if i%3 == 0 {
			style["bold"] = true
		}
		items = append(items, gox.Element("text", gox.Props{"style": style},
			gox.Text(fmt.Sprintf("├── item-%d.go", i))))
	}

	return gox.Element("box", gox.Props{
		"direction": "column",
		"border":    "rounded",
		"padding":   1,
	},
		gox.Element("text", gox.Props{"style": map[string]any{"bold": true, "color": "cyan"}},
			gox.Text("File Browser")),
		gox.Element("box", gox.Props{"height": 1}),
		gox.Fragment(items...),
	)
}

func main() {
	fmt.Println("=== goli CPU Usage Benchmark ===")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Println()

	// Test 1: Measure baseline idle CPU
	fmt.Println("Test 1: Measuring baseline idle CPU (3 seconds)...")
	idleCPU := measureCPU(3*time.Second, func(done <-chan struct{}) {
		<-done
	})
	fmt.Printf("  Baseline idle CPU: %.2f%%\n", idleCPU)

	// Test 2: Measure CPU with goli app running but idle (no input, no updates)
	fmt.Println("\nTest 2: Measuring goli app idle CPU (3 seconds)...")
	var output strings.Builder
	goliIdleCPU := measureCPU(3*time.Second, func(done <-chan struct{}) {
		app := goli.Render(ComplexUI, goli.Options{
			Width:  80,
			Height: 30,
			Output: &output,
		})
		<-done
		app.Dispose()
	})
	fmt.Printf("  goli idle CPU: %.2f%%\n", goliIdleCPU)
	fmt.Printf("  Overhead vs baseline: %.2f%%\n", goliIdleCPU-idleCPU)

	// Test 3: Measure CPU during rapid signal updates (stress test)
	fmt.Println("\nTest 3: Measuring CPU during rapid updates (1000/sec for 2 sec)...")
	output.Reset()
	counter, setCounter := goli.CreateSignal(0)
	updateCPU := measureCPU(2*time.Second, func(done <-chan struct{}) {
		app := goli.Render(func() gox.VNode {
			c := counter()
			return gox.Element("box", gox.Props{"direction": "column"},
				gox.Element("text", nil, gox.Text(fmt.Sprintf("Counter: %d", c))),
				ComplexUI(),
			)
		}, goli.Options{
			Width:  80,
			Height: 30,
			Output: &output,
		})

		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				app.Dispose()
				return
			case <-ticker.C:
				i++
				setCounter(i)
			}
		}
	})
	fmt.Printf("  CPU during 1000 updates/sec: %.2f%%\n", updateCPU)

	// Test 4: Measure CPU with realistic 60fps updates
	fmt.Println("\nTest 4: Measuring CPU at 60 updates/sec (2 sec)...")
	output.Reset()
	counter2, setCounter2 := goli.CreateSignal(0)
	throttledCPU := measureCPU(2*time.Second, func(done <-chan struct{}) {
		app := goli.Render(func() gox.VNode {
			c := counter2()
			return gox.Element("box", gox.Props{"direction": "column"},
				gox.Element("text", nil, gox.Text(fmt.Sprintf("Counter: %d", c))),
				ComplexUI(),
			)
		}, goli.Options{
			Width:  80,
			Height: 30,
			Output: &output,
		})

		ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				app.Dispose()
				return
			case <-ticker.C:
				i++
				setCounter2(i)
			}
		}
	})
	fmt.Printf("  CPU at 60fps: %.2f%%\n", throttledCPU)

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Baseline idle:         %.2f%%\n", idleCPU)
	fmt.Printf("goli app idle:         %.2f%% (overhead: +%.2f%%)\n", goliIdleCPU, goliIdleCPU-idleCPU)
	fmt.Printf("1000 updates/sec:      %.2f%%\n", updateCPU)
	fmt.Printf("60 updates/sec (60fps): %.2f%%\n", throttledCPU)

	// Check for issues
	idleOverhead := goliIdleCPU - idleCPU
	if idleOverhead > 5.0 {
		fmt.Println("\n⚠️  WARNING: Idle CPU overhead > 5% - there may be a busy loop!")
		os.Exit(1)
	} else if idleOverhead > 1.0 {
		fmt.Println("\n⚠️  Note: Idle CPU overhead > 1% - worth investigating")
	} else {
		fmt.Println("\n✓ CPU usage looks good!")
	}
}

// measureCPU runs a function for the given duration and returns CPU percentage
func measureCPU(duration time.Duration, fn func(done <-chan struct{})) float64 {
	runtime.GC() // Clean slate

	done := make(chan struct{})
	finished := make(chan struct{})

	// Get starting CPU times
	var rusageStart syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	startTime := time.Now()

	go func() {
		fn(done)
		close(finished)
	}()

	// Wait for duration
	time.Sleep(duration)
	close(done)

	// Wait for function to finish (with timeout)
	select {
	case <-finished:
	case <-time.After(time.Second):
	}

	// Get ending CPU times
	var rusageEnd syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageEnd)
	elapsed := time.Since(startTime)

	// Calculate CPU time used
	userStart := time.Duration(rusageStart.Utime.Nano())
	userEnd := time.Duration(rusageEnd.Utime.Nano())
	sysStart := time.Duration(rusageStart.Stime.Nano())
	sysEnd := time.Duration(rusageEnd.Stime.Nano())

	cpuUsed := (userEnd - userStart) + (sysEnd - sysStart)
	cpuPercent := (float64(cpuUsed) / float64(elapsed)) * 100

	return cpuPercent
}
