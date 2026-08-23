package sigfmt

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

// fuzzEdits holds the suggested-fix text edits of one analyzer run, converted
// to byte offsets, sorted and non-overlapping (analyzer contract).
type fuzzEdits struct {
	start, end int
	text       string
}

// fuzzApply runs the public analyzer over src and returns the source with
// every suggested fix applied. It mirrors applyAllFixes from
// glitch_corpus_test.go but tolerates unparsable input (fuzz payloads can be
// arbitrary bytes): a parse failure means "nothing to format", not a test
// error — the invariants below only start once the payload parses.
func fuzzApply(t *testing.T, src string) (string, bool) {
	t.Helper()

	analyzer := NewAnalyzer()
	require.NoError(t, analyzer.Flags.Parse(nil), "parse flags")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fuzz.go", src, parser.ParseComments)
	if err != nil {
		return src, false
	}

	var edits []fuzzEdits
	pass := &analysis.Pass{
		Analyzer: analyzer,
		Fset:     fset,
		Files:    []*ast.File{f},
		ReadFile: func(string) ([]byte, error) { return []byte(src), nil },
		Report: func(d analysis.Diagnostic) {
			for _, fix := range d.SuggestedFixes {
				for _, te := range fix.TextEdits {
					edits = append(edits, fuzzEdits{
						start: fset.Position(te.Pos).Offset,
						end:   fset.Position(te.End).Offset,
						text:  string(te.NewText),
					})
				}
			}
		},
	}
	if _, err := analyzer.Run(pass); err != nil {
		t.Fatalf("analyzer.Run on parsable input: %v", err)
	}

	out := src
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		out = out[:e.start] + e.text + out[e.end:]
	}
	return out, true
}

// TestFuzzFixInvariants is the seed run of the fuzz harness: the same
// three invariants the glitch corpus pins (output parses, is a gofmt fixed
// point, second pass is a no-op), executed over the seed corpus derived from
// the hostile-signature corpus. `go test -fuzz=FuzzFixInvariants` grows the
// corpus from here; `go test` replays seed + accumulated failures.
func TestFuzzFixInvariants(t *testing.T) {
	for i, seed := range fuzzSeeds(t) {
		t.Run(corpusName(i), func(t *testing.T) {
			assertFixInvariants(t, seed)
		})
	}
}

func assertFixInvariants(t *testing.T, src string) {
	t.Helper()

	fixed, ok := fuzzApply(t, src)
	if !ok {
		t.Skip("payload does not parse")
	}

	// (1) the fixed output parses
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "fixed.go", fixed, parser.ParseComments)
	require.NoError(t, err, "fixed output does not parse:\n--- fixed ---\n%s", fixed)

	// (2) gofmt preservation: IF the input was a gofmt fixed point, the
	// output must stay one. sigfmt formats signatures, not files — making
	// an unformatted file formatted is gofmt's job, so the guarantee is
	// "never break", not "always fix". (Found by the fuzzer itself: a
	// bare `package A` with no trailing newline is not a gofmt fixed
	// point, and sigfmt correctly leaves it alone.)
	if in, err := format.Source([]byte(src)); err == nil && string(in) == src {
		formatted, err := format.Source([]byte(fixed))
		require.NoError(t, err, "gofmt rejects fixed output:\n--- fixed ---\n%s", fixed)
		require.Equal(t, fixed, string(formatted), "gofmt-clean input became unformatted:\n--- fixed ---\n%s", fixed)
	}

	// (3) idempotence: second pass proposes nothing
	second, _ := fuzzApply(t, fixed)
	require.Equal(t, fixed, second, "second pass is not a no-op")
}

// fuzzSeeds returns the seed corpus: the glitch corpus entries, so replayed
// fuzz seeds stay exercised even without the Go fuzzing engine.
func fuzzSeeds(t *testing.T) []string {
	t.Helper()

	return append([]string(nil), glitchCorpus...)
}

// FuzzFixInvariants is the harness: for arbitrary bytes, IF the payload
// parses as a Go file, the analyzer's applied output must (1) parse,
// (2) be a gofmt fixed point, (3) be a second-pass no-op. Parse failures of
// the input are legitimate (not every byte string is Go), but once the input
// parses, a panic or broken output in the linter is a real bug.
//
// Run the fuzzer:  go test -fuzz=FuzzFixInvariants -fuzztime=60s
// Replay seeds:    go test -run=FuzzFixInvariants
// New crashers land in testdata/fuzz/FuzzFixInvariants/ as failing cases.
func FuzzFixInvariants(f *testing.F) {
	for _, src := range glitchCorpus {
		f.Add(src)
	}
	f.Fuzz(func(t *testing.T, src string) {
		assertFixInvariants(t, src)
	})
}
