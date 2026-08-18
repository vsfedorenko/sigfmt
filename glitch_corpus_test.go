package sigfmt

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// glitchCorpus holds deliberately hostile signatures: extreme parameter
// counts, nested generics, pointer-of-pointer types, variadic map/slice/
// func chains, curried return types. Each entry must survive the full
// fix cycle: the applied output stays parseable, is a gofmt fixed point,
// and a second pass is a no-op (idempotence).
var glitchCorpus = []string{
	// Many parameters that no longer fit one line — packing path.
	"package g\n\nfunc Many(a1, a2 int, b1, b2 string, c1, c2 bool, d float64, e rune, f byte,\n\tg1, g2, g3, g4, g5, g6, g7, g8 int, h [16]byte, i chan int, j map[string]int,\n\tk func(), l interface{}, m any, n error, o ...int) {\n\t_ = a1\n}\n",

	// Nested generics with unions and constraints.
	"package g\n\nfunc Nested[T any, U ~int | ~string, V comparable](\n\tt T,\n\tu U,\n\tv V,\n) (r1 T, r2 U, r3 V) {\n\treturn t, u, v\n}\n",

	// Generic receiver with pointer-of-pointer parameter.
	"package g\n\ntype Container[T any] struct{ val T }\n\nfunc (c *Container[T]) Get(\n\tother *Container[T],\n\tdeep ***T,\n) T {\n\treturn c.val\n}\n",

	// Variadic of a map of slice of funcs.
	"package g\n\nfunc OnlyVariadic(\n\trest ...map[string][]func(int) error,\n) {\n}\n",

	// Curried function return type.
	"package g\n\nfunc Curried(\n\tx int,\n) func() func() error {\n\treturn nil\n}\n",

	// Struct field with func type (packing path).
	"package g\n\ntype Handler struct {\n\tProcess func(\n\t\tctx int,\n\t\tevent string,\n\t) error\n}\n",

	// Interface method with multi-line params (aggressive packing).
	"package g\n\ntype Service interface {\n\tDo(\n\t\tctx int,\n\t\tid string,\n\t) error\n}\n",

	// Single very long parameter (no split possible inside one param).
	"package g\n\nfunc OneHuge(\n\tparam map[string]map[int][]func(string) (map[string]error, error),\n) error {\n\treturn nil\n}\n",
}

// applyAllFixes runs the public analyzer over source and returns the
// source with every suggested fix applied.
func applyAllFixes(t *testing.T, src string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "glitch.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
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
		text       string
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
						start: fset.Position(te.Pos).Offset,
						end:   fset.Position(te.End).Offset,
						text:  string(te.NewText),
					})
				}
			}
		},
	}
	if _, err := analyzer.Run(pass); err != nil {
		t.Fatalf("run: %v", err)
	}

	out := src
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		out = out[:e.start] + e.text + out[e.end:]
	}
	return out
}

// TestGlitchCorpus_FixCycleInvariants: for every hostile signature —
// the fixed output (1) parses, (2) is a gofmt fixed point, (3) a second
// analyzer pass over it is a no-op (idempotence).
func TestGlitchCorpus_FixCycleInvariants(t *testing.T) {
	for i, src := range glitchCorpus {
		t.Run(corpusName(i), func(t *testing.T) {
			fixed := applyAllFixes(t, src)

			// (1) parses
			fset := token.NewFileSet()
			if _, err := parser.ParseFile(fset, "fixed.go", fixed, parser.ParseComments); err != nil {
				t.Fatalf("fixed output does not parse: %v\n--- fixed ---\n%s", err, fixed)
			}

			// (2) gofmt fixed point
			formatted, err := format.Source([]byte(fixed))
			if err != nil {
				t.Fatalf("gofmt rejects fixed output: %v\n--- fixed ---\n%s", err, fixed)
			}
			if string(formatted) != fixed {
				t.Errorf("fixed output is not a gofmt fixed point:\n--- got ---\n%s--- want ---\n%s", fixed, formatted)
			}

			// (3) idempotence: second pass proposes nothing
			second := applyAllFixes(t, fixed)
			if second != fixed {
				t.Errorf("second pass is not a no-op:\n--- after 1st ---\n%s--- after 2nd ---\n%s", fixed, second)
			}
		})
	}
}

// TestGlitchCorpus_OriginalsAreValid pins that the corpus itself is
// well-formed: a broken fixture would silently test nothing.
func TestGlitchCorpus_OriginalsAreValid(t *testing.T) {
	for i, src := range glitchCorpus {
		fset := token.NewFileSet()
		if _, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments); err != nil {
			t.Errorf("%s: corpus entry does not parse: %v", corpusName(i), err)
		}
		if !strings.HasPrefix(src, "package g") {
			t.Errorf("%s: corpus entry missing package clause", corpusName(i))
		}
	}
}

func corpusName(i int) string {
	return "case" + strings.Repeat("0", maxDigits-len(itoa(i))) + itoa(i)
}

const maxDigits = 2

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// TestGlitchCorpus_FixedOutputCompiles: the fixed output must not only
// parse but type-check — a rewrite that changes semantics (e.g. drops a
// type constraint) would parse fine and still break the build. This is
// the strongest invariant the corpus can assert without importing the
// fixtures as a real module.
func TestGlitchCorpus_FixedOutputCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("type-check needs the go toolchain")
	}
	for i, src := range glitchCorpus {
		t.Run(corpusName(i), func(t *testing.T) {
			fixed := applyAllFixes(t, src)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module glitch.test\n\ngo 1.25\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "g.go"), []byte(fixed), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "vet", "./...")
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("fixed output does not type-check (go vet): %v\n%s\n--- fixed ---\n%s", err, out, fixed)
			}
		})
	}
}
