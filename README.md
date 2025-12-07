# sigfmt — Linter for Go Function Signatures

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue)](https://golang.org/dl/)
[![CI](https://github.com/vsfedorenko/sigfmt/actions/workflows/ci.yml/badge.svg)](https://github.com/vsfedorenko/sigfmt/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/vsfedorenko/sigfmt/branch/main/graph/badge.svg)](https://codecov.io/gh/vsfedorenko/sigfmt)
[![Go Report Card](https://goreportcard.com/badge/github.com/vsfedorenko/sigfmt)](https://goreportcard.com/report/github.com/vsfedorenko/sigfmt)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`sigfmt` is a powerful linter plugin for `golangci-lint` designed for automatic checking and formatting of function, method, and field function signatures in Go.

## 🎯 Motivation

### The Problem

The standard `gofmt` formatter provides basic formatting but leaves freedom of choice when breaking long function signatures across multiple lines. This often leads to inconsistency, poor readability, and polluted git history.

**1. Polluted git history:**
Unnecessary changes appear in diffs when different developers format code differently.

```diff
- func Save(data []byte,
-     path string) error {
+ func Save(data []byte, path string) error {
```

**2. Bloated interfaces and structs:**
Without parameter packing, interfaces and structs take up excessive vertical space.

```go
// Takes 16 lines for just three simple methods!
type UserService interface {
    Create(
        name string,
        email string,
        age int,
    ) error
    // ...
}
```

### The Solution

`sigfmt` solves these problems by enforcing strict but reasonable rules:
- **Compactness**: If a signature fits on one line (considering the limit), it *must* be on one line.
- **Structure**: If a signature is long, it should be formatted to use vertical space most efficiently.
- **Automation**: The linter points out problems and suggests automatic fixes.

## 🚀 Features

The linter analyzes:
- **Function declarations** (`func Foo(...)`)
- **Type methods** (`func (s *S) Bar(...)`)
- **Anonymous functions / Literals** (`var f = func(...)`)
- **Methods in interfaces** (`type I interface { Method(...) }`)
- **Struct fields with function type** (`type S struct { Callback func(...) }`)
- **Generics (Go 1.18+)**: Correct handling of type parameters `[T any]`.

## 🛠 Installation

`sigfmt` is a plugin for `golangci-lint`. You need to build a custom version of `golangci-lint` that includes this plugin.

1.  Create a `.custom-gcl.yml` file in your project root:
    ```yaml
    version: v2.7.1 # Use your desired golangci-lint version
    plugins:
      - module: 'github.com/vsfedorenko/sigfmt'
        version: v0.1.0 # Replace with the latest version
    ```

2.  Run the command to build the custom binary:
    ```bash
    golangci-lint custom
    ```
    This will create a `custom-gcl` binary in your directory.

## ⚙️ Configuration

Configure the linter in your `.golangci.yml` file.

**Parameters:**
*   `max-line-len` (int): Maximum allowed line length. Default: `120`.
*   `tab-width` (int): Tab width for visual calculation. Default: `8`.
*   `pack-struct-fields` (bool): Enable packing of struct fields. Default: `true`.
*   `pack-interface-methods` (bool): Enable packing of interface methods. Default: `true`.

**Example `.golangci.yml`:**

```yaml
linters-settings:
  custom:
    sigfmt:
      type: "module"
      description: "Advanced function signature formatter"
      settings:
        max-line-len: 120
        tab-width: 8
        pack-struct-fields: true
        pack-interface-methods: true
```

## 💡 Usage

To run the linter, use your custom binary:

```bash
# Run analysis
./custom-gcl run

# Automatically fix issues
./custom-gcl run --fix
```

## 🧠 How It Works

The linter operates in two stages:

#### 1. Collapse
Checks if the entire signature can be written on one line within `max-line-len`.

```diff
- func Sum(
-     a int,
-     b int,
- ) int { ... }
+ func Sum(a int, b int) int { ... }
```

#### 2. Reformat / Packing
If a signature **doesn't** fit on one line, the linter applies context-aware strategies.

*   **Regular functions**: Conservative approach. Preserves existing grouping unless clearly broken.
*   **Interfaces and Structs**: Aggressive packing to save vertical space.

```diff
type Logger interface {
-     Log(
-         level Level,
-         msg string,
-         args ...interface{},
-     )
+     Log(level Level, msg string, args ...interface{})
}
```

## 📸 Examples Gallery

### API Handlers
```diff
- func CreateUser(
-     w http.ResponseWriter,
-     r *http.Request,
- ) {
+ func CreateUser(w http.ResponseWriter, r *http.Request) {
      // ...
  }

- func UpdateUser(w http.ResponseWriter, r *http.Request, id string,
-     name string, email string) {
+ func UpdateUser(w http.ResponseWriter, r *http.Request, id string, name string, email string) {
      // ...
  }
```

### Service Interfaces
```diff
- type UserRepository interface {
-     Create(
-         ctx context.Context,
-         name string,
-         email string,
-     ) (*User, error)
-
-     Update(
-         ctx context.Context,
-         id int,
-         name string,
-         email string,
-     ) error
-
-     Delete(
-         ctx context.Context,
-         id int,
-     ) error
- }
+ type UserRepository interface {
+     Create(ctx context.Context, name string, email string) (*User, error)
+     Update(ctx context.Context, id int, name string, email string) error
+     Delete(ctx context.Context, id int) error
+ }
```
*(Reduced from 121 lines to 45 lines — 63% savings)*

### Generics (Go 1.18+)
```diff
- func Map[T any, R any](
-     slice []T,
-     fn func(T) R,
- ) []R {
+ func Map[T any, R any](slice []T, fn func(T) R) []R {
      // ...
  }
```

### Struct Fields
```diff
type Component struct {
-     Render func(
-         ctx context.Context,
-         props Props,
-     ) (Node, error)
+     Render func(ctx context.Context, props Props) (Node, error)
}
```

## ⚡ Performance

- **Fast**: Does not require type loading (`register.LoadModeSyntax`).
- **Efficient**: Uses minimal memory (~20-30MB for 10k LOC) and works only with AST.
- **Parallel**: Supports parallel execution via `golangci-lint`.

## 🤔 FAQ

**Why not use `gofmt` or `gofumpt`?**
They provide basic formatting but lack strict rules for line length and parameter packing. `sigfmt` complements them.

**Can parameter packing be disabled?**
Yes, use `pack-struct-fields: false` and `pack-interface-methods: false`.

**How to handle special cases?**
Use `//nolint:sigfmt` to ignore specific functions.

## 🤝 Contributing

Contributions are welcome!
1.  Fork the repository.
2.  Create a branch.
3.  Ensure tests pass (`make test`).
4.  Submit a Pull Request.

## 📝 License

MIT License

## 🔗 Useful Links

- [golangci-lint documentation](https://golangci-lint.run/)
- [Developing custom linters](https://golangci-lint.run/contributing/new-linters/)
