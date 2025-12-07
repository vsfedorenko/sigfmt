package field

import (
	"go/ast"
	"strings"
)

// RenderNames renders field names with commas.
// Example: for names [a, b, c] returns "a, b, c"
func RenderNames(names []*ast.Ident) string {
	if len(names) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, name := range names {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(name.Name)
	}
	return sb.String()
}
