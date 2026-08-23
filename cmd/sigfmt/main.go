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
// provides standard flags including -fix, -diff, -V (version), and package
// pattern arguments (e.g. ./..., ./pkg/foo, file.go).
//
// Driver quirk handled here: in the upstream singlechecker, -diff only takes
// effect when combined with -fix — a lone -diff silently degrades to a plain
// check run. This entry point promotes a lone -diff to -fix -diff, so
// `sigfmt -diff ./...` previews all proposed changes as a unified diff
// without writing any files (the `gofmt -d` behavior users expect).
package main

import (
	"os"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/vsfedorenko/sigfmt"
)

func main() {
	os.Args = promoteLoneDiff(os.Args)
	singlechecker.Main(sigfmt.NewAnalyzer())
}

// valueFlags are the driver and analyzer flags that consume the next token
// as their value (Go's flag package: non-boolean flags accept "-flag value").
// Every other flag is boolean, so it never swallows the following token.
// When the driver grows a new value flag, extend this set.
var valueFlags = map[string]bool{
	// analyzer flags
	"max-line-len": true,
	"tab-width":    true,
	"param-groups": true,
	// singlechecker/checker driver flags
	"cpuprofile": true,
	"memprofile": true,
	"trace":      true,
	"debug":      true,
	"c":          true,
	// legacy vet shims
	"tags": true,
}

// promoteLoneDiff inserts "-fix" before the first -diff token when -diff is
// enabled without -fix, mirroring flag.Parse semantics: the flag section
// ends at "--", a lone "-", or the first non-flag argument.
func promoteLoneDiff(argv []string) []string {
	if len(argv) == 0 {
		return argv
	}
	hasFix := false
	diffIdx := -1
	args := argv[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" || arg == "-" || !strings.HasPrefix(arg, "-") {
			break
		}
		// Go's flag package accepts one or two leading dashes ("-diff",
		// "--diff"); strip either form before cutting off "=value".
		body := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		name, value, hasValue := strings.Cut(body, "=")
		switch name {
		case "fix":
			hasFix = !hasValue || value == "true"
		case "diff":
			if diffIdx < 0 && (!hasValue || value == "true") {
				diffIdx = i + 1
			}
		}
		if !hasValue && valueFlags[name] && i+1 < len(args) {
			i++ // skip the flag's value token
		}
	}
	if hasFix || diffIdx < 0 {
		return argv
	}
	return slices.Insert(argv, diffIdx, "-fix")
}
