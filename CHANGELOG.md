# Changelog

All notable changes to **sigfmt** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.5.0] — 2026-08-23

### Added
- Standalone `sigfmt -diff ./...` dry-run mode: a lone `-diff` is promoted
  to `-fix -diff` by the CLI entry point, so it previews all proposed
  changes as a unified diff without writing files and exits 0 (the
  `gofmt -d` behavior). Previously `-diff` without `-fix` silently
  degraded to a plain check run — the upstream singlechecker only honors
  `-diff` in combination with `-fix`.

## [1.4.2] — 2026-08-22

### Changed
- Renderer fast path for plain type expressions (identifiers, qualified
  names, pointers, slices, maps, channels, ellipses): rendered by string
  construction instead of go/printer. Profiling attributed ~68% of
  analyzer allocations to printer.Fprint's tabwriter machinery for types
  that print identically either way. Benchmarks: violations corpus
  8.25ms → 4.0ms/op and 53.9k → 20.7k allocs/op (-62%); clean corpus
  -68% allocs. Against the gofmt baseline the analyzer drops from 2.2×
  slower to ~1.2×. Golden files byte-identical; full suite green.

## [1.4.1] — 2026-08-21

### Fixed
- `make test-update-golden` corrupted goldens for files analyzed in more
  than one pass (a package and its external test variant): identical
  suggested-fix edits were applied twice — the second application ran on
  rewritten bytes with stale offsets (`") error { // want"` became
  `") error/ want"`). Edits are now deduplicated by
  (path, pos, end, newText).

### Added
- Struct-tag budget edge case pinned: a one-line func field whose
  signature alone exceeds max-line-len (unlike a small signature
  outweighed by its tag) must be unpacked with the tag budgeted into the
  closing-paren line.

### Changed
- OSSF scorecard on main pushes (supply-chain hygiene metrics in the
  Security tab); scorecard-action pinned after the upstream moving `v2`
  tag was unpublished.

## [1.4.0] — 2026-08-20

### Added
- Struct-tag awareness for func-typed struct fields: a field's tag
  (`json:"…"`, `validate:"…"`, …) is never rewritten, but its width now
  counts against `max-line-len` — a signature collapses only when
  signature + tag fit on one line, and the packing path budgets results
  + tag into the last line (closing `)` moves to the parent indent when
  needed). Tagged one-liners whose overflow is the tag's doing are left
  alone instead of being split uselessly. Black-box corpus
  (`TestStructTagLineBudget`) pins the invariants: no new overflow, tag
  survival, whitespace-only gofmt disagreement, idempotence.

### Security
- CI security suite (`.github/workflows/security.yml`): govulncheck on
  every push/PR plus a daily cron, CodeQL analysis (Go), gitleaks over
  the full history, dependency review on PRs (fails on high/critical).
- `go` directive bumped 1.25.0 → 1.25.13: govulncheck reported 4
  reachable stdlib vulnerabilities — fixed by the toolchain patch.

## [1.3.0] — 2026-08-20

### Added
- Homebrew install: the repository doubles as a tap (`brew tap
  vsfedorenko/sigfmt <repo-url>`), with the formula auto-regenerated
  from release checksums at publish time.
- testify + mockery testing infrastructure (default configs): generated
  mock for the `format.Strategy` port; formatter orchestrator tests
  (first-applied-wins, no-strategy, nil-params short-circuit).

### Changed
- All test assertions migrated to testify exclusively: `assert`/`require`
  (want-first), mocks driven via `EXPECT()` only — no manual `if+Fatal`
  comparisons remain.
- golangci-lint: the empty config replaced with a full tuned suite
  (staticcheck, govet, gosec, revive, goconst, errcheck, errorlint, …
  plus gci/gofumpt formatters). Real findings fixed; zero issues on the
  expanded suite.
- `.mockery.yml` minimized to the `packages` selection — defaults are
  owned by the tool, not mirrored (regeneration byte-identical).

## [1.2.1] - 2026-08-19

### Fixed
- Multi-name func-typed struct fields (`Handler, Fallback func(...)`) no
  longer lose all names but the first when rewritten: the fix used to
  silently delete struct fields. Found by hand-probing the released CLI
  (`A, B func(a string) error` became `A func(...)`); pinned by a
  black-box corpus (collapse, packing, tag, three-name cases) plus an
  analysistest fixture — every declared name must survive the full fix
  cycle, idempotence included. (#36)

### Added
- Plugin-path e2e suite (`example/gcl_e2e_test.go`): builds the custom
  golangci-lint binary (`golangci-lint custom`, the same command as
  `make build-example`) and runs it against a throwaway third-party
  module. Pins the integration contracts found by hand-probing:
  the linter activates only when the target config declares it under
  `linters.settings.custom` (bare `enable:` yields "unknown linters" —
  golangci v2 behavior worth documenting in the test);
  `--fix` applies in place and a re-run is clean (idempotence through
  golangci); comment preservation (#29) holds through the plugin path;
  a 30-file bulk package reports every diagnostic once golangci's
  default caps (`max-same-issues=3`) are lifted. CI runs the suite in
  the Build Example job.

### Added
- Glitch corpus (8 hostile signatures): extreme parameter counts, nested
  generics with unions, generic receivers, pointer-of-pointer types,
  variadic map/slice/func chains, curried returns, struct fields and
  interface methods. For every entry the full fix cycle asserts three
  invariants — the applied output parses, is a gofmt fixed point, and a
  second pass is a no-op (idempotence) — plus a `go vet` type-check of
  the fixed output (a semantic-breaking rewrite would parse fine but
  fail the build; this catches it).

### Added
- CLI smoke test suite (`cmd/sigfmt/cli_smoke_test.go`, 7 tests): the
  released binary surface exercised end-to-end — clean package exits 0,
  violations exit non-zero with the diagnostic, `-fix -diff` is a
  preview mode (prints the patch, leaves files untouched, exits 0),
  `-fix` rewrites in place and the re-check passes, comment-preservation
  (#29) holds through the CLI (byte-identical file after `-fix`),
  unknown flags fail loudly, and `-max-line-len` reaches the analyzer
  through singlechecker (tight limit disables the collapse diagnostic).

### Added
- **Release binaries via GoReleaser.** Tags now build cross-platform
  archives (`linux`/`darwin`/`windows` × `amd64`/`arm64`, tar.gz — zip for
  windows) with `checksums.txt` attached to the GitHub Release. The release
  workflow also runs the test suite before building. First binary-bearing
  release will be the next tag (previous releases shipped source only).
  Foundation for the planned Homebrew tap / aqua entry.

### Fixed
- **`//line` directives no longer crash the analyzer.** `GetIndent` called
  `File.LineStart(File.Line(pos))`, which panics with `invalid line number`
  when a `//line` (or `/*line*/`) directive maps a position beyond the
  physical end of the file — the token line table is sparse under such
  directives. The indent is now resolved by a backward scan to the start of
  the physical line, which is directive-agnostic. Formatting still works on
  such files: collapsible signatures after a `//line` directive are collapsed,
  the directive text survives the fix, and a second run is a no-op
  (`TestLineDirectivesDoNotCrash`, `TestLineDirectiveCollapseAndStability`).
- **`//go:build ignore` files are skipped in file-argument mode.** Package
  patterns (`sigfmt ./...`) never load build-excluded files, but passing a
  file directly (`sigfmt gen.go`) bypasses the package loader and previously
  produced diagnostics on `go run` scripts. The analyzer now evaluates the
  file's build constraints with the toolchain's default context (GOOS,
  GOARCH, compiler, cgo, release tags) and skips files excluded from the
  current build — the same tolerance `go vet` has. `//go:build linux`
  remains lintable on Linux (`TestBuildIgnoredFileSkipped`).

### Changed
- **Documentation: golangci-lint v2 plugin path end-to-end.** The README
  previously showed the v1-era build (`version: v1.64.6`, manual blank-import)
  and a v1-format `.golangci.yml` (`linters-settings:` at top level) that
  golangci-lint v2 rejects with `unsupported version of the configuration`.
  Both README and docs/cookbook.md now document the v2 module-plugin path:
  `.custom-gcl.yml` from a released tag or a local `path:`, a verified
  `version: "2"` config example, and a tested-versions matrix
  (v2.7.1, v2.12.2 — proxy and local builds, `--fix`, settings decoding all
  exercised). Added a build note for Linux binutils ≥ 2.44 systems where the
  gold linker was removed (`CGO_ENABLED=0 golangci-lint custom`). Example
  directory bumped to v2.12.2 with corrected expected output; stale
  pre-commit `rev` in README updated to the latest tag.
- **Comment preservation (zero loss)**: signatures containing comments
  inside the rewritten range are now left untouched instead of being
  collapsed with the comments silently dropped. The renderer rebuilds
  signatures from the AST, so any `//` or `/* */` comment between the
  signature start and end would be lost by a rewrite; the analyzer now
  skips such signatures entirely. Doc comments above a signature are
  outside the rewrite range and unaffected. Enforced by the black-box
  `TestCommentPreservationZeroLoss` corpus test: applying all suggested
  fixes must lose zero comments and keep the output parseable. Existing
  comment fixtures updated to the new semantics.

### Added
- Performance benchmark suite (`make bench`): black-box benchmarks through
  the public analyzer API on a deterministic generated corpus (12 files × 40
  signatures, four signature shapes), with two profiles — `violations`
  (worst case: every signature needs formatting) and `clean` (incremental
  rerun of a formatted codebase). `BenchmarkGofmtBaseline` runs the exact
  `go/format.Source` call the gofmt binary makes over the same corpus, so
  the ratio is apples-to-apples. Corpus files are guarded to be gofmt fixed
  points. Measured on Go 1.25/linux-arm64: sigfmt with parsing is ~3.0×
  gofmt on violations and ~2.2× on clean code; analysis alone is ~7.2 ms /
  480 signatures (~15µs per signature). The <2× target remains open —
  CPU profiles point at allocation pressure in extraction/rendering.
- `gofmt -s` compatibility contract: `TestGoldenFilesAreGofmtSFixedPoints`
  verifies every golden file (the exact bytes sigfmt's suggested fixes
  produce) is a `gofmt -s` fixed point, checked against the real gofmt
  binary of the running toolchain. New `gofmt_compat` fixture covers
  simplified syntax around rewritten signatures (elided composite literal
  types, range over int).
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
