package sigfmt

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

// Benchmark corpus sizes (files and signatures per file).
const (
	benchFilesPerCategory = 12
	benchDeclsPerFile     = 40
)

// benchCorpus profiles. The violation corpus is the worst case (every
// signature needs formatting); the clean corpus is the incremental case (a
// mostly-formatted codebase where the analyzer must decide there is nothing
// to do).
type benchProfile struct {
	name string
	dir  string
}

func benchCorpora(tb testing.TB) []benchProfile {
	tb.Helper()
	violations := filepath.Join(os.TempDir(), fmt.Sprintf("sigfmt-bench-corpus-%d", benchCorpusSeed))
	clean := filepath.Join(os.TempDir(), fmt.Sprintf("sigfmt-bench-clean-%d", benchCorpusSeed))
	for dir, write := range map[string]func(string) error{
		violations: writeBenchCorpus,
		clean:      writeBenchCorpusClean,
	} {
		require.NoError(tb, os.MkdirAll(dir, 0o755), "mkdir corpus")
		require.NoError(tb, write(dir), "write corpus")
	}
	tb.Cleanup(func() {
		_ = os.RemoveAll(violations)
		_ = os.RemoveAll(clean)
	})
	return []benchProfile{{"violations", violations}, {"clean", clean}}
}

// benchSources loads every .go file of the corpus into memory.
func benchSources(tb testing.TB, dir string) (paths []string, sources map[string][]byte) {
	tb.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(tb, err, "glob")
	require.NotEmpty(tb, paths, "corpus is empty: %s", dir)
	sources = map[string][]byte{}
	for _, path := range paths {
		src, err := os.ReadFile(path)
		require.NoError(tb, err, "read %s", path)
		sources[path] = src
	}
	return paths, sources
}

// runAnalyzerBenchmark drives the public analyzer (NewAnalyzer) over the
// corpus. With withParse the files are re-parsed inside the timed loop —
// the fair comparison point against the gofmt baseline, which also pays the
// parse. Without it, the ASTs are parsed once outside the loop: the
// per-package cost a golangci-lint run pays on top of package loading.
func runAnalyzerBenchmark(b *testing.B, dir string, withParse bool) int {
	b.Helper()

	paths, sources := benchSources(b, dir)
	readFile := func(name string) ([]byte, error) { return sources[name], nil }
	analyzer := NewAnalyzer()
	if err := analyzer.Flags.Parse(nil); err != nil {
		b.Fatalf("parse flags: %v", err)
	}

	fset := token.NewFileSet()
	var asts []*ast.File
	if !withParse {
		for _, path := range paths {
			f, err := parser.ParseFile(fset, path, sources[path], parser.ParseComments)
			if err != nil {
				b.Fatalf("parse %s: %v", path, err)
			}
			asts = append(asts, f)
		}
	}

	diagnostics := 0
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diagnostics = 0
		if withParse {
			fset = token.NewFileSet()
			asts = asts[:0]
			for _, path := range paths {
				f, err := parser.ParseFile(fset, path, sources[path], parser.ParseComments)
				if err != nil {
					b.Fatalf("parse %s: %v", path, err)
				}
				asts = append(asts, f)
			}
		}
		pass := &analysis.Pass{
			Analyzer: analyzer,
			Fset:     fset,
			Files:    asts,
			ReadFile: readFile,
			Report:   func(_ analysis.Diagnostic) { diagnostics++ },
		}
		if _, err := analyzer.Run(pass); err != nil {
			b.Fatalf("run: %v", err)
		}
	}
	b.StopTimer()
	return diagnostics
}

// BenchmarkAnalyzer measures the analysis path (AST walk, signature
// extraction, formatting strategies, no-op guard) without parsing, per corpus
// profile. Corpus sanity is asserted: the violation profile must produce
// diagnostics, the clean profile must produce none.
func BenchmarkAnalyzer(b *testing.B) {
	for _, profile := range benchCorpora(b) {
		b.Run(profile.name, func(b *testing.B) {
			diagnostics := runAnalyzerBenchmark(b, profile.dir, false)
			if profile.name == "violations" && diagnostics == 0 {
				b.Fatalf("violation corpus produced no diagnostics — benchmark is measuring nothing")
			}
			if profile.name == "clean" && diagnostics != 0 {
				b.Fatalf("clean corpus produced %d diagnostics, want 0", diagnostics)
			}
		})
	}
}

// BenchmarkSigfmtWithParse measures sigfmt's full pipeline including parsing.
// Compare against BenchmarkGofmtBaseline on the same corpus profile — the
// roadmap target is <2x gofmt.
func BenchmarkSigfmtWithParse(b *testing.B) {
	for _, profile := range benchCorpora(b) {
		b.Run(profile.name, func(b *testing.B) {
			runAnalyzerBenchmark(b, profile.dir, true)
		})
	}
}

// BenchmarkGofmtBaseline measures what the gofmt binary does to the same
// corpus: parse every file and pretty-print it via go/format.
func BenchmarkGofmtBaseline(b *testing.B) {
	for _, profile := range benchCorpora(b) {
		b.Run(profile.name, func(b *testing.B) {
			_, sources := benchSources(b, profile.dir)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, src := range sources {
					if _, err := format.Source(src); err != nil {
						b.Fatalf("format: %v", err)
					}
				}
			}
		})
	}
}

// TestBenchCorpusIsGofmtClean guards the benchmark corpus itself: a corpus
// file that gofmt would reformat would leak formatting work into the
// measurement and make runs non-comparable. It uses go/format.Source — the
// exact library call the gofmt binary makes.
func TestBenchCorpusIsGofmtClean(t *testing.T) {
	for _, profile := range benchCorpora(t) {
		paths, sources := benchSources(t, profile.dir)
		for _, path := range paths {
			src := sources[path]
			formatted, err := format.Source(src)
			require.NoError(t, err, "format %s", path)
			assert.True(t, bytes.Equal(src, formatted), "corpus file %s is not gofmt-clean; regenerate the corpus generator output", path)
		}
	}
}
