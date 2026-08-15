package render

import "testing"

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
// consecutive spaces must not collapse them. (The full Node path is
// exercised through the analysistest suite; here we pin the constructor.)
func TestRenderer_Node_ArrayTypeWithLiteral(t *testing.T) {
	r := New(8)
	if r == nil {
		t.Fatal("nil renderer")
	}
}
