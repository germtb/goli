// Performance stress test
// Tests rendering performance with rapidly changing content and scrolling.
//
// Run with: go run ./examples/perf-stress/
package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/germtb/goli"
	"github.com/germtb/gox"
)

// Configuration
const (
	TotalRows    = 100 // Total rows of content
	Cols         = 80  // Columns per row
	VisibleRows  = 20  // Visible viewport
	InitialNoise = 5   // % of cells that change each frame
	InitialFPS   = 60
)

var fpsPresets = []int{30, 60, 120, 240, 500, 0} // 0 = uncapped

// State
var (
	scrollOffset, setScrollOffset = goli.CreateSignal(0)
	data, setData                 = goli.CreateSignal(generateInitialData())
	frameCount, setFrameCount     = goli.CreateSignal(0)
	measuredFps, setMeasuredFps   = goli.CreateSignal(0)
	renderTime, setRenderTime     = goli.CreateSignal(0.0)
	paused, setPaused             = goli.CreateSignal(false)
	targetFps, setTargetFps       = goli.CreateSignal(InitialFPS)
	noisePercent, setNoisePercent = goli.CreateSignal(InitialNoise)
)

// Random character
func randomChar() rune {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*"
	return rune(chars[rand.Intn(len(chars))])
}

// Generate a row of content (as runes for proper indexing)
func generateRow(rowIndex int) []rune {
	prefix := fmt.Sprintf("%04d | ", rowIndex)
	row := make([]rune, Cols)
	copy(row, []rune(prefix))
	for i := len(prefix); i < Cols; i++ {
		row[i] = '█'
	}
	return row
}

// Generate initial data
func generateInitialData() [][]rune {
	rows := make([][]rune, TotalRows)
	for i := 0; i < TotalRows; i++ {
		rows[i] = generateRow(i)
	}
	return rows
}

// Add noise to data (simulate rapidly changing content)
func addNoise() {
	noise := noisePercent()
	goli.SetWith(setData, func(rows [][]rune) [][]rune {
		// Clone rows
		newRows := make([][]rune, len(rows))
		for i, row := range rows {
			newRows[i] = make([]rune, len(row))
			copy(newRows[i], row)
		}

		cellsToChange := (TotalRows * Cols * noise) / 100

		for i := 0; i < cellsToChange; i++ {
			rowIdx := rand.Intn(TotalRows)
			colIdx := rand.Intn(Cols-7) + 7 // Skip line number prefix "0000 | "
			if colIdx < len(newRows[rowIdx]) {
				newRows[rowIdx][colIdx] = randomChar()
			}
		}

		return newRows
	}, data)
}

// Auto-scroll
func autoScroll() {
	goli.SetWith(setScrollOffset, func(offset int) int {
		newOffset := offset + 1
		if newOffset > TotalRows-VisibleRows {
			return 0
		}
		return newOffset
	}, scrollOffset)
}

func cycleFps(direction int) {
	current := targetFps()
	currentIndex := -1
	for i, preset := range fpsPresets {
		if preset == current {
			currentIndex = i
			break
		}
	}

	var newIndex int
	if currentIndex == -1 {
		if direction == 1 {
			newIndex = 0
		} else {
			newIndex = len(fpsPresets) - 1
		}
	} else {
		newIndex = (currentIndex + direction + len(fpsPresets)) % len(fpsPresets)
	}

	setTargetFps(fpsPresets[newIndex])
}

// App component
var App gox.Component = func(props gox.Props) gox.VNode {
	offset := scrollOffset()
	rows := data()
	currentFps := measuredFps()
	currentRenderTime := renderTime()
	isPaused := paused()
	target := targetFps()
	noise := noisePercent()
	frame := frameCount()

	// Get visible rows
	endIdx := offset + VisibleRows
	if endIdx > len(rows) {
		endIdx = len(rows)
	}
	visibleRows := rows[offset:endIdx]

	// Target FPS display
	targetDisplay := fmt.Sprintf("%d", target)
	if target == 0 {
		targetDisplay = "∞"
	}

	pauseStatus := "RUNNING"
	if isPaused {
		pauseStatus = "PAUSED"
	}

	children := []gox.VNode{
		// Header with stats
		gox.Element("box", gox.Props{"direction": "row"},
			gox.Element("text", gox.Props{
				"style": map[string]any{"color": "cyan", "bold": true},
			}, gox.Text(fmt.Sprintf("FPS: %d/%s | Render: %.2fms | Noise: %d%% | %s",
				currentFps, targetDisplay, currentRenderTime, noise, pauseStatus))),
		),

		// Controls
		gox.Element("text", gox.Props{
			"style": map[string]any{"color": "white", "dim": true},
		}, gox.Text("p=pause  +/-=fps  </>=noise  [/]=scroll  q=quit")),

		// Separator
		gox.Element("text", gox.Props{
			"style": map[string]any{"color": "white"},
		}, gox.Text(strings.Repeat("─", Cols))),
	}

	// Add visible rows
	for i, row := range visibleRows {
		globalRow := offset + i
		color := "green"
		if globalRow%2 != 0 {
			color = "yellow"
		}
		var bg string
		if i == VisibleRows/2 {
			bg = "blue"
		}

		children = append(children, gox.Element("text", gox.Props{
			"style": map[string]any{"color": color, "background": bg},
		}, gox.Text(string(row))))
	}

	// Separator
	children = append(children, gox.Element("text", gox.Props{
		"style": map[string]any{"color": "white"},
	}, gox.Text(strings.Repeat("─", Cols))))

	// Footer
	children = append(children, gox.Element("text", gox.Props{
		"style": map[string]any{"color": "white", "dim": true},
	}, gox.Text(fmt.Sprintf("Rows: %d | Visible: %d | Scroll: %d/%d | Frame: %d",
		TotalRows, VisibleRows, offset, TotalRows-VisibleRows, frame))))

	return gox.Element("box", gox.Props{
		"direction": "column",
	}, children...)
}

func main() {
	output := os.Stdout

	// Put terminal in raw mode
	var oldState *goli.State
	if goli.IsTerminal(goli.Stdin()) {
		var err error
		oldState, err = goli.MakeRaw(goli.Stdin())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set raw mode: %v\n", err)
			return
		}
	}
	defer func() {
		if oldState != nil {
			goli.Restore(goli.Stdin(), oldState)
		}
	}()

	// Get terminal size
	width, height := Cols, VisibleRows+5
	if w, h, err := goli.GetSize(goli.Stdout()); err == nil {
		width, height = w, h
	}

	// Create app
	application := goli.Render(func() gox.VNode {
		return gox.Element(App, nil)
	}, goli.Options{
		Width:  width,
		Height: height,
		Output: output,
	})
	defer application.Dispose()

	// Hide cursor
	io.WriteString(output, goli.HideCursor())
	defer io.WriteString(output, goli.ShowCursor())
	defer io.WriteString(output, goli.ClearScreen())

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Input channel - read keys in a goroutine, send to channel
	keyCh := make(chan string, 10)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			keyCh <- string(buf[:n])
		}
	}()

	// FPS tracking
	lastFpsTime := time.Now()
	framesSinceFps := 0

	// Main event loop - everything runs synchronously here
	running := true
	for running {
		// Calculate frame timing
		target := targetFps()
		var frameDelay time.Duration
		if target == 0 {
			frameDelay = time.Millisecond // uncapped but don't spin
		} else {
			frameDelay = time.Second / time.Duration(target)
		}

		// Wait for next event: timer, key, or signal
		select {
		case <-sigCh:
			running = false

		case key := <-keyCh:
			// Handle Ctrl+C
			if key == "\x03" {
				running = false
				continue
			}

			// Handle keys
			switch key {
			case "q":
				running = false
			case "p":
				goli.SetWith(setPaused, func(p bool) bool { return !p }, paused)
				application.Rerender()
			case goli.Up:
				goli.SetWith(setScrollOffset, func(o int) int {
					if o > 0 {
						return o - 1
					}
					return o
				}, scrollOffset)
				application.Rerender()
			case goli.Down:
				goli.SetWith(setScrollOffset, func(o int) int {
					if o < TotalRows-VisibleRows {
						return o + 1
					}
					return o
				}, scrollOffset)
				application.Rerender()
			case "[":
				goli.SetWith(setScrollOffset, func(o int) int {
					newO := o - 10
					if newO < 0 {
						return 0
					}
					return newO
				}, scrollOffset)
				application.Rerender()
			case "]":
				goli.SetWith(setScrollOffset, func(o int) int {
					newO := o + 10
					if newO > TotalRows-VisibleRows {
						return TotalRows - VisibleRows
					}
					return newO
				}, scrollOffset)
				application.Rerender()
			case "+", "=":
				cycleFps(1)
				application.Rerender()
			case "-", "_":
				cycleFps(-1)
				application.Rerender()
			case ">", ".":
				goli.SetWith(setNoisePercent, func(n int) int {
					if n < 100 {
						return n + 5
					}
					return n
				}, noisePercent)
				application.Rerender()
			case "<", ",":
				goli.SetWith(setNoisePercent, func(n int) int {
					if n > 0 {
						return n - 5
					}
					return n
				}, noisePercent)
				application.Rerender()
			}

		case <-time.After(frameDelay):
			// Animation frame
			if !paused() {
				frameStart := time.Now()

				// Update state
				addNoise()
				autoScroll()
				goli.SetWith(setFrameCount, func(c int) int { return c + 1 }, frameCount)

				// Track render time
				setRenderTime(float64(time.Since(frameStart).Nanoseconds()) / 1e6)

				// Calculate FPS
				framesSinceFps++
				now := time.Now()
				if now.Sub(lastFpsTime) >= time.Second {
					setMeasuredFps(framesSinceFps)
					framesSinceFps = 0
					lastFpsTime = now
				}

				// Render
				application.Rerender()
			}
		}
	}
}
