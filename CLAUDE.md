# Claude Code Instructions for goli

This file provides instructions for AI assistants working with the goli codebase.

## Critical Rules

1. **ONLY edit `.gox` files** - Never edit generated `_gox.go` files
2. **Use `gox` commands** - Use `gox run`, `gox build`, `gox test` for code with JSX
3. **Delete any `_gox.go` files** - These are generated and should not be in the repo

## Commands

```bash
# Running examples with JSX
gox run ./examples/select-demo

# Building
gox build ./examples/vim

# Testing examples with JSX
gox test ./examples/vim

# Testing library code (pure Go, no JSX)
go test ./...

# Building everything
go build ./...
```

## File Types

| Extension | Purpose | Edit? |
|-----------|---------|-------|
| `.gox` | JSX source files | YES |
| `_gox.go` | Generated Go files | NEVER |
| `.go` | Pure Go files (no JSX) | YES |

## Project Structure

```
goli/
├── *.go              # Core library (pure Go)
├── signals/          # Reactive primitives
├── examples/         # Example applications
│   ├── */app.gox     # JSX source (edit these)
│   └── */main.go     # Pure Go entry points
└── .claude/          # Claude Code configuration
```

## The gox Workflow

1. Write JSX in `.gox` files
2. Run with `gox run ./path/to/example`
3. gox uses an overlay to transform JSX on-the-fly
4. Never commit `_gox.go` files to the repo

## Key Patterns

### Creating Components

```go
// In a .gox file
func MyComponent(props MyProps) gox.VNode {
    return <box direction="column">
        <text>Hello</text>
    </box>
}
```

### Using Signals

```go
import "github.com/germtb/goli/signals"

count, setCount := signals.CreateSignal(0)
signals.SetWith(setCount, func(c int) int { return c + 1 }, count)
```

### Global Key Handlers

```go
cleanup := goli.Manager().SetGlobalKeyHandler(func(key string) bool {
    if key == goli.CtrlQ {
        app.Quit()
        return true
    }
    return false
})
defer cleanup()
```

## Common Mistakes to Avoid

1. Using `go run` instead of `gox run` for JSX files
2. Editing `_gox.go` files instead of `.gox` files
3. Committing generated `_gox.go` files
4. Using `go test` for examples (use `gox test`)
