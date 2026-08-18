# Example Linter Integration

This directory demonstrates how to use `sigfmt` as a **module plugin** for
`golangci-lint` **v2** (the `plugin-module-register` API).

## Prerequisites

- Go 1.25+
- [golangci-lint v2](https://golangci-lint.run/usage/install/)

## Setup

1. Build the custom `golangci-lint` binary that includes the plugin:

   ```bash
   golangci-lint custom
   ```

   This command reads `.custom-gcl.yml`, compiles the plugin from the parent
   directory (`path: ../`), and produces a `custom-gcl` binary in the current
   directory.

   On Linux with newer binutils (gold linker removed), use:

   ```bash
   CGO_ENABLED=0 golangci-lint custom
   ```

## Usage

Run the custom linter on the code:

```bash
./custom-gcl run
```

You should see output similar to (verified against golangci-lint v2.12.2):

```text
main.go:13:1: Signature can be formatted more compactly (sigfmt)
) {
^
...
3 issues:
* sigfmt: 3
```

## Configuration

The linter is configured in `.golangci.yml` (golangci-lint v2 format:
top-level `version: "2"`, settings under `linters.settings.custom.sigfmt`).
You can adjust `max-line-len` (default 120) to control when signatures
should be collapsed.

See the root [README.md](../README.md#️-configuration) for the full
parameter reference and a verified example config.
