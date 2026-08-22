package astinfo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vsfedorenko/sigfmt/internal/render"
)

func parseCode(t *testing.T, code string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	require.NoError(t, err)
	return fset, file
}

func TestFuncDecl(t *testing.T) {
	code := `package test

func Foo(a int, b string) error {
	return nil
}

func (r *Receiver) Method(x int) {
}
`
	fset, file := parseCode(t, code)
	r := render.New(8)
	e := New(r)

	funcDecl := file.Decls[0].(*ast.FuncDecl)
	sig := e.FuncDecl(fset, funcDecl)

	assert.NotNil(t, sig)
	assert.Equal(t, "Foo", sig.Name)
	assert.Equal(t, "func Foo(a int, b string) error", sig.OneLineText)
	assert.False(t, sig.IsStructField)
	assert.False(t, sig.IsInterfaceMethod)
	assert.Nil(t, sig.Receiver)

	methodDecl := file.Decls[1].(*ast.FuncDecl)
	sig = e.FuncDecl(fset, methodDecl)

	assert.NotNil(t, sig)
	assert.Equal(t, "Method", sig.Name)
	assert.Equal(t, "func (r *Receiver) Method(x int)", sig.OneLineText)
	assert.NotNil(t, sig.Receiver)
}

func TestFuncLit(t *testing.T) {
	code := `package test

var f = func(a int) string {
	return ""
}
`
	fset, file := parseCode(t, code)
	r := render.New(8)
	e := New(r)

	genDecl := file.Decls[0].(*ast.GenDecl)
	valueSpec := genDecl.Specs[0].(*ast.ValueSpec)
	funcLit := valueSpec.Values[0].(*ast.FuncLit)

	sig := e.FuncLit(fset, funcLit)

	assert.NotNil(t, sig)
	assert.Equal(t, "", sig.Name)
	assert.Equal(t, "func(a int) string", sig.OneLineText)
	assert.False(t, sig.IsStructField)
	assert.False(t, sig.IsInterfaceMethod)
}

func TestMethod(t *testing.T) {
	code := `package test

type MyInterface interface {
	DoSomething(ctx context.Context, id int) error
}
`
	fset, file := parseCode(t, code)
	r := render.New(8)
	e := New(r)

	genDecl := file.Decls[0].(*ast.GenDecl)
	typeSpec := genDecl.Specs[0].(*ast.TypeSpec)
	iface := typeSpec.Type.(*ast.InterfaceType)
	method := iface.Methods.List[0]

	sig := e.Method(fset, method.Names, method.Type.(*ast.FuncType))

	assert.NotNil(t, sig)
	assert.Equal(t, "DoSomething", sig.Name)
	assert.True(t, sig.IsInterfaceMethod)
	assert.False(t, sig.IsStructField)
}

func TestStructField(t *testing.T) {
	code := `package test

type MyStruct struct {
	Handler func(req Request) Response
}
`
	fset, file := parseCode(t, code)
	r := render.New(8)
	e := New(r)

	genDecl := file.Decls[0].(*ast.GenDecl)
	typeSpec := genDecl.Specs[0].(*ast.TypeSpec)
	structType := typeSpec.Type.(*ast.StructType)
	field := structType.Fields.List[0]

	sig := e.StructField(fset, field.Names, field.Type.(*ast.FuncType))

	assert.NotNil(t, sig)
	assert.Equal(t, "Handler", sig.Name)
	assert.True(t, sig.IsStructField)
	assert.False(t, sig.IsInterfaceMethod)
	assert.Equal(t, "Handler func(req Request) Response", sig.OneLineText)
}

func TestStructFieldMultipleNames(t *testing.T) {
	code := `package test

type MyStruct struct {
	Handler, Fallback func(req Request) Response
}
`
	fset, file := parseCode(t, code)
	r := render.New(8)
	e := New(r)

	genDecl := file.Decls[0].(*ast.GenDecl)
	typeSpec := genDecl.Specs[0].(*ast.TypeSpec)
	structType := typeSpec.Type.(*ast.StructType)
	field := structType.Fields.List[0]

	sig := e.StructField(fset, field.Names, field.Type.(*ast.FuncType))

	assert.NotNil(t, sig)
	// Every declared name must be rendered — a fix that drops any of them
	// silently deletes a struct field.
	assert.Equal(t, "Handler, Fallback", sig.Name)
	assert.Equal(t, "Handler, Fallback func(req Request) Response", sig.OneLineText)
	assert.True(t, sig.IsStructField)
}

// Edge inputs: no names, invalid positions — the extractors must return nil
// instead of building a broken signature.
func TestMethodAndStructFieldEdgeInputs(t *testing.T) {
	fset := token.NewFileSet()
	e := New(render.New(8))

	ft := &ast.FuncType{Params: &ast.FieldList{Opening: token.Pos(10), Closing: token.Pos(20)}}

	t.Run("method without names", func(t *testing.T) {
		assert.Nil(t, e.Method(fset, nil, ft), "no names -> nil")
	})

	t.Run("struct field without names", func(t *testing.T) {
		assert.Nil(t, e.StructField(fset, nil, ft), "no names -> nil")
	})

	t.Run("method with invalid start", func(t *testing.T) {
		names := []*ast.Ident{{Name: "M", NamePos: token.NoPos}}
		assert.Nil(t, e.Method(fset, names, ft), "invalid pos -> nil")
	})

	t.Run("func type without results closing", func(t *testing.T) {
		ftNoRes := &ast.FuncType{Params: &ast.FieldList{Opening: token.Pos(5), Closing: token.Pos(7)}}
		names := []*ast.Ident{{Name: "M", NamePos: token.Pos(1)}}
		sig := e.Method(fset, names, ftNoRes)
		require.NotNil(t, sig)
		assert.Equal(t, token.Pos(7), sig.DiagPos, "diagPos falls back to params closing")
	})
}
