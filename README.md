# goli

A React-like terminal UI framework for Go, using [gox](https://github.com/germtb/gox) for JSX syntax.

## Overview

goli provides:
- **Flexbox layout engine** - Familiar CSS-like layout
- **Reactive primitives** - Fine-grained signals and effects
- **Cell-level diffing** - Minimal ANSI output for efficient rendering
- **Focus management** - Tab navigation, global key handlers
- **Input components** - Text input with scrolling, select dropdowns
- **JSX syntax via gox** - Write components using JSX

## Installation

```bash
go get github.com/germtb/goli
```

You'll also need gox for JSX preprocessing:
```bash
go install github.com/germtb/gox/cmd/gox@latest
```

## Quick Start

### 1. Create a component (app.gox)

```jsx
package main

import (
    "github.com/germtb/goli"
    "github.com/germtb/gox"
)

func Greeting(props gox.Props) gox.VNode {
    name := props["name"].(string)
    return <box direction="column">
        <text style={map[string]any{"color": "green", "bold": true}}>
            Hello, {name}!
        </text>
        <text>Welcome to goli</text>
    </box>
}

func App() gox.VNode {
    return <box width={40} height={10}>
        <Greeting name="World" />
    </box>
}

func main() {
    goli.Run(func() gox.VNode {
        return App()
    }, goli.RunOptions{})
}
```

### 2. Run with gox

```bash
gox run ./
```

## Architecture

```
JSX (.gox) → gox preprocess → Go code → VNode tree → Layout → Buffer → Diff → ANSI
```

### Package Structure

| Package | Description |
|---------|-------------|
| `goli` | Core rendering, layout, app lifecycle, focus, input components |
| `goli/signals` | Reactive primitives: signals, effects, memos, batching |

## Reactive Primitives

goli uses fine-grained reactive primitives:

```go
import "github.com/germtb/goli/signals"

// Create a signal
count, setCount := signals.CreateSignal(0)

// Read value
fmt.Println(count()) // 0

// Update value
setCount(1)

// Update based on previous value
signals.SetWith(setCount, func(prev int) int { return prev + 1 }, count)

// Create derived state
doubled := signals.CreateMemo(func() int {
    return count() * 2
})

// Create side effects
signals.CreateEffect(func() signals.CleanupFunc {
    fmt.Println("Count changed to:", count())
    return nil // cleanup function
})

// Batch updates
signals.Batch(func() {
    setCount(1)
    setCount(2)
}) // Only triggers effects once
```

## Layout Props

Boxes support flexbox-like layout:

```jsx
<box
    direction="row"       // "row" | "column"
    justify="center"      // "start" | "center" | "end" | "space-between"
    align="center"        // "start" | "center" | "end" | "stretch"
    gap={1}               // Space between children
    padding={1}           // Inner spacing (or paddingTop/Right/Bottom/Left)
    width={20}            // Fixed width
    height={5}            // Fixed height
    flex={1}              // Flex grow factor
    border="rounded"      // "single" | "double" | "rounded" | "bold"
    position="absolute"   // "relative" | "absolute"
    x={5} y={3}           // Position for absolute elements
    style={map[string]any{
        "color": "red",
        "background": "blue",
        "bold": true,
    }}
>
    {children}
</box>
```

## Focus & Key Handling

goli provides focus management with Tab/Shift+Tab navigation and global key handlers:

```go
// Register a global key handler for app-wide shortcuts
cleanup := goli.Manager().SetGlobalKeyHandler(func(key string) bool {
    switch key {
    case goli.CtrlQ, "q":
        app.Quit()
        return true
    case goli.F1:
        showHelp()
        return true
    }
    return false // Let other handlers process this key
})
defer cleanup()

// Available key constants
goli.Enter, goli.Escape, goli.Tab, goli.Space
goli.Left, goli.Right, goli.Up, goli.Down
goli.Home, goli.End, goli.PageUp, goli.PageDown
goli.Backspace, goli.Delete, goli.Insert
goli.CtrlA - goli.CtrlZ
goli.ShiftTab, goli.ShiftEnter, goli.ShiftLeft, goli.ShiftRight, ...
goli.AltLeft, goli.AltRight, goli.CtrlLeft, goli.CtrlRight, ...
goli.F1 - goli.F12
```

## Input Components

```go
// Create a text input field
inp := goli.NewInput(goli.InputOptions{
    InitialValue: "",
    MaxLength:    50,
    Placeholder:  "Enter text...",
    Mask:         '*',  // For password fields
})

// Use in JSX - supports horizontal scrolling for long text
<input
    input={inp}
    width={20}
    style={map[string]any{"color": "white"}}
    cursorStyle={map[string]any{"background": "cyan"}}
    placeholderStyle={map[string]any{"dim": true}}
/>

// Create a select dropdown
sel := goli.NewSelect(goli.SelectOptions[string]{
    InitialValue: "option1",
})

// Use in JSX
<select select={sel} pointerWidth={2}>
    <option value="option1">First Option</option>
    <option value="option2">Second Option</option>
</select>
```

## Custom Intrinsic Elements

goli uses a registry pattern for intrinsic elements. You can register your own:

```go
func init() {
    goli.RegisterIntrinsic("mywidget", &goli.IntrinsicHandler{
        Measure: func(node gox.VNode, ctx *goli.LayoutContext) (int, int) {
            return 10, 3 // width, height
        },
        Layout: func(node gox.VNode, availWidth, availHeight int, ctx *goli.LayoutContext) *goli.LayoutBox {
            return &goli.LayoutBox{
                X: ctx.X, Y: ctx.Y,
                Width: 10, Height: 3,
                Node: node,
            }
        },
        Render: func(box *goli.LayoutBox, buf *goli.CellBuffer, clip *goli.ClipRegion) {
            // Draw your widget to the buffer
        },
        RenderLogical: func(box *goli.LayoutBox, buf *goli.LogicalBuffer, clip *goli.ClipRegion) {
            // Draw your widget (for diffing)
        },
    })
}
```

Then use in JSX:
```jsx
<mywidget someProp={value} />
```

## Examples

See the `examples/` directory:

| Example | Description |
|---------|-------------|
| `counter/` | Interactive counter with reactive state |
| `select-demo/` | Multiple selects, inputs, and Tab navigation |
| `vim/` | Vim-like editor with modes and commands |
| `nerdtree/` | File tree browser with expand/collapse |
| `console/` | Console capture with log viewer (Ctrl+L) |

Run any example:
```bash
gox run ./examples/counter
```

## Benchmarks

Comparison against [Ink](https://github.com/vadimdemedes/ink) (React for CLIs) rendering a 100-item file tree:

| Metric | goli (Go) | Ink (React/Bun) | Difference |
|--------|-----------|-----------------|------------|
| **Binary size** | 2.6 MB | 40 MB (node_modules) | ~15x smaller |
| **Startup time** | 0.17 ms | 17 ms | ~100x faster |
| **Memory usage** | 0.28 MB | 37 MB | ~130x less |
| **Idle CPU** | 0.00% | 1.0% | No overhead |
| **Update throughput** | 25,000/sec | 2,800/sec | ~9x faster |

*Tested on Apple M3 Max, Go 1.25, Bun 1.3. See `benchmarks/` for reproduction.*

### Why the difference?

- **No runtime overhead**: Go compiles to native code, no VM/JIT
- **Signals vs React**: Fine-grained reactivity avoids full tree reconciliation
- **Synchronous rendering**: No async scheduling overhead
- **Cell-level diffing**: Only changed characters are written to terminal

Run benchmarks yourself:
```bash
cd benchmarks/comparison-ink && ./run.sh
```

## License

MIT
