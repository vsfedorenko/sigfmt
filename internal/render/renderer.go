package render

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	"github.com/vsfedorenko/sigfmt/internal/pkg/field"
)

// Renderer handles AST-to-text rendering and indentation calculations.
type Renderer struct {
	tabWidth int
}

// New creates a new Renderer.
func New(tabWidth int) *Renderer {
	return &Renderer{tabWidth: tabWidth}
}

// Results converts the results list (return values) to a string.
// It handles cases with and without parentheses.
func (r *Renderer) Results(fset *token.FileSet, results *ast.FieldList) string {
	if results == nil || len(results.List) == 0 {
		return ""
	}

	// Special case: Single unnamed result doesn't need parentheses
	if len(results.List) == 1 && len(results.List[0].Names) == 0 {
		return " " + r.Node(fset, results.List[0].Type)
	}

	return " " + r.FieldList(fset, results, "(", ")")
}

// Node converts an AST node to its string representation.
func (r *Renderer) Node(fset *token.FileSet, n ast.Node) string {
	if n == nil {
		return ""
	}

	// Optimization: Direct dispatch for FieldList
	if fl, ok := n.(*ast.FieldList); ok {
		return r.FieldList(fset, fl, "(", ")")
	}

	// Fast path: the plain type expressions that dominate parameter lists
	// (identifiers, qualified names, pointers, slices/arrays/maps/chans of
	// those) render identically to go/printer but without its tabwriter
	// machinery — profiling showed printer.Fprint at ~68% of analyzer
	// allocations.
	if s, ok := plainTypeString(n); ok {
		return s
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return ""
	}

	return normalizeSpacing(buf.String())
}

// plainTypeString renders simple type expressions without go/printer.
// The ok result is false for anything with literals, composite structure,
// ellipses, funcs, interfaces, structs or generics — those keep the
// printer path (normalizeSpacing still matters for multi-line output).
func plainTypeString(n ast.Node) (string, bool) {
	switch t := n.(type) {
	case *ast.Ident:
		return t.Name, true
	case *ast.SelectorExpr:
		prefix, ok := plainTypeString(t.X)
		if !ok {
			return "", false
		}
		return prefix + "." + t.Sel.Name, true
	case *ast.StarExpr:
		prefix, ok := plainTypeString(t.X)
		if !ok {
			return "", false
		}
		return "*" + prefix, true
	case *ast.ArrayType:
		if t.Len != nil {
			return "", false // sized arrays may carry expressions
		}
		elem, ok := plainTypeString(t.Elt)
		if !ok {
			return "", false
		}
		return "[]" + elem, true
	case *ast.MapType:
		key, ok := plainTypeString(t.Key)
		if !ok {
			return "", false
		}
		val, ok := plainTypeString(t.Value)
		if !ok {
			return "", false
		}
		return "map[" + key + "]" + val, true
	case *ast.ChanType:
		elem, ok := plainTypeString(t.Value)
		if !ok {
			return "", false
		}
		switch t.Dir {
		case ast.RECV:
			return "<-chan " + elem, true
		case ast.SEND:
			return "chan<- " + elem, true
		default:
			return "chan " + elem, true
		}
	case *ast.Ellipsis:
		elem, ok := plainTypeString(t.Elt)
		if !ok {
			return "", false
		}
		return "..." + elem, true
	}
	return "", false
}

// normalizeSpacing collapses runs of whitespace to single spaces and trims
// the ends — but only OUTSIDE string literals. go/printer output may span
// multiple lines for composite type expressions; naive strings.Fields
// would also mangle spaces inside literals (e.g. array-length expressions
// like [unsafe.Sizeof("a  b")]byte).
func normalizeSpacing(src string) string {
	var sb strings.Builder
	sb.Grow(len(src))

	inLiteral := false
	prevSpace := true // trims leading spaces without special-casing index 0

	for i := 0; i < len(src); i++ {
		ch := src[i]

		if ch == '"' {
			inLiteral = !inLiteral
			prevSpace = false
			sb.WriteByte(ch)
			continue
		}

		if inLiteral {
			sb.WriteByte(ch)
			continue
		}

		if ch == ' ' || ch == '	' || ch == '\n' || ch == '\r' {
			if !prevSpace {
				sb.WriteByte(' ')
				prevSpace = true
			}
			continue
		}

		prevSpace = false
		sb.WriteByte(ch)
	}

	// Trim the trailing space the loop may have appended (inLiteral is
	// false here for well-formed printer output).
	out := sb.String()
	return strings.TrimSuffix(out, " ")
}

// FieldList converts a list of fields (parameters, results, generics) to a comma-separated string.
func (r *Renderer) FieldList(fset *token.FileSet, fl *ast.FieldList, openBracket, closeBracket string) string {
	if fl == nil || len(fl.List) == 0 {
		return openBracket + closeBracket
	}

	var sb strings.Builder
	sb.WriteString(openBracket)

	for i, f := range fl.List {
		if i > 0 {
			sb.WriteString(", ")
		}

		names := field.RenderNames(f.Names)
		if names != "" {
			sb.WriteString(names)
			sb.WriteString(" ")
		}

		sb.WriteString(r.Node(fset, f.Type))
	}

	sb.WriteString(closeBracket)
	return sb.String()
}
