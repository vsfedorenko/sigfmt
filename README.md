# sigfmt — Linter for Go Function Signatures

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue)](https://golang.org/dl/)
[![CI](https://github.com/vsfedorenko/sigfmt/actions/workflows/ci.yml/badge.svg)](https://github.com/vsfedorenko/sigfmt/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/vsfedorenko/sigfmt/branch/main/graph/badge.svg)](https://codecov.io/gh/vsfedorenko/sigfmt)
[![Go Report Card](https://goreportcard.com/badge/github.com/vsfedorenko/sigfmt)](https://goreportcard.com/report/github.com/vsfedorenko/sigfmt)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`sigfmt` is a powerful linter plugin for `golangci-lint` designed for automatic checking and formatting of function, method, and field function signatures in Go.

## 🎯 Motivation

### The Problem

The standard `gofmt` formatter provides basic formatting but leaves freedom of choice when breaking long function signatures across multiple lines. This often leads to:

1.  **Inconsistency in the codebase**
    ```go
    // File A: all on one line
    func ProcessUser(id int, name string, email string, age int) error { ... }

    // File B: "staircase" style
    func ProcessOrder(
        id int,
        userId int,
        total float64,
        status string,
    ) error { ... }

    // File C: chaotic grouping
    func CreateAccount(id int, name string,
        email string, password string) error { ... }
    ```

2.  **Poor readability**: Lines that are too long force horizontal scrolling, especially during code review.

3.  **Polluted git history**: When different developers format code differently, unnecessary changes appear in diffs:
    ```diff
    - func Save(data []byte,
    -     path string) error {
    + func Save(data []byte, path string) error {
    ```

4.  **Bloated interfaces**: Without parameter packing, interfaces take up too much space:
    ```go
    type UserService interface {
        Create(
            name string,
            email string,
            age int,
        ) error
        Update(
            id int,
            name string,
            email string,
            age int,
        ) error
        Delete(
            id int,
        ) error
    }
    // 16 lines for three simple methods!
    ```

### The Solution

`sigfmt` solves these problems by enforcing strict but reasonable rules:
- **Compactness**: If a signature fits on one line (considering the limit), it *must* be on one line.
- **Structure**: If a signature is long, it should be formatted to use vertical space most efficiently (especially for interfaces).
- **Automation**: The linter not only points out problems but also suggests automatic fixes.
- **Configurability**: Flexible settings for different code styles and line length limits.

## 🚀 Features

The linter analyzes the following Go language constructs:
- **Function declarations** (`func Foo(...)`)
- **Type methods** (`func (s *S) Bar(...)`)
- **Anonymous functions / Literals** (`var f = func(...)`)
- **Methods in interfaces** (`type I interface { Method(...) }`)
- **Struct fields with function type** (`type S struct { Callback func(...) }`)
- **Generics (Go 1.18+)**: Correct handling of type parameters `[T any]`.

### How It Works

The linter operates in two stages:

#### 1. Collapse
Checks if the entire signature (parameters + return values + receiver) can be written on one line so that its length doesn't exceed `max-line-len`.

**Example:**
```go
// Before (bad, takes 3 lines, but fits on one)
func Sum(
    a int,
    b int,
) int { ... }

// After (good, compact)
func Sum(a int, b int) int { ... }
```

#### 2. Reformat / Packing (Smart Reformatting)
If a signature **doesn't** fit on one line, the linter applies different strategies depending on context:

*   **For regular functions**:
    The linter acts conservatively. If you've already split arguments by lines, it tries not to interfere to avoid breaking author grouping. It only intervenes if formatting clearly violates style (e.g., some arguments on one line, some on another without system).

*   **For interfaces and struct fields**:
    The linter acts aggressively (**Packing** strategy). It tries to place several short parameters on one line so interface definition doesn't stretch for 50 screens.

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

    // After (compact packing)
    type Logger interface {
        Log(level Level, msg string, args ...interface{})
    }
    ```

## ⚙️ Configuration

The linter is configured through the `.golangci.yml` file.

**Parameters:**
*   `max-line-len` (int): Maximum allowed line length. Default: `120`.
*   `tab-width` (int): Tab width for calculating visual line length. Default: `8`.
*   `pack-struct-fields` (bool): Enable packing of struct fields with func type. Default: `true`.
*   `pack-interface-methods` (bool): Enable packing of interface methods. Default: `true`.

**Example `.golangci.yml`:**

```yaml
linters-settings:
  custom:
    sigfmt:
      path: .bin/linters/sigfmt.so
      description: "Advanced function signature formatter"
      original-url: github.com/username/sigfmt
      settings:
        max-line-len: 120
        tab-width: 8
        pack-struct-fields: true
        pack-interface-methods: true
```

## 🛠 Installation and Building

Since `sigfmt` is a plugin, it needs to be compiled with the same Go version used to build `golangci-lint`.

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/username/sigfmt.git
    cd sigfmt
    ```

2.  **Build the plugin:**
    ```bash
    go build -buildmode=plugin -o sigfmt.so linter.go
    ```

3.  **Connect in config** (see above).

## 💡 Usage Example

The [example/](example/) directory contains a ready-to-use project example configured to use this linter as a Go module.

To run the example:

1.  Navigate to the example directory:
    ```bash
    cd example
    ```
2.  Build a custom version of `golangci-lint`:
    ```bash
    golangci-lint custom
    ```
3.  Run the linter:
    ```bash
    ./custom-gcl run
    ```

## 🧪 Development and Testing

The project has an extensive test suite covering edge cases and various language constructs.

### Using Makefile

The project includes a `Makefile` for development convenience:

```bash
# Show all available commands
make help

# Run tests
make test

# Run tests with race detector
make test-race

# Run tests with coverage
make test-coverage

# Update golden files
make test-update-golden

# Format code
make fmt

# Run linter
make lint

# Run all checks (fmt, vet, lint, test)
make check

# Build custom golangci-lint binary
make build-example

# Run custom linter on example
make run-example

# Clean build artifacts
make clean
```

### Test Structure

*   `testdata/src/features/`: Tests for language features (generics, comments, variadics).
*   `testdata/src/limits/`: Tests for different line length limits (80, 100, 120, 140, 160).
*   `testdata/src/settings/`: Tests for various linter settings.

### Updating Golden Files

Golden files contain expected results of applying fixes. To update them, use:

```bash
# Via Makefile (recommended)
make test-update-golden

# Directly via GODEBUG
GODEBUG=analysistest.fix=true go test ./...
```

## 📊 Real-World Usage Examples

### Example 1: API Handlers

**Before:**
```go
func CreateUser(
    w http.ResponseWriter,
    r *http.Request,
) {
    // ...
}

func UpdateUser(w http.ResponseWriter, r *http.Request, id string,
    name string, email string) {
    // ...
}
```

**After:**
```go
func CreateUser(w http.ResponseWriter, r *http.Request) {
    // ...
}

func UpdateUser(w http.ResponseWriter, r *http.Request, id string, name string, email string) {
    // ...
}
```

### Example 2: Service Interfaces

**Before (121 lines for 8 methods):**
```go
type UserRepository interface {
    Create(
        ctx context.Context,
        name string,
        email string,
    ) (*User, error)

    Update(
        ctx context.Context,
        id int,
        name string,
        email string,
    ) error

    Delete(
        ctx context.Context,
        id int,
    ) error
    // ... 5 more methods
}
```

**After (45 lines for the same 8 methods - 63% savings):**
```go
type UserRepository interface {
    Create(ctx context.Context, name string, email string) (*User, error)
    Update(ctx context.Context, id int, name string, email string) error
    Delete(ctx context.Context, id int) error
    // ... 5 more methods
}
```

### Example 3: Generics (Go 1.18+)

**Before:**
```go
func Map[T any, R any](
    slice []T,
    fn func(T) R,
) []R {
    // ...
}
```

**After:**
```go
func Map[T any, R any](slice []T, fn func(T) R) []R {
    // ...
}
```

## ⚡ Performance

- **Fast execution**: Doesn't require loading type information (`register.LoadModeSyntax`)
- **Minimal memory consumption**: Works only with AST and tokens
- **Parallel processing**: Can analyze multiple files simultaneously via `golangci-lint`

On a medium codebase (10k LOC):
- Analysis time: ~100-200ms
- Memory: ~20-30MB

## 🤔 FAQ

### Why not use `gofmt` or `gofumpt`?

`gofmt` and `gofumpt` are great for basic formatting, but they don't enforce strict rules on line length and parameter packing. `sigfmt` complements them by providing an additional level of consistency.

### Can parameter packing be disabled?

Yes! Use the `pack-struct-fields: false` and `pack-interface-methods: false` settings to disable aggressive parameter packing.

### Does it work with Go 1.18+ Generics?

Yes, `sigfmt` fully supports generics, including type parameters in square brackets.

### How to handle special cases?

Use the `//nolint:sigfmt` comment to disable the linter for a specific function:

```go
//nolint:sigfmt
func SpecialCase(
    a int,
    b string,
) {
    // Linter will skip this function
}
```

### Is automatic fixing supported?

Yes! The linter provides suggested fixes that can be applied via:
```bash
golangci-lint run --fix
```

## 🛡️ Implementation Details

*   **Based on `go/ast` and `go/token`**: Reliable parsing via standard Go libraries
*   **Fast execution**: Doesn't require heavy type loading (`register.LoadModeSyntax`)
*   **Precise control**: Uses custom AST node renderer for precise control over spaces and commas, unlike standard `go/printer`
*   **Generics support**: Correct handling of type parameters `[T any]` (Go 1.18+)
*   **Test coverage**: Over 100 test cases, including edge cases

## 🤝 Contributing

We welcome contributions to the project! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make sure tests pass (`make test`)
4. Commit changes (`git commit -m 'Add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

### PR Requirements:
- ✅ All tests pass (`make test`)
- ✅ Code is formatted (`make fmt`)
- ✅ Linters pass without errors (`make lint`)
- ✅ Tests added for new functionality
- ✅ Documentation updated (if necessary)

## 📝 License

MIT License

## 🔗 Useful Links

- [golangci-lint documentation](https://golangci-lint.run/)
- [Go AST package](https://pkg.go.dev/go/ast)
- [Developing custom linters](https://golangci-lint.run/contributing/new-linters/)

---

**Technical documentation for AI assistants:**
- [CLAUDE.md](CLAUDE.md) — context for Claude AI
- [GEMINI.md](GEMINI.md) — context for Gemini AI
