# sigfmt Configuration Cookbook

Real-world `.golangci.yml` snippets and migration recipes for common Go
codebases. Each recipe shows the configuration, the before/after diff, and notes
on when to reach for it.

---

## Table of Contents

1.  [Quick Start — Sensible Defaults](#1-quick-start--sensible-defaults)
2.  [Service Layer with Many Parameters](#2-service-layer-with-many-parameters)
3.  [Interface-Heavy Codebase](#3-interface-heavy-codebase)
4.  [Struct with Callback Fields](#4-struct-with-callback-fields)
5.  [Migration from `golines`](#5-migration-from-golines)
6.  [Migration from `wsl`](#6-migration-from-wsl)
7.  [Configuration Reference](#7-configuration-reference)

---

## 1. Quick Start — Sensible Defaults

The defaults are tuned for the most common case: a 120-character limit with
aggressive packing for interfaces and structs, and conservative packing for
regular functions.

```yaml
# .golangci.yml
linters-settings:
  custom:
    sigfmt:
      type: "module"
      description: "Function signature formatter"
      settings:
        max-line-len: 120
        tab-width: 8
        pack-struct-fields: true
        pack-interface-methods: true
```

This is equivalent to omitting `settings` entirely — every key has a default.
Use this snippet as a starting point and adjust as needed.

---

## 2. Service Layer with Many Parameters

Service-layer functions often have a `context.Context` followed by business
parameters. With `param-groups` you can guarantee that the context always sits
on its own visual line (or is paired with a transaction handle), making diffs
smaller and signatures easier to scan.

```yaml
linters-settings:
  custom:
    sigfmt:
      type: "module"
      description: "Function signature formatter"
      settings:
        max-line-len: 120
        param-groups:
          - ["context.Context", "*sql.Tx"]  # ctx + tx always share a line
          - ["context.Context"]              # ctx alone on its own line
```

**Before** (manual formatting, inconsistent):

```go
func CreateOrder(
    ctx context.Context,
    tx *sql.Tx,
    orderID string,
    items []Item,
    shippingAddress *Address,
    paymentMethod PaymentMethod,
    options ...Option,
) (*Order, error) {
```

**After** (sigfmt with `param-groups`):

```go
func CreateOrder(
    ctx context.Context, tx *sql.Tx,
    orderID string, items []Item, shippingAddress *Address,
    paymentMethod PaymentMethod, options ...Option,
) (*Order, error) {
```

> **Tip:** `param-groups` matches on the rendered type string, so
> `"context.Context"` matches the imported type. For package-qualified types
> include the qualifier, e.g. `"*sql.Tx"`, `"time.Duration"`.

---

## 3. Interface-Heavy Codebase

Projects that lean on interfaces (DDD, hexagonal architecture, mock-heavy
testing) accumulate many method signatures. Aggressive packing collapses them
so a 30-method interface stays readable.

```yaml
linters-settings:
  custom:
    sigfmt:
      type: "module"
      description: "Function signature formatter"
      settings:
        max-line-len: 120
        pack-interface-methods: true   # multiple params per line
```

**Before:**

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
}
```

**After:**

```go
type UserRepository interface {
    Create(ctx context.Context, name string, email string) (*User, error)
    Update(ctx context.Context, id int, name string, email string) error
}
```

A 24-line interface becomes 4 lines. If you prefer one parameter per line
(traditional Go style), set `pack-interface-methods: false` and sigfmt will only
collapse signatures that fit on a single line.

---

## 4. Struct with Callback Fields

Event-driven code often has structs full of `func` fields — callbacks, hooks,
and handlers. `pack-struct-fields` collapses short ones and packs long ones.

```yaml
linters-settings:
  custom:
    sigfmt:
      type: "module"
      description: "Function signature formatter"
      settings:
        max-line-len: 120
        pack-struct-fields: true
```

**Before:**

```go
type Handler struct {
    OnStart func(
        id string,
    ) error

    OnProcess func(
        ctx context.Context,
        data []byte,
        meta map[string]string,
    ) error

    OnProcessWithVeryLongNameAndManyParameters func(
        parameterOne string,
        parameterTwo string,
        parameterThree string,
        parameterFour string,
    ) error
}
```

**After:**

```go
type Handler struct {
    OnStart func(id string) error

    OnProcess func(ctx context.Context, data []byte, meta map[string]string) error

    OnProcessWithVeryLongNameAndManyParameters func(parameterOne string,
        parameterTwo string, parameterThree string, parameterFour string) error
}
```

---

## 5. Migration from `golines`

[`golines`](https://github.com/segmentio/golines) is a general-purpose long-line
shortener. `sigfmt` is purpose-built for function signatures and adds semantic
grouping, so the two tools produce different output for the same input.

### What `golines` does

`golines` wraps any long line at a column limit — it does not understand
parameter semantics, interfaces, or struct fields. It will break a function
signature wherever it hits the limit, which can split a logical parameter group.

### What `sigfmt` does

`sigfmt` understands Go signatures and applies two strategies:

1.  **Collapse** — if the whole signature fits, put it on one line.
2.  **Pack** — if it does not fit, pack parameters semantically, respecting
    `param-groups`.

### Side-by-side comparison

**Input (what both tools receive):**

```go
func ProcessOrder(ctx context.Context, orderID string, items []Item, shippingAddress *Address, paymentMethod PaymentMethod, options ...Option) (*Order, error) {
```

**`golines` output** (wraps at the column limit, no semantic awareness):

```go
func ProcessOrder(ctx context.Context, orderID string, items []Item,
    shippingAddress *Address, paymentMethod PaymentMethod,
    options ...Option) (*Order, error) {
```

**`sigfmt` output** (with `param-groups: [["context.Context", "*sql.Tx"]]` —

  not active here, but demonstrating semantic grouping logic):

```go
func ProcessOrder(
    ctx context.Context, orderID string, items []Item,
    shippingAddress *Address, paymentMethod PaymentMethod, options ...Option,
) (*Order, error) {
```

### Feature comparison

| Feature | `golines` | `sigfmt` |
|---|---|---|
| Shortens long lines (any code) | ✅ | ❌ (signatures only) |
| Collapses multi-line signatures that fit | ❌ | ✅ |
| Semantic parameter grouping (`param-groups`) | ❌ | ✅ |
| Interface method packing | ❌ | ✅ |
| Struct field packing | ❌ | ✅ |
| `golangci-lint` integration (diagnostics + `--fix`) | ❌ (standalone) | ✅ |
| Runs in CI as a linter gate | ❌ | ✅ |

### How to migrate

1.  **Replace** the `golines` step in your formatting pipeline with `sigfmt`
    as a `golangci-lint` custom linter (see [Quick Start](#1-quick-start--sensible-defaults)).
2.  **Keep `golines`** if you also need to shorten non-signature long lines
    (struct literals, long chains, etc.). The two tools are complementary —
    `golines` handles general line wrapping, `sigfmt` handles signatures.
3.  **Run `sigfmt --fix`** once to reformat your existing signatures, then commit
    the result. Subsequent runs only flag new violations.
4.  **Configure `param-groups`** to encode your project's conventions (e.g.
    always pair `context.Context` with a transaction handle).

---

## 6. Migration from `wsl`

[`wsl`](https://github.com/bombsimon/wsl) (Whitespace Linter) enforces
whitespace rules: cuddled blocks, empty lines, and whether a multi-line
expression should be on one line. It overlaps with `sigfmt` only in the narrow
case of "should this be multi-line or single-line?" — and `sigfmt` goes further
on signatures.

### What `wsl` covers

`wsl` focuses on *whitespace* — blank lines between statements, cuddled
`if`/`for` blocks, and whether append/call expressions should be collapsed.
It does **not** pack parameters within a multi-line signature.

### What `sigfmt` covers that `wsl` does not

| Feature | `wsl` | `sigfmt` |
|---|---|---|
| Collapse multi-line signatures that fit | partial (some cases) | ✅ |
| Pack multiple parameters per line (interfaces, structs) | ❌ | ✅ |
| Semantic parameter grouping (`param-groups`) | ❌ | ✅ |
| Aggressive interface/struct packing | ❌ | ✅ |
| Long-signature reformatting strategy | ❌ | ✅ |
| Whitespace / cuddling rules (blocks, blank lines) | ✅ | ❌ |

### Practical guidance

- **Use both.** `wsl` handles general whitespace (cuddled blocks, blank lines);
  `sigfmt` handles signature-specific formatting. They do not conflict.
- **Remove `wsl`'s signature-collapsing behavior** (if it's producing noise) and
  let `sigfmt` own that domain. `sigfmt`'s packing strategy is more
  sophisticated than `wsl`'s binary single-line/multi-line choice.
- `sigfmt` provides **suggested fixes** (`--fix`), so you can batch-apply its
  formatting. `wsl` also supports autofix, so both can run in the same
  `golangci-lint --fix` pass.

---

## 7. Configuration Reference

| Setting | Type | Default | Description |
|---|---|---|---|
| `max-line-len` | int | `120` | Maximum visual line length (including indentation) before wrapping is required. |
| `tab-width` | int | `8` | Visual width of a tab character for length calculations. |
| `pack-struct-fields` | bool | `true` | Aggressively pack function-type fields in structs (multiple params per line). |
| `pack-interface-methods` | bool | `true` | Aggressively pack method signatures in interfaces. |
| `param-groups` | list of lists | `[]` | Type-name groups kept together on the same line. Each group is a list of rendered type strings (e.g. `["context.Context", "*sql.Tx"]`). |

### Standalone CLI flags

When running `sigfmt` directly (not via `golangci-lint`), the same settings are
available as flags:

```bash
sigfmt -max-line-len 100 -tab-width 4 -pack-struct-fields=false ./...
sigfmt -fix ./...
```

| Flag | Equivalent setting |
|---|---|
| `-max-line-len <n>` | `max-line-len` |
| `-tab-width <n>` | `tab-width` |
| `-pack-struct-fields` | `pack-struct-fields` |
| `-pack-interface-methods` | `pack-interface-methods` |

> **Note:** `param-groups` is only available via the `golangci-lint` plugin
> configuration; the standalone CLI does not currently expose it.
