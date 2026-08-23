package sigfmt_test

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/analysis"

	"github.com/vsfedorenko/sigfmt"
)

// ExampleNew shows a golangci-lint configuration: the settings map mirrors
// the linters.settings.sigfmt section of .golangci.yml. Unknown or omitted
// keys fall back to the defaults.
func ExampleNew() {
	plugin, err := sigfmt.New(map[string]any{
		"max-line-len": 100,
		"tab-width":    4,
		"param-groups": [][]string{
			{"context.Context", "error"},
		},
	})
	if err != nil {
		panic(err)
	}
	analyzers, err := plugin.BuildAnalyzers()
	if err != nil {
		panic(err)
	}
	for _, a := range analyzers {
		fmt.Println(a.Name, "-", a.Doc)
	}
	// Output:
	// sigfmt - Checks if multi-line function signatures can be collapsed to one line or reformatted more compactly
}

// ExampleNewAnalyzer walks the CLI flags the standalone analyzer exposes.
// The same settings exist as map keys in the golangci-lint plugin mode.
func ExampleNewAnalyzer() {
	a := sigfmt.NewAnalyzer()

	var flags []string
	a.Flags.VisitAll(func(f *flag.Flag) {
		flags = append(flags, fmt.Sprintf("%s=%v", f.Name, f.DefValue))
	})
	sort.Strings(flags)
	fmt.Println(a.Name)
	for _, f := range flags {
		fmt.Println(" ", f)
	}
	// Output:
	// sigfmt
	//   ignore-tests=false
	//   max-line-len=120
	//   pack-interface-methods=true
	//   pack-struct-fields=true
	//   param-groups=
	//   tab-width=8
}

// Example_analyzerRun runs one real analysis pass over a fixture with three
// signature kinds (interface method, struct field, func decl) and prints the
// diagnostics, then the fixture with every suggested fix applied. This is
// what a driver sees when it loads sigfmt: the standalone CLI, golangci-lint,
// or any other go/analysis consumer.
func Example_analyzerRun() {
	src := `package demo

type Handler interface {
	Process(
		request *Request,
	) (*Response, error)
}

type Server struct {
	OnEvent func(
		event string,
	) error
}

func Serve(
	addr string,
	handler Handler,
) error {
	return nil
}
`
	dir, err := os.MkdirTemp("", "sigfmt-example")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		panic(err)
	}

	a := sigfmt.NewAnalyzer()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	type edit struct {
		start, end int
		newText    string
	}
	var edits []edit
	pass := &analysis.Pass{
		Analyzer: a,
		Fset:     fset,
		Files:    []*ast.File{f},
		ReadFile: func(string) ([]byte, error) { return []byte(src), nil },
		Report: func(d analysis.Diagnostic) {
			fmt.Printf("line %d: %s\n", fset.Position(d.Pos).Line, d.Message)
			for _, fix := range d.SuggestedFixes {
				for _, te := range fix.TextEdits {
					edits = append(edits, edit{
						start:   fset.Position(te.Pos).Offset,
						end:     fset.Position(te.End).Offset,
						newText: string(te.NewText),
					})
				}
			}
		},
	}
	if _, err := a.Run(pass); err != nil {
		panic(err)
	}

	// Apply the fixes back-to-front so the offsets stay valid.
	out := src
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		out = out[:e.start] + e.newText + out[e.end:]
	}
	fmt.Println("---")
	fmt.Print(out)
	// Output:
	// line 6: Signature can be formatted more compactly
	// line 12: Signature can be formatted more compactly
	// line 18: Signature can be formatted more compactly
	// ---
	// package demo
	//
	// type Handler interface {
	// 	Process(request *Request) (*Response, error)
	// }
	//
	// type Server struct {
	// 	OnEvent func(event string) error
	// }
	//
	// func Serve(addr string, handler Handler) error {
	// 	return nil
	// }
}
