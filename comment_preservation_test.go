package sigfmt

import (
	"fmt"
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

// commentFuzzCorpus is the deterministic corpus for comment preservation:
// every entry is a Go source file whose signatures contain comments in
// unusual positions. The invariant under test: applying every suggested fix
// must not change the number of comment lines in the file — sigfmt either
// leaves a commented signature alone or rewrites it without touching
// comments.
var commentFuzzCorpus = []string{
	// Trailing parameter comments.
	"package p\n\nfunc A(\n\ta int, // param a\n\tb string, // param b\n) error {\n\treturn nil\n}\n",

	// Standalone comment line inside the parameter list.
	"package p\n\nfunc B(\n\t// ctx is required\n\tctx context.Context,\n\tname string,\n) error {\n\treturn nil\n}\n",

	// Comment before the closing paren.
	"package p\n\nfunc C(\n\ta int,\n\tb int,\n\t// both required\n) error {\n\treturn nil\n}\n",

	// Block comment inside the signature.
	"package p\n\nfunc D(\n\ta int, /* alpha */\n\tb int,\n) error {\n\treturn nil\n}\n",

	// Comment in a long signature that cannot collapse (packing path).
	"package p\n\nfunc E(ctx context.Context, request *Request, options map[string]string, // options map\n\ttimeout time.Duration, retries int, logger *Logger) (*Response, error) {\n\treturn nil, nil\n}\n",

	// Comment in an interface method.
	"package p\n\ntype I interface {\n\tM(\n\t\ta int, // a\n\t\tb int, // b\n\t) error\n}\n",

	// Comment in a struct field with func type.
	"package p\n\ntype S struct {\n\tF func(\n\t\tx int, // x\n\t\ty int,\n\t) error\n}\n",

	// Doc comment above the func is OUTSIDE the signature: collapsing is
	// still allowed; the doc comment must survive untouched.
	"package p\n\n// F does nothing.\nfunc F(\n\ta int,\n\tb int,\n) {\n}\n",

	// Mixed: one commented signature (must be skipped) and one clean
	// signature (must still be collapsed).
	"package p\n\nfunc G(\n\ta int, // keep me\n\tb int,\n) {\n}\n\nfunc H(\n\ta int,\n\tb int,\n) {\n}\n",

	// Generic function with comments inside the parameter list — the
	// generics renderer branch (#61) must skip it, not rewrite it.
	"package p\n\nfunc Map[T any, U any](\n\titems []T, // input slice\n\tfn func(T) U, // mapping fn\n) []U {\n\treturn nil\n}\n",
	// Generic interface method: comment inside the parameters of a method
	// whose interface type is generic.
	"package p\n\ntype I[T any] interface {\n\tM(\n\t\tx T, // x\n\t) T\n}\n",
	// Multi-name struct field (Handler, Fallback) with a comment inside
	// the func-typed field signature — the multi-name renderer branch must
	// skip the whole field, keeping every name.
	"package p\n\ntype T struct {\n\tHandler, Fallback func(\n\t\tw string, // w\n\t\tr string,\n\t) error\n}\n",
	// Variadic parameter carrying a trailing comment — the ellipsis
	// branch must skip the signature.
	"package p\n\nfunc V(\n\ta int, // a\n\trest ...string, // rest\n) error {\n\treturn nil\n}\n",
	// Method with a receiver and a commented parameter list — the
	// receiver-prefix branch must skip it.
	"package p\n\ntype S struct{}\n\nfunc (s *S) M(\n\tx int, // x\n\ty int,\n) error {\n\treturn nil\n}\n",
	// Struct tag after a commented func-typed field: the tag is outside
	// the rewritten span and must survive untouched.
	"package p\n\ntype T struct {\n\tF func(\n\t\tx int, // x\n\t\ty int,\n\t) error `json:\"f\"`\n}\n",
	// Plain-type fast path (#53): every parameter type is a plain
	// identifier rendered without go/printer — a comment must still veto
	// the rewrite before the fast path runs.
	"package p\n\nfunc FP(\n\ta int, // a\n\tb string, // b\n\tc bool, // c\n\tr rune, // r\n) error {\n\treturn nil\n}\n",
}

// TestCommentPreservationZeroLoss runs the public analyzer over the corpus,
// applies every suggested fix, and verifies that no comment text is lost.
func TestCommentPreservationZeroLoss(t *testing.T) {
	for i, src := range commentFuzzCorpus {
		t.Run(fmt.Sprintf("case%02d", i), func(t *testing.T) {
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

			// Apply edits from last to first so offsets stay valid.
			out := src
			for j := len(edits) - 1; j >= 0; j-- {
				e := edits[j]
				out = out[:e.start] + e.newText + out[e.end:]
			}

			assert.Equal(t, countComments(src), countComments(out),
				"comment loss: original markers vs formatted\noriginal:\n%s\nformatted:\n%s", src, out)

			// The result must still parse — a fix must never break the file.
			_, err = parser.ParseFile(token.NewFileSet(), path, out, parser.ParseComments)
			require.NoError(t, err, "formatted output does not parse:\n%s", out)

			// The mixed case (commented G alongside clean H): the clean
			// signature must still be collapsed.
			if strings.Contains(src, "func G(") {
				assert.Contains(t, out, "func H(a int, b int) {", "clean signature was not collapsed alongside the commented one:\n%s", out)
			}
		})
	}
}

// countComments counts comment markers in source text.
func countComments(src string) int {
	count := 0
	for i := 0; i+1 < len(src); i++ {
		if src[i] == '/' && (src[i+1] == '/' || src[i+1] == '*') {
			count++
		}
	}
	return count
}
