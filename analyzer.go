package sigfmt

import (
	"flag"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/vsfedorenko/sigfmt/internal/config"
)

// NewAnalyzer returns an analysis.Analyzer configured with CLI flags for all
// sigfmt settings. It reuses the same analysis logic (PluginLineWrap.run) as
// the golangci-lint plugin, so the standalone CLI and the plugin always behave
// identically.
//
// The returned analyzer is designed for use with
// golang.org/x/tools/go/analysis/singlechecker.Main, which provides the -fix
// flag, package/file argument handling, diagnostic output, and exit codes.
// The sigfmt-specific flags (-max-line-len, -tab-width, etc.) are exposed via
// Analyzer.Flags and parsed by the analysis driver.
func NewAnalyzer() *analysis.Analyzer {
	var (
		maxLineLen           int
		tabWidth             int
		packStructFields     bool
		packInterfaceMethods bool
		paramGroupsStr       string
	)

	flags := flag.NewFlagSet(analyzerName, flag.ExitOnError)
	flags.IntVar(&maxLineLen, "max-line-len", config.DefaultMaxLineLen,
		"maximum line length before multi-line signatures are required")
	flags.IntVar(&tabWidth, "tab-width", config.DefaultTabWidth,
		"visual width of a tab character for length calculations")
	flags.BoolVar(&packStructFields, "pack-struct-fields", config.DefaultPackStructFields,
		"aggressively pack function-type struct fields")
	flags.BoolVar(&packInterfaceMethods, "pack-interface-methods", config.DefaultPackInterfaceMethods,
		"aggressively pack method signatures in interfaces")
	flags.StringVar(&paramGroupsStr, "param-groups", "",
		"semicolon-separated parameter type groups; each group is a comma-separated list of type names (e.g. 'context.Context,error;io.Reader,io.Writer')")

	return &analysis.Analyzer{
		Name:  analyzerName,
		Doc:   analyzerDoc,
		Flags: *flags,
		Run: func(pass *analysis.Pass) (any, error) {
			settings := config.Settings{
				MaxLineLen:           maxLineLen,
				TabWidth:             tabWidth,
				PackStructFields:     packStructFields,
				PackInterfaceMethods: packInterfaceMethods,
				ParamGroups:          parseParamGroupsFlag(paramGroupsStr),
			}
			plugin := &PluginLineWrap{settings: settings}
			return plugin.run(pass)
		},
	}
}

// parseParamGroupsFlag parses a CLI string like
// "context.Context,error;io.Reader,io.Writer" into [][]string.
// An empty string produces nil (no groups).
func parseParamGroupsFlag(s string) [][]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var result [][]string
	for _, group := range strings.Split(s, ";") {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		var types []string
		for _, t := range strings.Split(group, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				types = append(types, t)
			}
		}
		if len(types) > 0 {
			result = append(result, types)
		}
	}
	return result
}
