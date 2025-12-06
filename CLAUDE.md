# CLAUDE.md

This file contains context and guidelines for AI assistants (Claude, Gemini, etc.) when working with code in this repository.

## Project Overview

**Name:** `funcwrap`
**Type:** Custom linter plugin for `golangci-lint`
**Language:** Go (requires Go 1.18+ for generics support)
**Main Goal:** Automatic checking and formatting of function signatures in Go code

### What the Linter Analyzes

The linter processes all function signature constructs in Go:
- **Function declarations** (`func Foo(...)`)
- **Method declarations** (`func (s *S) Bar(...)`)
- **Anonymous functions / Function literals** (`var f = func(...)`)
- **Interface methods** (`type I interface { Method(...) }`)
- **Struct fields with func type** (`type S struct { Callback func(...) }`)
- **Generic functions** (Go 1.18+): Correctly handles type parameters `[T any, R comparable]`

## Core Logic

The linter operates in two modes:

### 1. Collapse Mode
**Trigger:** Multi-line signature fits on a single line within `max-line-len` limit
**Action:** Suggest collapsing to one line
**Diagnostic:** `"Multi-line signature can be collapsed to one line"`

**Example:**
```go
// Before (3 lines, but fits in one)
func Sum(
    a int,
    b int,
) int { ... }

// After (collapsed)
func Sum(a int, b int) int { ... }
```

### 2. Reformat (Packing) Mode
**Trigger:** Signature doesn't fit on one line
**Behavior depends on context:**

- **For regular functions:** Conservative approach - preserves existing multi-line formatting to avoid breaking intentional parameter grouping
- **For interface methods & struct fields:** Aggressive packing - attempts to fit multiple parameters per line to reduce vertical space

**Diagnostic:** `"Signature can be reformatted more compactly"`

**Example (Interface):**
```go
// Before (too sparse)
type Logger interface {
    Log(
        level Level,
        msg string,
        args ...interface{},
    )
}

// After (packed)
type Logger interface {
    Log(level Level, msg string, args ...interface{})
}
```

## Architecture

### File Structure

**`linter.go`** - Main implementation (~900 lines)
- **`PluginLineWrap`**: Plugin structure implementing `golangci-lint` plugin interface
- **`Settings`**: Configuration structure with all linter settings
- **`signatureInfo`**: Internal structure storing signature analysis results
  - Contains both original position and reformatted text
  - Tracks whether to collapse or reformat
- **`checkSignature`**: Decision engine determining action (`collapse`, `reformat`, or skip)
  - Calculates line lengths considering tabs and indentation
  - Applies different strategies based on node type
- **`buildReformattedSignature`**: Generates new signature text
- **`renderFieldListGrouped`**: Smart parameter packing logic
- **Helper renderers**: `renderNode`, `renderFieldList`, `renderType` for precise AST-to-text conversion

**`linter_test.go`** - Test suite (~200 lines, recently refactored)
- Uses `golang.org/x/tools/go/analysis/analysistest` framework
- **`runTestWithSettings`**: Unified test runner (consolidated from two previous functions)
- **Golden file management**:
  - `updateGoldenFiles`: Main orchestrator
  - `collectEditsFromResults`: Groups edits by file
  - `applyEditsToFile`: Applies edits with offset management
  - `applyEdit`: Single edit application
- **Constants**: Extracted for maintainability (e.g., `settingMaxLineLen`)

### Key Constants (linter.go:14-27)

```go
const (
    defaultMaxLineLen         = 120
    defaultTabWidth           = 8
    analyzerName              = "funcwrap"
    diagnosticMessage         = "Multi-line signature can be collapsed to one line"
    diagnosticMessageReformat = "Signature can be reformatted more compactly"
    fixMessage                = "Collapse to one line"
    fixMessageReformat        = "Reformat with grouped parameters"
    actionCollapse            = "collapse"
    actionReformat            = "reformat"
)
```

## Configuration

Settings are passed as a map to the `New` method. All settings have sensible defaults.

### Available Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `max-line-len` | int | 120 | Maximum allowed line length |
| `tab-width` | int | 8 | Tab width for visual length calculation |
| `pack-struct-fields` | bool | true | Enable aggressive packing for struct func fields |
| `pack-interface-methods` | bool | true | Enable aggressive packing for interface methods |

### Example Configuration (.golangci.yml)

```yaml
linters-settings:
  custom:
    funcwrap:
      path: .bin/linters/funcwrap.so
      description: "Advanced function signature formatter"
      settings:
        max-line-len: 120        # Maximum line length
        tab-width: 8             # Tab width for calculations
        pack-struct-fields: true # Pack struct fields aggressively
        pack-interface-methods: true # Pack interface methods aggressively
```

## Testing

### Test Directory Structure

```
testdata/src/
├── features/          # Language construct tests
│   ├── anonymous.go   # Anonymous functions, blank identifiers
│   ├── comments.go    # Functions with comments
│   ├── complex_types.go # Maps, channels, complex types
│   ├── functions.go   # Standard function declarations
│   ├── generics.go    # Go 1.18+ generics
│   ├── interfaces.go  # Interface methods
│   └── struct_fields.go # Struct fields with func types
├── limits/            # Line length boundary tests
│   ├── length80/      # 80 character limit
│   ├── length100/     # 100 character limit
│   ├── length120/     # 120 character limit (default)
│   ├── length140/     # 140 character limit
│   └── length160/     # 160 character limit
└── settings/          # Configuration tests
    ├── defaults/      # Default settings
    ├── no_pack/       # Packing disabled
    ├── pack_structs/  # Only struct packing
    └── pack_interfaces/ # Only interface packing
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Update golden files (use when diagnostic messages change)
go test -update ./...

# Using Makefile
make test              # Run tests
make test-race         # Run with race detector
make test-coverage     # Run with coverage report
make test-update-golden # Update golden files
```

### Test Methodology

- Each `.go` file contains code with `// want "diagnostic message"` comments
- Golden files (`.go.golden`) contain expected fixed code
- Framework compares suggested fixes against golden files
- Tests verify both diagnostic messages AND correct fixes

## Implementation Details

### Technical Approach

- **AST-based**: Uses `go/ast` for parsing, `go/token` for positions
- **Fast**: Does NOT require type information (`register.LoadModeSyntax`)
  - Only syntax-level analysis
  - No expensive type checking
- **Custom rendering**: Implements own AST-to-text conversion
  - Standard `go/printer` doesn't give enough control over whitespace
  - Custom logic in `renderNode`, `renderFieldList`, etc.
  - Precise control over commas, spaces, line breaks
- **Generic-aware**: Properly handles type parameters `[T any]` without special-casing

### Performance Characteristics

- Memory: ~20-30MB for typical 10k LOC codebase
- Speed: ~100-200ms analysis time
- No type loading overhead
- Suitable for CI/CD pipelines

## Development Workflow

### Making Changes to Diagnostic Messages

When updating diagnostic messages:
1. Update constants in `linter.go`
2. Run tests - they will fail
3. Update test files: `find testdata/src -name "*.go" -exec sed -i '' 's/OLD_MESSAGE/NEW_MESSAGE/g' {} \;`
4. Update golden files: `find testdata/src -name "*.golden" -exec sed -i '' 's/OLD_MESSAGE/NEW_MESSAGE/g' {} \;`
5. Verify: `go test ./...`

### Code Style Notes

- Comments are in English (recently standardized from mixed RU/EN)
- Test file refactored to eliminate duplication
- Constants used instead of magic strings
- Helper functions broken down for maintainability

## Example Project

The `example/` directory demonstrates real-world integration:
- **`.custom-gcl.yml`**: Builds custom `golangci-lint` binary with this plugin
- **`.golangci.yml`**: Configuration for running the linter
- **Build process**: Uses Go modules to pull in the linter
- **Usage**: `cd example && golangci-lint custom && ./custom-gcl run`

### Purpose of Example

1. **User reference**: Shows how to integrate plugin
2. **Integration test**: Verifies plugin works with `golangci-lint`
3. **Documentation**: Live example of configuration