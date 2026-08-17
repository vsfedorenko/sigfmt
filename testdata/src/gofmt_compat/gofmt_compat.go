// Package gofmt_compat verifies that sigfmt's suggested fixes leave files that
// use gofmt -s simplified syntax (composite literals, range over int) as
// gofmt -s fixed points: the auto-fix output never diverges from gofmt -s.
package gofmt_compat

import "context"

// Point is a small value type used in composite literals below.
type Point struct {
	X, Y int
}

// CollapsedSignatureWithSimplifiedLiterals has a multi-line signature that
// sigfmt collapses, while the body relies on gofmt -s simplified composite
// literals (inner composite literal types elided).
func CollapsedSignatureWithSimplifiedLiterals(
	ctx context.Context,
	grid [][]int,
) []Point { // want "Signature can be formatted more compactly"
	_ = ctx
	triangles := [][]Point{
		{{1, 2}, {3, 4}},
		{{5, 6}, {7, 8}},
	}
	first := triangles[0][0]
	return []Point{first, {X: 9, Y: 10}}
}

// PackedSignatureWithRangeOverInt collapses onto one line next to a
// range-over-int loop and elided composite literal types in the body.
func PackedSignatureWithRangeOverInt(
	first string,
	second string,
) []string { // want "Signature can be formatted more compactly"
	values := []string{first, second}
	out := make([]string, 0, len(values))
	for i := range len(values) {
		out = append(out, values[i])
	}
	return out
}

// InterfaceWithSimplifiedUsage packs its methods while the function below it
// slices a composite literal and ranges over an int, all in -s form already.
type InterfaceWithSimplifiedUsage interface {
	// Transform should be packed onto one line by sigfmt.
	Transform(
		ctx context.Context,
		input []byte,
	) ([]byte, error) // want "Signature can be formatted more compactly"

	// Identity stays as is.
	Identity() error
}

// UseSimplifiedSyntax exercises gofmt -s forms in every surrounding position
// of a signature that sigfmt rewrites.
func UseSimplifiedSyntax(
	handler InterfaceWithSimplifiedUsage,
	points []Point,
) [][]Point { // want "Signature can be formatted more compactly"
	matrix := [][]Point{
		{{1, 1}, {2, 2}},
		{{3, 3}, {4, 4}},
	}
	for i := range len(matrix) {
		matrix[i] = append(matrix[i], points[i])
	}
	return matrix
}

// VariadicAndElidedLiterals combines a variadic parameter with gofmt -s
// simplified literals in default-free Go fashion.
func VariadicAndElidedLiterals(
	name string,
	values ...int,
) map[string][]Point { // want "Signature can be formatted more compactly"
	points := make([]Point, 0, len(values))
	for _, v := range values {
		points = append(points, Point{X: v, Y: v})
	}
	return map[string][]Point{
		name: {{X: 0, Y: 0}},
	}
}
