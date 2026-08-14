package sigfmt

import (
	"flag"

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
			}
			plugin := &PluginLineWrap{settings: settings}
			return plugin.run(pass)
		},
	}
}
