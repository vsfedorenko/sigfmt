# sigfmt — Linter for Go Function Signatures

[![Go Reference](https://pkg.go.dev/badge/github.com/vsfedorenko/sigfmt.svg)](https://pkg.go.dev/github.com/vsfedorenko/sigfmt)
[![GitHub Release](https://img.shields.io/github/v/release/vsfedorenko/sigfmt)](https://github.com/vsfedorenko/sigfmt/releases)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue)](https://golang.org/dl/)
[![CI](https://github.com/vsfedorenko/sigfmt/actions/workflows/ci.yml/badge.svg)](https://github.com/vsfedorenko/sigfmt/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/vsfedorenko/sigfmt/branch/main/graph/badge.svg)](https://codecov.io/gh/vsfedorenko/sigfmt)
[![Go Report Card](https://goreportcard.com/badge/github.com/vsfedorenko/sigfmt)](https://goreportcard.com/report/github.com/vsfedorenko/sigfmt)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`sigfmt` is a **golangci-lint plugin** and **Go function formatter** that
automatically checks and formats function, method, and field signatures —
enforcing consistent **code style** across your Go codebase. It collapses
multi-line signatures that fit on one line and packs long signatures compactly,
keeping diffs clean and code readable.

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

### Option 1: Automatic (Recommended)

1.  Create a `.custom-gcl.yml` file in your project root:
    ```yaml
    version: v1.64.6 # Use your desired golangci-lint version
    plugins:
      - module: 'github.com/vsfedorenko/sigfmt'
        version: v0.1.0 # Replace with the latest version
    ```

2.  Run the command to build the custom binary:
    ```bash
    golangci-lint custom
    ```
    This will create a `custom-gcl` binary in your directory.

### Option 2: Manual

1.  Add a blank import of the module inside `cmd/golangci-lint/plugins.go` (or equivalent):
    ```go
    import _ "github.com/vsfedorenko/sigfmt"
    ```
2.  Run `go mod tidy` and `make build` to produce your custom `golangci-lint`.

## ⚙️ Configuration

Configure the linter in your `.golangci.yml` file.

**Parameters:**
*   `max-line-len` (int): Maximum allowed line length. Default: `120`.
*   `tab-width` (int): Tab width for visual calculation. Default: `8`.
*   `pack-struct-fields` (bool): Enable packing of struct fields. Default: `true`.
*   `pack-interface-methods` (bool): Enable packing of interface methods. Default: `true`.
*   `param-groups` (list of lists): Define groups of parameter types that should be kept together on the same line.
*   `ignore-tests` (bool): Skip `_test.go` files entirely. Test files are frequently table-driven with intentionally wide signatures; many teams prefer formatting them manually. Default: `false`.

**Generated files are skipped automatically.** Files carrying the
conventional [`Code generated ... DO NOT EDIT.`](https://go.dev/s/generatedcode)
header before the package clause (e.g. `*.pb.go` output, mocks, `zz_*`
stringers) produce no diagnostics — generated code is not hand-maintained.
There is no setting for this: it always applies, matching the behaviour of
golangci-lint core linters.

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
        param-groups:
          - ["context.Context", "*sql.Tx"] # Group ctx and tx together
          - ["context.Context"]            # Ensure ctx is on its own line (if no tx)
```

## 💡 Usage

### As a golangci-lint plugin

To run the linter, use your custom binary:

```bash
# Run analysis
./custom-gcl run

# Automatically fix issues
./custom-gcl run --fix
```

### Standalone CLI

The `sigfmt` binary can also be used directly without golangci-lint:

```bash
# Run analysis on a package
sigfmt ./...

# Automatically fix issues in-place
sigfmt -fix ./...

# With custom settings
sigfmt -max-line-len 100 -tab-width 4 ./...

# With parameter groups
sigfmt -param-groups "context.Context,error;io.Reader,io.Writer" ./...
```

Available CLI flags:

| Flag | Default | Description |
|---|---|---|
| `-max-line-len` | `120` | Maximum line length before multi-line signatures are required |
| `-tab-width` | `8` | Visual width of a tab character for length calculations |
| `-pack-struct-fields` | `true` | Aggressively pack function-type struct fields |
| `-pack-interface-methods` | `true` | Aggressively pack method signatures in interfaces |
| `-param-groups` | _(none)_ | Semicolon-separated parameter type groups (e.g. `"context.Context,error;io.Reader,io.Writer"`) |
| `-fix` | `false` | Apply suggested fixes automatically (provided by `singlechecker`) |
| `-V` | _(version)_ | Print analyzer version and exit |

## 🧠 How It Works

The linter analyzes function signatures using the Go AST (Abstract Syntax Tree) and applies a two-stage formatting strategy:

#### 1. Collapse Stage
**Goal:** Maximize compactness for short signatures.

The linter calculates the visual width of a signature if it were on a single line. If it fits within `max-line-len` (default 120), the signature **must** be collapsed.

**Example:**
```diff
- func Sum(
-     a int,
-     b int,
- ) int { ... }
+ func Sum(a int, b int) int { ... }
```

**Diagnostic Message:** `"Signature can be formatted more compactly"`
**Suggested Fix Message:** `"Format signature"`

This applies to:
- Function declarations
- Type methods (with receivers)
- Anonymous functions / closures
- Interface methods
- Struct fields with function types

#### 2. Reformat / Packing Stage
**Goal:** Optimize vertical space for long signatures.

If a signature **doesn't** fit on one line, the linter applies context-aware packing strategies.
The diagnostic message for this stage is the same as for the collapse stage: `"Signature can be formatted more compactly"`, with the fix message `"Format signature"`.

##### A. Regular Functions (Conservative)
Preserves logical parameter grouping. Uses minimal reformatting to respect developer intent.

```diff
  func ProcessData(
-     param1 string,
-     param2 string,
-     param3 int,
-     param4 bool,
  ) error {
+ func ProcessData(
+     param1 string, param2 string,
+     param3 int, param4 bool,
+ ) error {
      // ...
  }
```

##### B. Interfaces & Structs (Aggressive)
Packs multiple parameters per line to minimize vertical space bloat. Interface definitions often have many similar methods, so aggressive packing significantly improves readability.

```diff
  type Logger interface {
-     Log(
-         level Level,
-         msg string,
-         args ...interface{},
-     )
+     Log(level Level, msg string, args ...interface{})

-     Error(
-         msg string,
-         err error,
-     )
+     Error(msg string, err error)
  }
```

##### C. Parameter Groups (Advanced)
When configured with `param-groups`, the linter keeps semantically related parameters together on the same line:

```yaml
param-groups:
  - ["context.Context", "*sql.Tx"]  # Always group ctx and tx
  - ["context.Context"]              # If no tx, keep ctx on its own line
```

```diff
- func Query(
-     ctx context.Context,
-     tx *sql.Tx,
-     sql string,
-     args ...interface{},
- ) error {
+ func Query(
+     ctx context.Context, tx *sql.Tx,
+     sql string, args ...interface{},
+ ) error {
      // ...
  }
```

#### Width Calculation
The linter calculates visual width considering:
- Tab expansion (`tab-width`, default 8)
- Receiver length (for methods)
- Type parameter length (for generics)
- Return type length
- Comment preservation

**Example:**
```go
// Visual width = len("func Map[T any, R any](items []T, fn func(T) R) []R")
func Map[T any, R any](items []T, fn func(T) R) []R  // 54 chars (fits in 120)
```

## 📸 Examples Gallery

### 1. Basic Function Collapsing

**Simple Functions:**
```diff
- func ShortFunction(
-     a int,
-     b string,
- ) error {
+ func ShortFunction(a int, b string) error {
      return nil
  }

- func Sum(
-     nums ...int,
- ) int {
+ func Sum(nums ...int) int {
      total := 0
      for _, n := range nums {
          total += n
      }
      return total
  }
```

**Multiple Return Values:**
```diff
- func MultipleReturns(
-     x int,
-     y int,
- ) (int, error) {
+ func MultipleReturns(x int, y int) (int, error) {
      return x + y, nil
  }

- func NamedReturns(
-     a int,
-     b int,
- ) (sum int, err error) {
+ func NamedReturns(a int, b int) (sum int, err error) {
      return a + b, nil
  }
```

**Mixed Parameters (shorthand notation):**
```diff
- func MixedParams(
-     a, b int,
-     c string,
- ) error {
+ func MixedParams(a, b int, c string) error {
      return nil
  }
```

### 2. Methods

**Type Methods:**
```diff
  type Calculator struct{}

- func (c *Calculator) Add(
-     a int,
-     b int,
- ) int {
+ func (c *Calculator) Add(a int, b int) int {
      return a + b
  }
```

**Anonymous Functions / Closures:**
```diff
- var myFunc = func(
-     a int,
-     b int,
- ) int {
+ var myFunc = func(a int, b int) int {
      return a + b
  }

  group.Go(
-     func(
-         ctx context.Context,
-     ) error {
+     func(ctx context.Context) error {
          // ...
      },
  )
```

### 3. Interfaces (Aggressive Packing)

**Simple Interface Methods:**
```diff
  type MyInterface interface {
-     Method(
-         ctx context.Context,
-     ) error
+     Method(ctx context.Context) error

-     Get(
-         id string,
-     ) error
+     Get(id string) error

-     GetMultiple(
-         id string,
-     ) (string, error)
+     GetMultiple(id string) (string, error)
  }
```

**Complex Interfaces (Packing Multiple Parameters Per Line):**
```diff
  type ComplexInterface interface {
-     ProcessWithVeryLongNameAndManyParameters(
-         parameterOne string,
-         parameterTwo string,
-         parameterThree string,
-         parameterFour string,
-     ) error
+     ProcessWithVeryLongNameAndManyParameters(parameterOne string, parameterTwo string,
+         parameterThree string, parameterFour string) error

-     ProcessManyParams(
-         parameterOne string,
-         parameterTwo string,
-         parameterThree string,
-         parameterFour string,
-         parameterFive string,
-         parameterSix string,
-         parameterSeven string,
-         parameterEight string,
-     ) error
+     ProcessManyParams(parameterOne string, parameterTwo string, parameterThree string,
+         parameterFour string, parameterFive string, parameterSix string,
+         parameterSeven string, parameterEight string) error
  }
```

**Service Interfaces (Real-World Example):**
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

**Variadic Interface Methods:**
```diff
  type VariadicInterface interface {
-     Process(
-         items ...string,
-     ) error
+     Process(items ...string) error
  }
```

**Handler Interfaces (Functional Parameters):**
```diff
  type HandlerInterface interface {
-     Handle(
-         ctx context.Context,
-         handler func(string) error,
-     ) error
+     Handle(ctx context.Context, handler func(string) error) error

      HandleMultiple(ctx context.Context, handlers ...func(string) error) error
  }
```

### 4. Struct Fields with Function Types

**Simple Handlers:**
```diff
  type Handler struct {
-     Process func(
-         ctx context.Context,
-     ) error
+     Process func(ctx context.Context) error
  }
```

**Multiple Function Fields:**
```diff
  type MultiHandler struct {
-     OnStart func(
-         id string,
-     ) error
+     OnStart func(id string) error

      OnStop func() error

-     OnProcessWithVeryLongNameAndManyParameters func(
-         parameterOne string,
-         parameterTwo string,
-         parameterThree string,
-         parameterFour string,
-     ) error
+     OnProcessWithVeryLongNameAndManyParameters func(parameterOne string,
+         parameterTwo string, parameterThree string, parameterFour string) error

-     GetData func(
-         key string,
-     ) (string, error)
+     GetData func(key string) (string, error)
  }
```

**Variadic and Named Returns:**
```diff
  type VariadicHandler struct {
-     Process func(
-         items ...string,
-     ) error
+     Process func(items ...string) error
  }

  type NamedReturnsHandler struct {
-     Process func(
-         id string,
-     ) (result string, err error)
+     Process func(id string) (result string, err error)
  }
```

**Higher-Order Functions:**
```diff
  type HigherOrderHandler struct {
-     GetHandler func(
-         config string,
-     ) func(string) error
+     GetHandler func(config string) func(string) error
  }

  type CallbackHandler struct {
-     Process func(
-         callback func(string) error,
-     ) error
+     Process func(callback func(string) error) error

      ProcessMultiple func(callback func(string) error, fallback func() error) error
  }
```

### 5. Generics (Go 1.18+)

**Basic Generics:**
```diff
- func Generic[
-     T any,
- ](
-     val T,
- ) {
+ func Generic[T any](val T) {
      // ...
  }
```

**Multiple Type Parameters:**
```diff
- func Map[
-     T any,
-     R any,
- ](
-     items []T,
-     fn func(T) R,
- ) []R {
+ func Map[T any, R any](items []T, fn func(T) R) []R {
      result := make([]R, len(items))
      for i, item := range items {
          result[i] = fn(item)
      }
      return result
  }
```

**Generic Interfaces:**
```diff
  type GenericInterface[T any] interface {
-     Process(
-         item T,
-     ) error
+     Process(item T) error

      GetAll() []T
  }

  type MultiGenericInterface[K comparable, V any] interface {
-     Get(
-         key K,
-     ) (V, bool)
+     Get(key K) (V, bool)

-     Set(
-         key K,
-         value V,
-     ) error
+     Set(key K, value V) error

      Delete(key K)
  }
```

**Generic Struct Fields:**
```diff
  type GenericHandler[T any] struct {
-     Process func(
-         item T,
-     ) error
+     Process func(item T) error

      Transform func(item T) T
  }
```

### 6. Complex Type Definitions

**Channel Types:**
```diff
- func Stream(
-     ctx context.Context,
-     in <-chan Item,
-     out chan<- Result,
- ) error {
+ func Stream(ctx context.Context, in <-chan Item, out chan<- Result) error {
      // ...
  }
```

**API Handlers:**
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

**Complex Order Processing:**
```diff
- func ProcessOrder(
-     ctx context.Context,
-     orderID string,
-     items []Item,
-     shippingAddress *Address,
-     paymentMethod PaymentMethod,
-     options ...Option,
- ) (
-     *Order,
-     error,
- ) {
+ func ProcessOrder(ctx context.Context, orderID string, items []Item, shippingAddress *Address, paymentMethod PaymentMethod, options ...Option) (*Order, error) {
      // ...
  }
```

### 7. Parameter Grouping (Advanced Feature)

When using `param-groups` configuration, related parameters are kept together for better semantic organization:

**Database Operations (context.Context + *sql.Tx grouping):**
```diff
  // Configuration: param-groups: [["context.Context", "*sql.Tx"]]

- func LongQueryFunctionWithManyArguments(
-     ctx context.Context,
-     tx *sql.Tx,
-     query string,
-     args ...interface{},
- ) error {
+ func LongQueryFunctionWithManyArguments(
+     ctx context.Context, tx *sql.Tx,
+     query string, args ...interface{}) error {
      return nil
  }
```

**Repository Interface with Grouping:**
```diff
  type Repository interface {
-     Create(
-         ctx context.Context,
-         tx *sql.Tx,
-         id int,
-         name string,
-     ) error
+     Create(ctx context.Context, tx *sql.Tx,
+         id int, name string) error

-     Update(
-         ctx context.Context,
-         data []byte,
-     ) error
+     Update(ctx context.Context, data []byte) error
  }
```

**Handler Functions with Grouping:**
```diff
  type Handler struct {
-     OnCreate func(
-         ctx context.Context,
-         tx *sql.Tx,
-         data string,
-     ) error
+     OnCreate func(ctx context.Context,
+         tx *sql.Tx, data string) error

-     OnUpdate func(
-         ctx context.Context,
-         id int,
-     ) error
+     OnUpdate func(ctx context.Context, id int) error
  }
```

### 8. Long Signatures (Reformatting Strategy)

When signatures don't fit on one line, `sigfmt` applies intelligent packing to maximize readability while minimizing vertical space:

```diff
  func ComplexCalculation(
-     inputMatrix [][]float64,
-     weights []float64,
-     bias float64,
-     activationFunc func(float64) float64,
-     learningRate float64,
-     epochs int,
-     dropoutRate float64,
  ) error {
+ func ComplexCalculation(
+     inputMatrix [][]float64, weights []float64, bias float64,
+     activationFunc func(float64) float64, learningRate float64,
+     epochs int, dropoutRate float64,
+ ) error {
      // ...
  }
```

### 9. Local Structs in Functions

```diff
  func ComplexCase() {
      type LocalStruct struct {
-         Handler func(
-             ctx context.Context,
-             id string,
-         ) error
+         Handler func(ctx context.Context, id string) error

          Simple func() error

-         VeryLongHandler func(
-             parameterWithVeryLongName string,
-             anotherParameterWithVeryLongName string,
-             yetAnotherParameterWithVeryLongName string,
-         ) error
+         VeryLongHandler func(parameterWithVeryLongName string,
+             anotherParameterWithVeryLongName string,
+             yetAnotherParameterWithVeryLongName string) error
      }
      _ = LocalStruct{}
  }
```

## ⚖️ Comparison: sigfmt vs gofumpt vs golines vs wsl

sigfmt is a **focused** tool — it formats function signatures and nothing else.
The other tools cover broader or adjacent domains. Use this matrix to decide
when to reach for each.

| Feature | `sigfmt` | `gofumpt` | `golines` | `wsl` |
|---|---|---|---|---|
| **Domain** | Function signatures only | General Go formatting (strict gofmt) | Long-line shortening (any code) | Whitespace / cuddling rules |
| Collapse multi-line sigs that fit | ✅ | ❌ | ❌ | partial |
| Pack multiple params per line | ✅ | ❌ | ❌ | ❌ |
| Semantic parameter grouping (`param-groups`) | ✅ | ❌ | ❌ | ❌ |
| Aggressive interface/struct packing | ✅ | ❌ | ❌ | ❌ |
| Configurable line length (`max-line-len`) | ✅ | ❌ | ✅ | ❌ |
| Shortens non-signature long lines | ❌ | ❌ | ✅ | ❌ |
| Enforces block cuddling / blank lines | ❌ | ❌ | ❌ | ✅ |
| `golangci-lint` plugin with `--fix` | ✅ | ✅ | ✅ | ✅ |
| Suggested fixes (diagnostics + autofix) | ✅ | ✅ | ✅ (rewrites files) | ✅ |

### When to use sigfmt

- You want **consistent function signatures** — collapsed when they fit,
  semantically packed when they don't.
- You have **interface-heavy** or **struct-with-callbacks** code and want to
  minimize vertical bloat.
- You need **semantic grouping** (e.g. `context.Context` always paired with a
  transaction handle).

### When to combine sigfmt with other tools

- **sigfmt + gofumpt**: gofumpt for general formatting, sigfmt for signatures.
  They don't conflict.
- **sigfmt + golines**: golines to shorten non-signature long lines
  (struct literals, call chains), sigfmt to own signatures.
- **sigfmt + wsl**: wsl for block cuddling and blank-line rules, sigfmt for
  signature packing. wsl's single-line/multi-line heuristic for signatures is
  coarser than sigfmt's packing strategy — let sigfmt own that domain.

> **Bottom line:** sigfmt is complementary, not a replacement. It fills the gap
> that gofumpt, golines, and wsl leave open: intelligent, semantic formatting of
> Go function signatures. For full migration recipes and side-by-side examples,
> see the **[Configuration Cookbook](docs/cookbook.md)**.

## ⚡ Performance

- **Fast**: Does not require type loading (`register.LoadModeSyntax`).
- **Efficient**: Uses minimal memory (~20-30MB for 10k LOC) and works only with AST.
- **Parallel**: Supports parallel execution via `golangci-lint`.

## 🤔 FAQ

**Why not use `gofmt` or `gofumpt?**
They provide basic formatting but lack strict rules for line length and parameter packing. `sigfmt` complements them.

**Can parameter packing be disabled?**
Yes, use `pack-struct-fields: false` and `pack-interface-methods: false`.

**How to handle special cases?**
Use `//nolint:sigfmt` to ignore specific functions.

## 💻 Development

The project uses `Makefile` for common development tasks:

*   `make test`: Run all tests.
*   `make test-race`: Run tests with the race detector.
*   `make test-coverage`: Generate a test coverage report.
*   `make test-update-golden`: Update golden files (use when diagnostic messages or expected fixes change).
*   `make fmt`: Format Go code.
*   `make lint`: Run linters.
*   `make check`: Run all checks (test, fmt, lint).
*   `make build-example`: Build the example custom `golangci-lint` binary.
*   `make run-example`: Run the linter on the example project.
*   `make clean`: Clean up build artifacts.

### Updating Diagnostic Messages

When updating diagnostic messages or changing the expected output of the linter, follow these steps:
1.  Update constants in `linter.go` (or `internal/format/formatter.go` for messages).
2.  Run tests (`make test`). They will likely fail, indicating where the messages need to be updated.
3.  Update diagnostic messages in test files: `find testdata/src -name "*.go" -exec sed -i '' 's/OLD_MESSAGE/NEW_MESSAGE/g' {} \;` (replace `OLD_MESSAGE` and `NEW_MESSAGE` accordingly).
4.  Update golden files: `make test-update-golden`
5.  Verify all tests pass: `make test`

## ❤️ Community & Acknowledgements

This project is made with ❤️ for the open-source community by [Vadim Fedorenko](https://github.com/vsfedorenko).

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