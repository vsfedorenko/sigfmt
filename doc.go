// Package sigfmt checks that multi-line function signatures are formatted
// as compactly as the configured line limit allows.
//
// Two behaviors, one diagnostic:
//
//   - Collapse: a multi-line signature that fits within max-line-len on a
//     single line is collapsed to one line.
//   - Pack: a signature that cannot fit on one line is reformatted with
//     multiple parameters per line (interfaces and struct fields are packed
//     aggressively; function declarations preserve parameter grouping).
//
// Every finding carries a suggested fix, so drivers that support -fix
// (the standalone CLI, golangci-lint --fix) rewrite the signature in place.
// Comments inside a signature are never rewritten: commented signatures are
// reported without a fix rather than losing the comment text.
//
// The package integrates two ways.
//
// # golangci-lint plugin
//
// sigfmt registers itself as a golangci-lint custom plugin via
// github.com/golangci/plugin-module-register. Configure it in .golangci.yml:
//
//	plugins:
//	  - module: github.com/vsfedorenko/sigfmt
//
//	linters:
//	  settings:
//	    sigfmt:
//	      max-line-len: 120
//
// Recognized settings (all optional; defaults in parentheses):
//
//	max-line-len (120)         maximum line length before multi-line is required
//	tab-width (8)              visual width of a tab in length calculations
//	pack-struct-fields (true)  aggressively pack function-type struct fields
//	pack-interface-methods (true)
//	                           aggressively pack method signatures in interfaces
//	param-groups (none)        type groups kept on one line when packing;
//	                           e.g. [["context.Context","error"],["io.Reader","io.Writer"]]
//	ignore-tests (false)       skip _test.go files entirely
//
// # Standalone analyzer
//
// NewAnalyzer returns a standard analysis.Analyzer with the same settings
// exposed as CLI flags. It is designed for use with
// golang.org/x/tools/go/analysis/singlechecker.Main:
//
//	package main
//
//	import (
//		"golang.org/x/tools/go/analysis/singlechecker"
//
//		"github.com/vsfedorenko/sigfmt"
//	)
//
//	func main() { singlechecker.Main(sigfmt.NewAnalyzer()) }
//
// The standalone CLI and the golangci-lint plugin share one analysis
// implementation and always report the same findings.
package sigfmt
