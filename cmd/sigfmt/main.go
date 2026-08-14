// Package main implements the sigfmt standalone CLI.
//
// sigfmt is a golangci-lint plugin for checking and formatting Go function
// signatures. This binary lets you run the same checks directly from the
// command line without golangci-lint, and can automatically apply suggested
// fixes with the -fix flag.
//
// Usage:
//
//	sigfmt [flags] [packages]
//
// The CLI is built on golang.org/x/tools/go/analysis/singlechecker, which
// provides standard flags including -fix, -V (version), and package pattern
// arguments (e.g. ./..., ./pkg/foo, file.go).
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/vsfedorenko/sigfmt"
)

func main() {
	singlechecker.Main(sigfmt.NewAnalyzer())
}
