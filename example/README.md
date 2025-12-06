# Example Linter Integration

This directory demonstrates how to use the `funcwrap-linter` as a module plugin for `golangci-lint`.

## Prerequisites

- Go 1.23+
- [golangci-lint](https://golangci-lint.run/usage/install/)

## Setup

1. Build the custom `golangci-lint` binary that includes the plugin:

```bash
golangci-lint custom
```

This command will read `.custom-gcl.yml`, compile the plugin from the parent directory (`../`), and produce a `custom-gcl` binary in the current directory.

## Usage

Run the custom linter on the code:

```bash
./custom-gcl run
```

You should see output similar to:

```text
main.go:10:1: Signature fits in one line (funcwrap)
func CheckSignature(
^
```

## Configuration

The linter is configured in `.golangci.yml` under `linters.settings.custom.funcwrap`.
You can adjust `max-line-len` (default 120) to control when signatures should be collapsed.
