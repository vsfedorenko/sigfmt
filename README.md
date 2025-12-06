# Line Wrap Linter

This is an example linter that can be used as a plugin for `golangci-lint`.

## Overview

Line Wrap Linter checks Go function signatures that span multiple lines and suggests collapsing them to a single line when they fit within the configured maximum line length (default: 120 characters).

## Features

- ✅ Detects multi-line function signatures
- ✅ Supports Go 1.18+ generics (type parameters)
- ✅ Handles function declarations, methods, and interface methods
- ✅ Provides automatic fixes via SuggestedFixes
- ✅ Configurable maximum line length
- ✅ Comprehensive test coverage

## Supported Constructs

The linter analyzes:
- Function declarations (`func Foo(...)`)
- Methods with receivers (`func (r *Receiver) Foo(...)`)
- Function literals / closures (`var f = func(...)`)
- Interface methods
- Functions with generics (`func Foo[T any](...)`)
- Functions with multiple return values
- Functions with named return values
- Variadic parameters

## Installation

```bash
go get github.com/golangci/example-plugin-module-linter
```

## Configuration

Add to your `.golangci.yml`:

```yaml
linters-settings:
  custom:
    line-wrap:
      type: module
      description: Checks if multi-line function signatures can be collapsed to one line
      settings:
        max-line-len: 120  # Optional, defaults to 120
```

## Examples

### Before
```go
func ShortFunction(
    a int,
    b string,
) error {
    return nil
}
```

### After (with auto-fix)
```go
func ShortFunction(a int, b string) error {
    return nil
}
```

## Testing

```bash
# Run all tests
go test -v ./...

# Build the linter
go build ./...
```

## Development

See [REFACTORING.md](REFACTORING.md) for details on recent improvements and architecture.

See [CLAUDE.md](CLAUDE.md) for AI assistant guidance on working with this codebase.
