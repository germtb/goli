# How to Use Goli

## CRITICAL RULES

1. **ONLY edit `.gox` files** - Never edit generated `_gox.go` files
2. **ONLY use `gox run`** - Never use `go run` or `go build` for examples with JSX
3. **Delete any `_gox.go` files** - These are generated and should not be in the repo

## Running Examples

```bash
# CORRECT - use gox run
gox run ./examples/select-demo

# WRONG - never do this
go build ./...
go run ./examples/select-demo
```

## Editing Components

```bash
# CORRECT - edit the .gox source file
vim examples/select-demo/main.gox

# WRONG - never edit generated files
vim examples/select-demo/main_gox.go  # NO!
```

## File Types

| Extension | Purpose | Edit? |
|-----------|---------|-------|
| `.gox` | JSX source files | YES |
| `_gox.go` | Generated Go files | NEVER |
| `.go` | Pure Go files (no JSX) | YES |

## The gox Workflow

1. Write JSX in `.gox` files
2. Run with `gox run ./path/to/example`
3. gox generates `_gox.go` on the fly and runs it
4. Never commit `_gox.go` files to the repo

## Testing the Library

```bash
# For library tests (pure Go, no JSX)
go test ./...

# For running examples with JSX
gox run ./examples/select-demo
```
