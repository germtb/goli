# Contributing to goli

Thank you for your interest in contributing to goli!

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/germtb/goli.git
cd goli
```

2. Install gox (JSX preprocessor):
```bash
go install github.com/germtb/gox/cmd/gox@latest
```

3. Verify setup:
```bash
go build ./...
go test ./...
gox run ./examples/counter
```

## Project Structure

- `*.go` - Core library files (pure Go)
- `signals/` - Reactive primitives package
- `examples/` - Example applications
- `.claude/` - AI assistant configuration

## Working with JSX

goli uses [gox](https://github.com/germtb/gox) for JSX syntax. Files with JSX use the `.gox` extension.

**Important:**
- Edit `.gox` files, never `_gox.go` files
- Use `gox run/build/test` for JSX code
- Use `go run/build/test` for pure Go code

## Running Tests

```bash
# Library tests (pure Go)
go test ./...

# Example tests (require gox)
gox test ./examples/vim
```

## Code Style

- Run `gofmt` before committing
- Follow standard Go conventions
- Keep functions focused and small
- Add tests for new functionality

## Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Run formatter: `gofmt -w .`
6. Submit a pull request

## Adding Examples

1. Create a new directory in `examples/`
2. Add `app.gox` for the main component
3. Test with `gox run ./examples/yourexample`
4. Update README.md examples table

## Reporting Issues

- Check existing issues first
- Include Go version and OS
- Provide minimal reproduction steps
- Include error messages and stack traces

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
