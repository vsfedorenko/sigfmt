package sigfmt

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

// lineDirectiveCorpus covers sources with //line directives. Before the
// fix, GetIndent panicked on such input: File.LineStart(File.Line(pos))
// requires the logical line number to fit the (sparse, directive-adjusted)
// line table, and a directive mapping beyond the physical end of the file
// made it throw "invalid line number". The linter must behave like go vet:
// process such files normally instead of crashing.
var lineDirectiveCorpus = []string{
	// //line directive remapping a signature to another file's coordinates.
	"package p\n\n//line other.go:100\nfunc F(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n",

	// /*line*/ block form of the same directive.
	"package p\n\n/*line other.go:42:7*/\nfunc G(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n",

	// Directive between declarations: the renderer must still resolve
	// indents of the following signature without panicking.
	"package p\n\nfunc H(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n\n//line gen.go:500\nfunc I(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n",

	// Interface method after a directive.
	"package p\n\n//line other.go:7\ntype I interface {\n\tM(\n\t\ta int,\n\t\tb string,\n\t) error\n}\n",

	// Struct func-field after a directive.
	"package p\n\n//line other.go:9\ntype S struct {\n\tF func(\n\t\tx int,\n\t\ty string,\n\t) error\n}\n",
}

// TestLineDirectivesDoNotCrash runs the public analyzer over the corpus,
// applies every suggested fix, and asserts the run completes without a
// panic and the result still parses.
func TestLineDirectivesDoNotCrash(t *testing.T) {
	for i, src := range lineDirectiveCorpus {
		t.Run(filepath.Join("case", string(rune('A'+i))), func(t *testing.T) {
			out := runAndFix(t, src)

			_, err := parser.ParseFile(token.NewFileSet(), "out.go", out, parser.ParseComments)
			require.NoError(t, err, "formatted output does not parse:\n%s", out)

			// //line directives must survive the fix untouched.
			if strings.Contains(src, "//line") {
				assert.Contains(t, out, "//line", "//line directive was dropped by the fix:\n%s", out)
			}
			if strings.Contains(src, "/*line") {
				assert.Contains(t, out, "/*line", "/*line*/ directive was dropped by the fix:\n%s", out)
			}
		})
	}
}

// TestLineDirectiveCollapseAndStability proves both directions: a
// collapsible signature after a //line directive IS still collapsed, and
// re-running on the fixed source is clean (no oscillation).
func TestLineDirectiveCollapseAndStability(t *testing.T) {
	src := lineDirectiveCorpus[0]

	out, editCount := runAndFixCount(t, src)
	require.NotZero(t, editCount, "first run produced no edits — //line must not disable formatting")
	assert.Contains(t, out, "func F(a int, b string) error {", "signature after //line was not collapsed:\n%s", out)
	assert.Contains(t, out, "//line other.go:100", "//line directive did not survive the fix:\n%s", out)

	// Second run over the fixed source must be a no-op.
	_, editCount = runAndFixCount(t, out)
	require.Zero(t, editCount, "second run still reports edits (oscillation)")
}

// TestBuildIgnoredFileSkipped proves the file-argument escape hatch is
// closed: a file with `//go:build ignore` passed DIRECTLY to the analyzer
// (bypassing the package loader) produces no diagnostics, matching go vet.
// A sibling file without the constraint must still be reported.
func TestBuildIgnoredFileSkipped(t *testing.T) {
	ignored := "//go:build ignore\n\npackage main\n\nfunc F(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n"
	normal := "package main\n\nfunc G(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n"

	for name, tc := range map[string]struct {
		src       string
		wantEdits int
	}{
		"ignored":     {ignored, 0},
		"normal":      {normal, 1},
		"linux-tag":   {"//go:build linux\n\npackage main\n\nfunc F(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n", 1},
		"windows-tag": {"//go:build windows\n\npackage main\n\nfunc F(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n", 0},
	} {
		t.Run(name, func(t *testing.T) {
			_, edits := runAndFixCount(t, tc.src)
			assert.Equal(t, tc.wantEdits, edits, "edit count")
		})
	}
}

// runAndFix runs the public analyzer over src, applies all suggested fixes,
// and returns the formatted source.
func runAndFix(t *testing.T, src string) string {
	out, _ := runAndFixCount(t, src)
	return out
}

// runAndFixCount additionally returns the number of text edits applied.
func runAndFixCount(t *testing.T, src string) (string, int) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.go")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))

	analyzer := NewAnalyzer()
	require.NoError(t, analyzer.Flags.Parse(nil), "parse flags")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	require.NoError(t, err, "fixture does not parse")

	type edit struct {
		start, end int
		newText    string
	}
	var edits []edit
	pass := &analysis.Pass{
		Analyzer: analyzer,
		Fset:     fset,
		Files:    []*ast.File{f},
		ReadFile: func(string) ([]byte, error) { return []byte(src), nil },
		Report: func(d analysis.Diagnostic) {
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
	_, err = analyzer.Run(pass)
	require.NoError(t, err, "run")

	out := src
	for j := len(edits) - 1; j >= 0; j-- {
		e := edits[j]
		out = out[:e.start] + e.newText + out[e.end:]
	}
	return out, len(edits)
}
