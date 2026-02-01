// Goli version of the benchmark UI - a file tree with 100 items
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/goli/signals"
	"github.com/germtb/gox"
)

func FileTree(selectedIndex int) gox.VNode {
	var items []gox.VNode
	for i := 0; i < 100; i++ {
		style := map[string]any{"color": "white"}
		prefix := "  "
		if i == selectedIndex {
			style["bold"] = true
			style["color"] = "cyan"
			prefix = "> "
		}
		items = append(items, gox.Element("text", gox.Props{"style": style},
			gox.Text(fmt.Sprintf("%s├── file-%03d.go", prefix, i))))
	}

	return gox.Element("box", gox.Props{
		"direction": "column",
		"border":    "rounded",
		"padding":   1,
	},
		gox.Element("text", gox.Props{"style": map[string]any{"bold": true, "color": "green"}},
			gox.Text("File Browser (goli)")),
		gox.Element("box", gox.Props{"height": 1}),
		gox.Fragment(items...),
	)
}

func main() {
	mode := "benchmark"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "startup":
		measureStartup()
	case "memory":
		measureMemory()
	case "idle":
		measureIdleCPU()
	case "updates":
		measureUpdates()
	case "benchmark":
		runAllBenchmarks()
	default:
		fmt.Println("Usage: goli-bench [startup|memory|idle|updates|benchmark]")
	}
}

func runAllBenchmarks() {
	fmt.Println("=== goli Benchmark ===")
	fmt.Printf("Go version: %s\n\n", runtime.Version())

	measureStartup()
	measureMemory()
	measureIdleCPU()
	measureUpdates()
}

func measureStartup() {
	start := time.Now()

	var output strings.Builder
	selected, _ := signals.CreateSignal(0)

	app := goli.Render(func() gox.VNode {
		return FileTree(selected())
	}, goli.Options{
		Width:  60,
		Height: 40,
		Output: &output,
	})

	elapsed := time.Since(start)
	app.Dispose()

	fmt.Printf("Startup time: %v\n", elapsed)
}

func measureMemory() {
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	var output strings.Builder
	selected, _ := signals.CreateSignal(0)

	app := goli.Render(func() gox.VNode {
		return FileTree(selected())
	}, goli.Options{
		Width:  60,
		Height: 40,
		Output: &output,
	})

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	app.Dispose()

	fmt.Printf("Memory used: %.2f MB\n", float64(m2.Alloc-m1.Alloc)/(1024*1024))
	fmt.Printf("Total alloc: %.2f MB\n", float64(m2.TotalAlloc-m1.TotalAlloc)/(1024*1024))
}

func measureIdleCPU() {
	var output strings.Builder
	selected, _ := signals.CreateSignal(0)

	app := goli.Render(func() gox.VNode {
		return FileTree(selected())
	}, goli.Options{
		Width:  60,
		Height: 40,
		Output: &output,
	})

	// Measure CPU over 2 seconds idle
	var rusageStart syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	startTime := time.Now()

	time.Sleep(2 * time.Second)

	var rusageEnd syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageEnd)
	elapsed := time.Since(startTime)

	app.Dispose()

	userStart := time.Duration(rusageStart.Utime.Nano())
	userEnd := time.Duration(rusageEnd.Utime.Nano())
	sysStart := time.Duration(rusageStart.Stime.Nano())
	sysEnd := time.Duration(rusageEnd.Stime.Nano())

	cpuUsed := (userEnd - userStart) + (sysEnd - sysStart)
	cpuPercent := (float64(cpuUsed) / float64(elapsed)) * 100

	fmt.Printf("Idle CPU: %.2f%%\n", cpuPercent)
}

func measureUpdates() {
	var output strings.Builder
	selected, setSelected := signals.CreateSignal(0)

	app := goli.Render(func() gox.VNode {
		return FileTree(selected())
	}, goli.Options{
		Width:           60,
		Height:          40,
		Output:          &output,
		DisableThrottle: true,
	})

	// Measure 1000 updates
	start := time.Now()
	for i := 0; i < 1000; i++ {
		setSelected(i % 100)
	}
	elapsed := time.Since(start)

	app.Dispose()

	fmt.Printf("1000 updates: %v (%.0f updates/sec)\n", elapsed, 1000/elapsed.Seconds())
}
