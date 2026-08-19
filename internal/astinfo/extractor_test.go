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
