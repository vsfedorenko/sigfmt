# Changelog

All notable changes to **sigfmt** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `.pre-commit-hooks.yaml` — official [pre-commit](https://pre-commit.com/)
  hooks `sigfmt` (check) and `sigfmt-fix` (manual in-place reformat) for the
  standalone CLI; documented in README.
- "Editor integration" README section — VS Code (Run on Save) and Neovim
  (`BufWritePre` autocmd) format-on-save recipes for the standalone CLI, plus
  the `-diff` flag in the CLI reference table.
- CHANGELOG.md — version history in Keep a Changelog format.
- `docs/cookbook.md` — real-world configuration cookbook with side-by-side
  migration examples for `golines` and `wsl` users.
- "Comparison" section in README — decision matrix: `sigfmt` vs `gofumpt` vs
  `golines` vs `wsl`.

## [1.0.0] — 2025-12-07

### Added
- **Collapse stage**: multi-line signatures that fit within `max-line-len`
  (default `120`) are collapsed to a single line.
- **Reformat / packing stage**: long signatures that do not fit on one line are
  packed with context-aware strategies:
  - *Regular functions* — conservative packing that preserves logical parameter
    grouping.
  - *Interfaces* — aggressive packing (`pack-interface-methods`, default `on`)
    that places multiple parameters per line.
  - *Structs* — aggressive packing (`pack-struct-fields`, default `on`) for
    function-type fields.
- **Generics (Go 1.18+) support**: correct handling of type parameters
  `[T any]` in width calculations.
- **Anonymous functions / closures**: collapsing of multi-line `func` literals,
  including inline callbacks (e.g. `group.Go(func(ctx) error { ... })`).
- **`param-groups` configuration**: keep semantically related parameter types on
  the same line (e.g. always pair `context.Context` with `*sql.Tx`).
- **Standalone CLI** (`cmd/sigfmt`) for running the linter without
  `golangci-lint`, with the same analysis logic and CLI flags
  (`-max-line-len`, `-tab-width`, `-pack-struct-fields`,
  `-pack-interface-methods`).
- AST-based, syntax-only analysis — fast (`register.LoadModeSyntax`), no type
  loading required.
- Suggested fixes (`--fix`) with a single unified diagnostic message:
  *"Signature can be formatted more compactly"*.
- `//nolint:sigfmt` directive support.
- GitHub Actions CI (`ci.yml`), release workflow (`release.yml`), and
  Dependabot configuration.
- Codecov integration.
- Issue templates and community health files.
- MIT License.
- Comprehensive README with examples gallery, FAQ, and performance notes.

### Changed
- Simplified diagnostic messages to a single unified message across all
  signature kinds (functions, methods, interfaces, structs).

## [0.2.0] — 2025-12-07

### Added
- Settings system (`internal/config`) with `max-line-len`, `tab-width`,
  `pack-struct-fields`, `pack-interface-methods`, and `param-groups`.
- Makefile for common development tasks (`test`, `test-race`, `test-coverage`,
  `test-update-golden`, `fmt`, `lint`, `build-example`, `run-example`).
- Godoc comments for public package API.

### Changed
- Translations migrated to English.
- README refactored for clarity and structure.

## [0.1.0] — 2025-12-06

### Added
- Initial release: `sigfmt` golangci-lint plugin for function signature
  formatting.
- Project structure: `linter.go`, `internal/config`, `internal/format`,
  `internal/astinfo`, `internal/domain`, `internal/render`.
- `analysistest`-based test suite with `.golden` files under
  `testdata/src/{features,limits,settings}`.
- Basic function/method declaration collapsing.
- Interface method and struct field support.

[Unreleased]: https://github.com/vsfedorenko/sigfmt/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/vsfedorenko/sigfmt/releases/tag/v1.0.0
[0.2.0]: https://github.com/vsfedorenko/sigfmt/releases/tag/v0.2.0
[0.1.0]: https://github.com/vsfedorenko/sigfmt/releases/tag/v0.1.0
