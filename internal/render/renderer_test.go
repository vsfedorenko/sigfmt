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

		// fast-path branches of plainTypeString (62.9% -> full)
		{"ChanRecv", &ast.ChanType{Dir: ast.RECV, Value: &ast.Ident{Name: "int"}}, "<-chan int"},
		{"ChanSend", &ast.ChanType{Dir: ast.SEND, Value: &ast.Ident{Name: "int"}}, "chan<- int"},
		{"ChanBoth", &ast.ChanType{Dir: ast.RECV | ast.SEND, Value: &ast.Ident{Name: "int"}}, "chan int"},
		{"Ellipsis", &ast.Ellipsis{Elt: &ast.Ident{Name: "int"}}, "...int"},
		{"ChanOfPtr", &ast.ChanType{Dir: ast.RECV, Value: &ast.StarExpr{X: &ast.Ident{Name: "T"}}}, "<-chan *T"},
		{"MapOfSlices", &ast.MapType{Key: &ast.Ident{Name: "string"}, Value: &ast.ArrayType{Elt: &ast.Ident{Name: "byte"}}}, "map[string][]byte"},
		{"SizedArrayFallsBackToPrinter", &ast.ArrayType{Len: &ast.Ident{Name: "N"}, Elt: &ast.Ident{Name: "byte"}}, "[N]byte"},
		{"ChanOfSizedArray", &ast.ChanType{Dir: ast.RECV, Value: &ast.ArrayType{Len: &ast.Ident{Name: "N"}, Elt: &ast.Ident{Name: "byte"}}}, "<-chan [N]byte"},
		{"EllipsisOfSizedArray", &ast.Ellipsis{Elt: &ast.ArrayType{Len: &ast.Ident{Name: "N"}, Elt: &ast.Ident{Name: "byte"}}}, "...[N]byte"},
		{"MapOfChan", &ast.MapType{Key: &ast.ChanType{Dir: ast.SEND, Value: &ast.Ident{Name: "int"}}, Value: &ast.Ident{Name: "bool"}}, "map[chan<- int]bool"},
		{"StructFallsBackToPrinter", &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}}}, "struct { int }"},
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
