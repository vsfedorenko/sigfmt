# Contributing to sigfmt

Thank you for your interest in contributing to **sigfmt**! This document covers
the basics of getting set up and the expectations for contributions.

## Getting Started

1. **Fork** the repository and clone your fork.
2. Ensure you have **Go 1.25+** installed ([downloads](https://go.dev/dl/)).
3. Optionally install [`golangci-lint`](https://golangci-lint.run/usage/install/).
4. Create a feature branch: `git checkout -b my-feature`.

## Development Setup

The full project context for AI assistants and contributors is documented in
[`CLAUDE.md`](./CLAUDE.md). It covers the architecture, core logic, and testing
workflow. Read it first if you plan to change the linter logic.

## Building & Testing

Common tasks are provided via the `Makefile`:

| Command                     | Description                              |
|-----------------------------|------------------------------------------|
| `make test`                 | Run all tests                            |
| `make test-race`            | Run tests with the race detector         |
| `make test-coverage`        | Generate an HTML coverage report         |
| `make test-update-golden`   | Regenerate `.golden` expected files      |
| `make test-clean`           | Run tests without cache                  |
| `make fmt`                  | Format code (`gofmt -s`)                 |
| `make lint`                 | Run `golangci-lint`                      |
| `make vet`                  | Run `go vet`                             |
| `make check`                | Run fmt + vet + lint + test              |

## Golden Files Workflow

sigfmt uses the [`analysistest`](https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest)
framework with `.golden` files to verify diagnostic output. Tests live under
`testdata/src/{features,limits,settings}`.

When you change linter behavior:

1. Modify the relevant `testdata/.../*.go` files and their `// want "..."`
   expectations.
2. Regenerate expected output:

   ```bash
   make test-update-golden
   ```

3. Review the diff carefully — `.golden` changes should match your intent, not
   just make the tests pass.
4. Run the full suite to confirm:

   ```bash
   make test
   ```

## Code Style

- Run `gofmt -s -w .` (or `make fmt`) before committing.
- Keep comments in English.
- Use constants for user-facing strings (see `linter.go`).
- Minimize allocations; the linter should stay fast.

## Pull Requests

1. Ensure `make test` passes locally.
2. Update `.golden` files if you changed linter output (see above).
3. Verify `gofmt -s -l .` produces no output (code is formatted).
4. Add tests for new behavior.
5. Keep commits focused; use clear commit messages.

See the pull request template for the full checklist.

## Reporting Issues

Use the [issue templates](https://github.com/vsfedorenko/sigfmt/issues/new/choose)
to file bug reports or feature requests. For security vulnerabilities, see
[SECURITY.md](./SECURITY.md) — **do not open a public issue**.

## Code of Conduct

By participating in this project you agree to abide by the
[Code of Conduct](./CODE_OF_CONDUCT.md).
