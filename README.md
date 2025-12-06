# Line Wrap Linter

This is a linter plugin for `golangci-lint`.

## Overview

Line Wrap Linter checks Go function signatures and suggests better formatting. It mainly focuses on:
1.  **Collapsing** multi-line signatures into a single line if they fit within the configured maximum line length (default: 120 characters).
2.  **Packing** parameters for interface methods and struct function fields to use fewer lines if they don't fit in one line, while respecting the max line length.

## Features

- ✅ Detects multi-line function signatures
- ✅ Supports Go 1.18+ generics (type parameters)
- ✅ Handles:
    - Function declarations (`func Foo(...)`)
    - Methods with receivers (`func (r *Receiver) Foo(...)`)
    - Function literals / closures (`var f = func(...)`)
    - Interface methods
    - Struct fields of function type
- ✅ Provides automatic fixes via SuggestedFixes
- ✅ Configurable maximum line length
- ✅ Smart formatting:
    - **Standalone functions:** Only collapses if fits in one line. Preserves existing multi-line formatting if it doesn't fit.
    - **Interfaces & Structs:** Aggressively packs parameters to optimize vertical space.

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
      description: Checks if multi-line function signatures can be collapsed to one line or formatted better
      settings:
        max-line-len: 120  # Optional, defaults to 120
```

## Examples

### Collapsing (All types)

**Before:**
```go
func ShortFunction(
    a int,
    b string,
) error {
    return nil
}
```

**After (with auto-fix):**
```go
func ShortFunction(a int, b string) error {
    return nil
}
```

### Packing (Interface Methods & Struct Fields)

**Before:**
```go
type MyInterface interface {
    ProcessWithManyParameters(
        p1 string,
        p2 string,
        p3 string,
        p4 string,
    ) error
}
```

**After (with auto-fix, assuming max-len allows):**
```go
type MyInterface interface {
    ProcessWithManyParameters(p1 string, p2 string, p3 string,
        p4 string) error
}
```

## Testing

```bash
# Run all tests
go test -v ./...
```

## Development

The project structure:
- `linter.go`: Core logic.
- `linter_test.go`: Tests runner.
- `testdata/src/features/`: Test cases for language features (interfaces, structs, functions).
- `testdata/src/limits/`: Test cases for different line length limits.

See [CLAUDE.md](CLAUDE.md) or [GEMINI.md](GEMINI.md) for AI assistant guidance.