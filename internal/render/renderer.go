package render

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

// Renderer handles AST-to-text rendering and indentation calculations.
type Renderer struct {
	tabWidth int
}

// New creates a new Renderer.
func New(tabWidth int) *Renderer {
	return &Renderer{tabWidth: tabWidth}
}

// VisualLength calculates the visual length of a string, accounting for tab expansion.
func (r *Renderer) VisualLength(s string) int {
	length := 0
	for _, c := range s {
		if c == '\t' {
			length += r.tabWidth
		} else {
			length++
		}
	}
	return length
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

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return ""
	}

	return strings.Join(strings.Fields(buf.String()), " ")
}

// FieldList converts a list of fields (parameters, results, generics) to a comma-separated string.
func (r *Renderer) FieldList(fset *token.FileSet, fl *ast.FieldList, openBracket, closeBracket string) string {
	if fl == nil || len(fl.List) == 0 {
		return openBracket + closeBracket
	}

	var sb strings.Builder
	sb.WriteString(openBracket)

	for i, field := range fl.List {
		if i > 0 {
			sb.WriteString(", ")
		}

		for j, name := range field.Names {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(name.Name)
		}

		if len(field.Names) > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(r.Node(fset, field.Type))
	}

	sb.WriteString(closeBracket)
	return sb.String()
}

// GetIndent reads the source file to find the indentation of the line containing pos.
func (r *Renderer) GetIndent(fset *token.FileSet, pos token.Pos) string {
	f := fset.File(pos)
	if f == nil {
		return ""
	}

	content, err := os.ReadFile(f.Name())
	if err != nil {
		return ""
	}

	offset := f.Offset(pos)
	if offset >= len(content) {
		return ""
	}

	lineStart := f.LineStart(f.Line(pos))
	lineStartOffset := f.Offset(lineStart)

	if lineStartOffset > offset {
		return ""
	}

	prefix := content[lineStartOffset:offset]

	var indent strings.Builder
	for _, b := range prefix {
		if b == ' ' || b == '\t' {
			indent.WriteByte(b)
		} else {
			break
		}
	}
	return indent.String()
}
