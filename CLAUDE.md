# CLAUDE.md

This file contains context and guidelines for AI assistants (Claude, Gemini, etc.) when working with code in this repository.

## Project Overview

This is a linter plugin for `golangci-lint`.
Main goal: checking and improving the formatting of function signatures in Go.

The linter analyzes:
- Function and method declarations.
- Function literals.
- Interface methods.
- Struct fields with `func` type.

## Logic

1.  **Collapse:** If a signature (parameters + return values) fits on a single line (considering `MaxLineLen`), the linter suggests collapsing it.
2.  **Reformat (Packing):**
    - For **interface methods** and **struct fields**: The linter attempts to pack parameters more compactly (multiple per line) if they do not fit on a single line.
    - For **regular functions**: The linter preserves existing multi-line formatting if it already exists and the signature does not fit on one line (to avoid messing up manual formatting).

## Architecture

-   `linter.go`: Main implementation.
    -   `PluginLineWrap`: plugin structure.
    -   `signatureInfo`: structure to store information about the found signature.
    -   `checkSignature`: determines the action (`collapse`, `reformat`, or nothing).
    -   `buildReformattedSignature` / `renderFieldListGrouped`: logic for generating new text.
-   `linter_test.go`: Running tests via `analysistest`.

## Testing

Test data is located in `testdata/src/`:
-   `features/`: Functionality tests (various language constructs).
-   `limits/`: Boundary value tests (80, 100, 120, 140, 160 characters).

Running tests:
```bash
go test ./...
```

## Configuration

Settings are passed via a map to the `New` method.
Main setting: `max-line-len` (int, default 120).

## Implementation Details

-   Uses `go/ast` and `go/token`.
-   Does not require type information (`register.LoadModeSyntax`).
-   Custom logic for rendering AST nodes (`renderNode`, `renderFieldList`) for precise format control.
-   Handling of Generics (type parameters).

## Example Project

The `example/` directory contains a self-contained Go project that demonstrates how to integrate this linter as a module plugin for `golangci-lint`.
-   It includes `.custom-gcl.yml` for building a custom linter binary.
-   It includes `.golangci.yml` for configuring the linter.
-   It serves as a reference for users and a verification step for developers.