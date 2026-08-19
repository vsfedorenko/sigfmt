package field

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderNames(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  string
	}{
		{"Empty", []string{}, ""},
		{"Single", []string{"a"}, "a"},
		{"Multiple", []string{"a", "b", "c"}, "a, b, c"},
		{"Two", []string{"x", "y"}, "x, y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var idents []*ast.Ident
			for _, name := range tt.names {
				idents = append(idents, ast.NewIdent(name))
			}

			got := RenderNames(idents)
			assert.Equal(t, tt.want, got, "RenderNames()")
		})
	}
}
