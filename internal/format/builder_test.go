package format

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

func TestBuilder_BuildReformattedSignature(t *testing.T) {
	fset := token.NewFileSet()
	file := fset.AddFile("test.go", -1, 1000)
	for i := 0; i < 10; i++ {
		file.AddLine(i * 10)
	}

	r := render.New(4)
	cfg := config.Settings{MaxLineLen: 80, TabWidth: 4}
	builder := NewBuilder(cfg, r)

	tests := []struct {
		name     string
		sig      *domain.Signature
		contains string
	}{
		{
			name: "RegularFunction",
			sig: &domain.Signature{
				Name:        "Foo",
				Start:       file.Pos(0),
				OneLineText: "func Foo(a int, b string) error",
				FuncType: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "int"}},
							{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "string"}},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: &ast.Ident{Name: "error"}}},
					},
				},
			},
			contains: "func Foo",
		},
		{
			name: "InterfaceMethod",
			sig: &domain.Signature{
				Name:              "Method",
				Start:             file.Pos(0),
				OneLineText:       "Method(x int)",
				IsInterfaceMethod: true,
				FuncType: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Names: []*ast.Ident{{Name: "x"}}, Type: &ast.Ident{Name: "int"}},
						},
					},
				},
			},
			contains: "Method",
		},
		{
			name: "StructField",
			sig: &domain.Signature{
				Name:          "Handler",
				Start:         file.Pos(0),
				OneLineText:   "Handler func(req Request) Response",
				IsStructField: true,
				FuncType: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Names: []*ast.Ident{{Name: "req"}}, Type: &ast.Ident{Name: "Request"}},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: &ast.Ident{Name: "Response"}}},
					},
				},
			},
			contains: "Handler func",
		},
		{
			name: "NilFuncType",
			sig: &domain.Signature{
				Name:     "Invalid",
				FuncType: nil,
			},
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.BuildReformattedSignature(newSyntheticFile(fset), tt.sig)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestBuilder_MatchesGroup(t *testing.T) {
	fset := token.NewFileSet()
	r := render.New(4)
	cfg := config.Settings{}
	builder := NewBuilder(cfg, r)

	fields := []*ast.Field{
		{Type: &ast.SelectorExpr{X: &ast.Ident{Name: "context"}, Sel: &ast.Ident{Name: "Context"}}},
		{Type: &ast.StarExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: "sql"}, Sel: &ast.Ident{Name: "Tx"}}}},
		{Type: &ast.Ident{Name: "int"}},
	}

	tests := []struct {
		name     string
		startIdx int
		group    []string
		want     bool
	}{
		{
			name:     "MatchesGroup",
			startIdx: 0,
			group:    []string{"context.Context", "*sql.Tx"},
			want:     true,
		},
		{
			name:     "DoesNotMatch",
			startIdx: 0,
			group:    []string{"context.Context", "int"},
			want:     false,
		},
		{
			name:     "OutOfBounds",
			startIdx: 2,
			group:    []string{"int", "string"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.matchesGroup(fset, fields, tt.startIdx, tt.group)
			assert.Equal(t, tt.want, result)
		})
	}
}
