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
}

// TestCommentPreservationZeroLoss runs the public analyzer over the corpus,
// applies every suggested fix, and verifies that no comment text is lost.
func TestCommentPreservationZeroLoss(t *testing.T) {
	for i, src := range commentFuzzCorpus {
		t.Run(fmt.Sprintf("case%02d", i), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "fixture.go")
			if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}

			analyzer := NewAnalyzer()
			if err := analyzer.Flags.Parse(nil); err != nil {
				t.Fatalf("parse flags: %v", err)
			}

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
			if err != nil {
				t.Fatalf("fixture does not parse: %v", err)
			}

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
			if _, err := analyzer.Run(pass); err != nil {
				t.Fatalf("run: %v", err)
			}

			// Apply edits from last to first so offsets stay valid.
			out := src
			for j := len(edits) - 1; j >= 0; j-- {
				e := edits[j]
				out = out[:e.start] + e.newText + out[e.end:]
			}

			if got, want := countComments(out), countComments(src); got != want {
				t.Errorf("comment loss: original has %d comment markers, formatted has %d\noriginal:\n%s\nformatted:\n%s",
					want, got, src, out)
			}

			// The result must still parse — a fix must never break the file.
			if _, err := parser.ParseFile(token.NewFileSet(), path, out, parser.ParseComments); err != nil {
				t.Fatalf("formatted output does not parse: %v\n%s", err, out)
			}

			// Case 8 (mixed): the clean signature must still be collapsed.
			if i == len(commentFuzzCorpus)-1 && !strings.Contains(out, "func H(a int, b int) {") {
				t.Errorf("clean signature was not collapsed alongside the commented one:\n%s", out)
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
