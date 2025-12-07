package render

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisualLength(t *testing.T) {
	r := New(8)
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"Empty", "", 0},
		{"NoTabs", "func Foo(a int)", 15},
		{"OneTab", "func\tFoo(a int)", 22},
		{"MultipleTabs", "\t\tfunc Foo(a int)", 31},
		{"Spaces", "    func Foo(a int)", 19},
		{"Mixed", "    \tfunc Foo(a int)", 27},
		{"LongString", strings.Repeat("a", 100), 100},
		{"TabWidth4", "func\tFoo(a int)", 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := r
			if tt.name == "TabWidth4" {
				renderer = New(4)
			}
			assert.Equal(t, tt.expected, renderer.VisualLength(tt.input))
		})
	}
}

func TestGetIndent(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "testfile.go")

	content := `package main

func main() {
	var a int
		var b string
	// comment
    func Foo() {
		// nested comment
	}
}
`
	err := os.WriteFile(filePath, []byte(content), 0o644)
	assert.NoError(t, err)

	fset := token.NewFileSet()
	f := fset.AddFile(filePath, -1, len(content))
	f.SetLinesForContent([]byte(content))
	r := New(8)

	tests := []struct {
		name     string
		lineNum  int
		colNum   int
		expected string
	}{
		{"NoIndent", 1, 1, ""},
		{"RootFunc", 3, 1, ""},
		{"OneTab", 4, 2, "\t"},
		{"TwoTabs", 5, 3, "\t\t"},
		{"FourSpaces", 7, 5, "    "},
		{"NestedTwoTabs", 8, 3, "\t\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := f.Pos(f.Offset(f.LineStart(tt.lineNum)) + tt.colNum - 1)
			assert.Equal(t, tt.expected, r.GetIndent(fset, pos))
		})
	}
}

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