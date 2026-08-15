package render

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestNormalizeSpacing verifies whitespace collapsing outside string
// literals and byte-preservation inside them.
func TestNormalizeSpacing(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"collapse spaces", "a  b\t c\nd", "a b c d"},
		{"trim ends", "  x  ", "x"},
		{"empty", "   ", ""},
		{
			"literal spaces preserved",
			`[unsafe.Sizeof("a  b")]byte`,
			`[unsafe.Sizeof("a  b")]byte`,
		},
		{
			"literal with spaces and trailing text",
			`x  [len("k  k")]int  y`,
			`x [len("k  k")]int y`,
		},
		{
			"two literals",
			`f("a  b",  "c   d")`,
			`f("a  b", "c   d")`,
		},
		{
			"escaped quote keeps literal tracking sane",
			`[len("a \"q\"  b")]byte`,
			`[len("a \"q\"  b")]byte`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSpacing(tt.in); got != tt.want {
				t.Errorf("normalizeSpacing(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestRenderer_Node_ArrayTypeWithLiteral guards the regression: rendering
// an array type whose length expression contains a string literal with
// consecutive spaces must not collapse them — the old strings.Fields
// normalization rewrote "a  b" into "a b" inside the literal.
func TestRenderer_Node_ArrayTypeWithLiteral(t *testing.T) {
	src := "package p\n\nvar buf [unsafe.Sizeof(\"a  b\")]byte\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	var array *ast.ArrayType
	ast.Inspect(f, func(n ast.Node) bool {
		if at, ok := n.(*ast.ArrayType); ok {
			array = at
		}
		return true
	})
	if array == nil {
		t.Fatal("no ArrayType found in source")
	}

	got := New(8).Node(fset, array)
	want := `[unsafe.Sizeof("a  b")]byte`
	if got != want {
		t.Fatalf("Node() = %q, want %q", got, want)
	}
}
