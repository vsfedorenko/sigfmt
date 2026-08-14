# sigfmt CLI

A standalone command-line tool for checking and formatting Go function
signatures — the same logic as the [`sigfmt`](../..) golangci-lint plugin, usable
without golangci-lint.

## Installation

```bash
go install github.com/vsfedorenko/sigfmt/cmd/sigfmt@latest
```

## Quick Start

```bash
# Check all packages in the current module
sigfmt ./...

# Check a specific package
sigfmt ./internal/format

# Check a specific file
sigfmt main.go
```

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--max-line-len` | `120` | Maximum line length. Signatures that fit within this limit are collapsed to one line. |
| `--tab-width` | `8` | Visual width of a tab character for length calculations. |
| `--pack-struct-fields` | `true` | Aggressively pack function-type struct fields (multiple params per line). |
| `--pack-interface-methods` | `true` | Aggressively pack method signatures in interfaces (multiple params per line). |
| `--fix` | `false` | Apply all suggested fixes automatically and write the results back to the source files. |

> Booleans are specified as `--pack-struct-fields=false`. The `--fix` flag is
> provided by the underlying analysis driver.

## Usage Examples

### Report only (CI mode)

```bash
$ sigfmt ./...
/Users/alice/project/service.go:45:4: Signature can be formatted more compactly
/Users/alice/project/handler.go:102:5: Signature can be formatted more compactly
```

Exit code is non-zero if any diagnostics are found, making it suitable for CI:

```bash
sigfmt ./... && echo "All signatures formatted correctly"
```

### Apply fixes automatically

```bash
sigfmt --fix ./...
```

This rewrites the affected files in place with the suggested formatting.

### Custom line length

```bash
# Enforce a stricter 80-column limit
sigfmt --max-line-len=80 ./...
```

### Disable interface packing

```bash
sigfmt --pack-interface-methods=false ./...
```

## How It Works

The CLI uses the exact same analysis pipeline as the golangci-lint plugin:

1. Parses Go source files using `go/parser` (syntax-only, no type loading — fast).
2. Walks the AST for function declarations, method expressions, function
   literals, interface methods, and struct fields of function type.
3. For each signature, applies formatting strategies (collapse, pack, param
   groups, consistency) and reports a diagnostic with a suggested fix.

See the [main project README](../..) for details on the formatting rules.

## Differences from the golangci-lint Plugin

| Feature | golangci-lint plugin | standalone CLI |
| --- | --- | --- |
| Requires golangci-lint | Yes (custom build) | No |
| `--fix` support | Via `golangci-lint --fix` | Built-in |
| Param groups config | golangci-lint YAML | Not available in CLI¹ |
| Runs alongside other linters | Yes | No |

¹ Param groups are a niche feature configured per-project; they remain
available through the golangci-lint plugin.
