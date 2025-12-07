package render

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNode(t *testing.T) {
	fset := token.NewFileSet()
	r := New(8)

	tests := []struct {
		name     string
		node     ast.Node
		expected string
	}{
		{"Ident", &ast.Ident{Name: "int"}, "int"},
		{"StarExpr", &ast.StarExpr{X: &ast.Ident{Name: "Context"}}, "*Context"},
		{"SelectorExpr", &ast.SelectorExpr{X: &ast.Ident{Name: "context"}, Sel: &ast.Ident{Name: "Context"}}, "context.Context"},
		{"ArrayType", &ast.ArrayType{Elt: &ast.Ident{Name: "string"}}, "[]string"},
		{"MapType", &ast.MapType{Key: &ast.Ident{Name: "string"}, Value: &ast.Ident{Name: "int"}}, "map[string]int"},
		{"FuncType", &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}}}, "func(int)"},
		{"InterfaceType", &ast.InterfaceType{Methods: &ast.FieldList{}}, "interface { }"},
		{"EmptyFieldList", &ast.FieldList{}, "()"},
		{"NilNode", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, r.Node(fset, tt.node))
		})
	}
}

func TestFieldList(t *testing.T) {
	fset := token.NewFileSet()
	r := New(8)

	tests := []struct {
		name         string
		fieldList    *ast.FieldList
		openBracket  string
		closeBracket string
		expected     string
	}{
		{"Empty", &ast.FieldList{}, "(", ")", "()"},
		{"SingleNamed", &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "int"}}}}, "(", ")", "(a int)"},
		{"MultipleNamed", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "int"}},
			{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "string"}},
		}}, "(", ")", "(a int, b string)"},
		{"MultipleSameType", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "a"}, {Name: "b"}}, Type: &ast.Ident{Name: "int"}},
		}}, "(", ")", "(a, b int)"},
		{"Mixed", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "a"}, {Name: "b"}}, Type: &ast.Ident{Name: "int"}},
			{Names: []*ast.Ident{{Name: "c"}}, Type: &ast.Ident{Name: "string"}},
		}}, "(", ")", "(a, b int, c string)"},
		{"TypeParams", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "T"}}, Type: &ast.Ident{Name: "any"}},
			{Names: []*ast.Ident{{Name: "R"}}, Type: &ast.Ident{Name: "comparable"}},
		}}, "[", "]", "[T any, R comparable]"},
		{"Variadic", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "args"}}, Type: &ast.Ellipsis{Elt: &ast.Ident{Name: "interface{}"}}},
		}}, "(", ")", "(args ...interface{})"},
		{"FuncTypeParam", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "fn"}}, Type: &ast.FuncType{
				Params:  &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "string"}}}},
			}},
		}}, "(", ")", "(fn func(int) string)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, r.FieldList(fset, tt.fieldList, tt.openBracket, tt.closeBracket))
		})
	}
}

func TestResults(t *testing.T) {
	fset := token.NewFileSet()
	r := New(8)

	tests := []struct {
		name     string
		results  *ast.FieldList
		expected string
	}{
		{"Empty", &ast.FieldList{}, ""},
		{"SingleUnnamed", &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "error"}}}}, " error"},
		{"SingleNamed", &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "err"}}, Type: &ast.Ident{Name: "error"}}}}, " (err error)"},
		{"MultipleUnnamed", &ast.FieldList{List: []*ast.Field{
			{Type: &ast.Ident{Name: "int"}},
			{Type: &ast.Ident{Name: "error"}},
		}}, " (int, error)"},
		{"MultipleNamed", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "count"}}, Type: &ast.Ident{Name: "int"}},
			{Names: []*ast.Ident{{Name: "err"}}, Type: &ast.Ident{Name: "error"}},
		}}, " (count int, err error)"},
		{"Mixed", &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{{Name: "count"}}, Type: &ast.Ident{Name: "int"}},
			{Type: &ast.Ident{Name: "error"}},
		}}, " (count int, error)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, r.Results(fset, tt.results))
		})
	}
}
