// Package goli provides the reactive TUI application lifecycle.
package goli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/germtb/goli/signals"
	"github.com/germtb/gox"
)

// App represents a reactive TUI application.
type App struct {
	renderer    *Renderer
	disposeRoot func()
	rerender    func()
	quit        func()
}

// Default frame rate limit (60 FPS = ~16.67ms per frame)
const defaultFrameInterval = 16 * time.Millisecond

// Render creates a reactive TUI application from a gox component.
func Render(appFn func() gox.VNode, opts Options) *App {
	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	r := NewRenderer(Options{
		Width:  opts.Width,
		Height: opts.Height,
		Output: output,
	})

	var disposeRoot func()
	var currentVNode gox.VNode
	var hasVNode bool

	// Simple throttling - just track last render time
	var lastRender time.Time
	throttleDisabled := opts.DisableThrottle

	doRender := func() {
		if !hasVNode {
			return
		}

		// Throttle: skip render if not enough time has passed
		if !throttleDisabled {
			now := time.Now()
			if now.Sub(lastRender) < defaultFrameInterval {
				return // Skip this render, next signal change will try again
			}
			lastRender = now
		}

		if opts.OnRender != nil {
			opts.OnRender()
		}
		r.Render(currentVNode)
	}

	disposeRoot = signals.CreateRoot(func(dispose signals.DisposeFunc) func() {
		signals.CreateEffect(func() signals.CleanupFunc {
			defer func() {
				if r := recover(); r != nil {
					if opts.OnError != nil {
						if err, ok := r.(error); ok {
							opts.OnError(err)
						}
					}
				}
			}()

			currentVNode = appFn()
			hasVNode = true
			doRender()
			return nil
		})

		return dispose
	})

	return &App{
		renderer:    r,
		disposeRoot: disposeRoot,
		rerender:    doRender,
	}
}

// Rerender forces a re-render.
func (a *App) Rerender() {
	a.rerender()
}

// Dispose cleans up the app.
func (a *App) Dispose() {
	if a.disposeRoot != nil {
		a.disposeRoot()
		a.disposeRoot = nil
	}
}

// Renderer returns the underlying renderer.
func (a *App) Renderer() *Renderer {
	return a.renderer
}

// Resize resizes the terminal.
func (a *App) Resize(width, height int) {
	a.renderer.Resize(width, height)
	a.rerender()
}

// Quit signals the application to exit.
func (a *App) Quit() {
	if a.quit != nil {
		a.quit()
	}
}

// RunOptions configures the Run function.
type RunOptions struct {
	Width              int
	Height             int
	Output             io.Writer
	OnMount            func(*App)
	OnUnmount          func()
	OnRender           func()
	OnError            func(error)
	CaptureConsole     bool // Capture console output (default: true). Press Ctrl+L to toggle log viewer.
	MaxConsoleMessages int  // Maximum number of console messages to keep (default: 1000)
}

// Run runs a TUI app with full terminal handling.
func Run(appFn func() gox.VNode, opts RunOptions) {
	// Get terminal size (use actual terminal size if available)
	width := opts.Width
	height := opts.Height

	if width == 0 || height == 0 {
		if w, h, err := GetSize(Stdout()); err == nil {
			if width == 0 {
				width = w
			}
			if height == 0 {
				height = h
			}
		}
	}

	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	// Setup console capture if enabled (default: true)
	captureConsole := opts.CaptureConsole
	maxMessages := opts.MaxConsoleMessages
	if maxMessages <= 0 {
		maxMessages = 1000
	}

	var logCapture *LogCapture
	showLogs, setShowLogs := signals.CreateSignal(false)

	if captureConsole {
		logCapture = NewLogCapture(maxMessages)
		logCapture.Start()
	}

	// Determine output - use original stdout if capturing
	output := opts.Output
	if output == nil {
		if logCapture != nil {
			output = logCapture.OriginalStdout()
		} else {
			output = os.Stdout
		}
	}

	// Wrap app with console panel overlay
	wrappedAppFn := func() gox.VNode {
		appContent := appFn()
		logsVisible := showLogs()

		if !logsVisible || logCapture == nil {
			return appContent
		}

		// Render logs as bottom panel
		messages := logCapture.Messages()
		panelHeight := height / 3
		if panelHeight < 6 {
			panelHeight = 6
		}
		panelY := height - panelHeight
		maxLines := panelHeight - 4 // Account for border, padding, header

		// Get visible messages
		visibleMessages := messages
		if len(visibleMessages) > maxLines {
			visibleMessages = visibleMessages[len(visibleMessages)-maxLines:]
		}

		// Build message nodes using gox
		messageNodes := make([]gox.VNode, 0, len(visibleMessages)+1)

		// Header
		messageNodes = append(messageNodes, gox.Element("text", gox.Props{
			"style": map[string]any{"bold": true, "color": "cyan"},
		}, gox.Text(formatPanelHeader(len(messages)))))

		// Message lines
		for _, msg := range visibleMessages {
			color := "white"
			switch msg.Level {
			case LogLevelError:
				color = "red"
			case LogLevelWarn:
				color = "yellow"
			}
			messageNodes = append(messageNodes, gox.Element("text", gox.Props{
				"style": map[string]any{"color": color},
				"wrap":  true,
			}, gox.Text(" "+FormatMessage(msg))))
		}

		// Console panel
		consolePanel := gox.Element("box", gox.Props{
			"position":   "absolute",
			"x":          0,
			"y":          panelY,
			"width":      width,
			"height":     panelHeight,
			"border":     "single",
			"overflow":   "hidden",
			"style":      map[string]any{"background": "black", "color": "white"},
			"paddingTop": 0,
		}, messageNodes...)

		// Wrap with container
		return gox.Element("box", gox.Props{"width": width, "height": height}, appContent, consolePanel)
	}

	// Put terminal in raw mode for single key input
	var oldState *State
	if IsTerminal(Stdin()) {
		var err error
		oldState, err = MakeRaw(Stdin())
		if err != nil {
			// Fallback: continue without raw mode (for testing)
			oldState = nil
		}
	}

	// Restore terminal on exit
	defer func() {
		if oldState != nil {
			Restore(Stdin(), oldState)
		}
	}()

	app := Render(wrappedAppFn, Options{
		Width:    width,
		Height:   height,
		Output:   output,
		OnRender: opts.OnRender,
		OnError:  opts.OnError,
	})

	// Hide cursor
	io.WriteString(output, HideCursor())
	defer io.WriteString(output, ShowCursor())

	// Clear screen on exit
	defer io.WriteString(output, ClearScreen())

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)

	// Done channel
	done := make(chan struct{})
	var cleanedUp bool

	// Cleanup function
	cleanup := func() {
		if cleanedUp {
			return
		}
		cleanedUp = true
		if logCapture != nil {
			logCapture.Stop()
		}
		app.Dispose()
		if opts.OnUnmount != nil {
			opts.OnUnmount()
		}
		close(done)
	}

	// Set quit function on app
	app.quit = cleanup

	// Setup console shortcuts as global key handler
	// (only triggers if no focusable consumes the key)
	var cleanupGlobalHandler func()
	if logCapture != nil {
		cleanupGlobalHandler = Manager().SetGlobalKeyHandler(func(key string) bool {
			if key == CtrlL {
				setShowLogs(!showLogs())
				return true
			}
			if key == CtrlK && showLogs() {
				logCapture.Clear()
				return true
			}
			return false
		})
	}

	// Handle signals
	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGWINCH:
				// Terminal resized - get new size and resize app
				if w, h, err := GetSize(Stdout()); err == nil {
					width = w
					height = h
					app.Resize(width, height)
				}
			case syscall.SIGINT, syscall.SIGTERM:
				if cleanupGlobalHandler != nil {
					cleanupGlobalHandler()
				}
				cleanup()
				return
			}
		}
	}()

	// Start input reader
	go func() {
		buf := make([]byte, 64)
		for {
			select {
			case <-done:
				return
			default:
				n, err := os.Stdin.Read(buf)
				if err != nil {
					// Any error on stdin (EOF, closed, etc.) - stop reading
					// The app continues running for programmatic control
					return
				}
				key := string(buf[:n])

				// Ctrl+C exits
				if key == "\x03" {
					if cleanupGlobalHandler != nil {
						cleanupGlobalHandler()
					}
					cleanup()
					return
				}

				// Route to focus manager (handles Tab, routes to focused element, then global handler)
				HandleKey(key)
			}
		}
	}()

	if opts.OnMount != nil {
		opts.OnMount(app)
	}

	// Wait for exit
	<-done
}

// formatPanelHeader formats the console panel header.
func formatPanelHeader(count int) string {
	return fmt.Sprintf(" Console (%d) - Ctrl+L close, Ctrl+K clear", count)
}
