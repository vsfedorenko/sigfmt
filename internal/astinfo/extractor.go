package astinfo

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

// Extractor extracts signature information from AST nodes.
type Extractor struct {
	renderer *render.Renderer
}

// New creates a new Extractor.
func New(renderer *render.Renderer) *Extractor {
	return &Extractor{renderer: renderer}
}

// FuncDecl extracts info from a function declaration.
func (e *Extractor) FuncDecl(fset *token.FileSet, decl *ast.FuncDecl) *domain.Signature {
	start := decl.Type.Func
	end := decl.Type.Params.End()
	if decl.Type.Results != nil {
		end = decl.Type.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &domain.Signature{
		Start:         start,
		End:           end,
		DiagPos:       computeDiagPos(decl.Type.Params, decl.Type.Results),
		OneLineText:   e.buildFuncDeclSignature(fset, decl),
		FuncType:      decl.Type,
		Receiver:      decl.Recv,
		Name:          decl.Name.Name,
		IsStructField: false,
	}
}

// FuncLit extracts info from a function literal.
func (e *Extractor) FuncLit(fset *token.FileSet, lit *ast.FuncLit) *domain.Signature {
	start := lit.Type.Func
	end := lit.Type.Params.End()
	if lit.Type.Results != nil {
		end = lit.Type.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &domain.Signature{
		Start:         start,
		End:           end,
		DiagPos:       computeDiagPos(lit.Type.Params, lit.Type.Results),
		OneLineText:   e.buildFuncLitSignature(fset, lit.Type),
		FuncType:      lit.Type,
		Receiver:      nil,
		Name:          "",
		IsStructField: false,
	}
}

// Method extracts info from an interface method.
func (e *Extractor) Method(fset *token.FileSet, name *ast.Ident, ft *ast.FuncType) *domain.Signature {
	start := name.Pos()
	end := ft.Params.End()
	if ft.Results != nil {
		end = ft.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &domain.Signature{
		Start:             start,
		End:               end,
		DiagPos:           computeDiagPos(ft.Params, ft.Results),
		OneLineText:       e.buildMethodSignature(fset, name.Name, ft),
		FuncType:          ft,
		Receiver:          nil,
		Name:              name.Name,
		IsStructField:     false,
		IsInterfaceMethod: true,
	}
}

// StructField extracts info from a struct field.
func (e *Extractor) StructField(fset *token.FileSet, name *ast.Ident, ft *ast.FuncType) *domain.Signature {
	start := name.Pos()
	end := ft.Params.End()
	if ft.Results != nil {
		end = ft.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &domain.Signature{
		Start:         start,
		End:           end,
		DiagPos:       computeDiagPos(ft.Params, ft.Results),
		OneLineText:   e.buildStructFieldSignature(fset, name.Name, ft),
		FuncType:      ft,
		Receiver:      nil,
		Name:          name.Name,
		IsStructField: true,
	}
}

func computeDiagPos(params, results *ast.FieldList) token.Pos {
	diagPos := params.Closing
	if results != nil && results.Closing.IsValid() {
		diagPos = results.Closing
	}
	return diagPos
}

// buildSignature helper
func (e *Extractor) buildSignature(fset *token.FileSet, prefix string, ft *ast.FuncType, results *ast.FieldList) string {
	var sb strings.Builder
	sb.WriteString(prefix)

	if ft.TypeParams != nil {
		sb.WriteString(e.renderer.FieldList(fset, ft.TypeParams, "[", "]"))
	}

	sb.WriteString(e.renderer.FieldList(fset, ft.Params, "(", ")"))

	if results != nil {
		sb.WriteString(e.renderer.Results(fset, results))
	}

	return sb.String()
}

func (e *Extractor) buildFuncDeclSignature(fset *token.FileSet, decl *ast.FuncDecl) string {
	var prefix strings.Builder
	prefix.WriteString("func ")

	if decl.Recv != nil {
		prefix.WriteString(e.renderer.FieldList(fset, decl.Recv, "(", ")"))
		prefix.WriteString(" ")
	}

	prefix.WriteString(decl.Name.Name)
	return e.buildSignature(fset, prefix.String(), decl.Type, decl.Type.Results)
}

func (e *Extractor) buildFuncLitSignature(fset *token.FileSet, ft *ast.FuncType) string {
	return e.buildSignature(fset, "func", ft, ft.Results)
}

func (e *Extractor) buildMethodSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return e.buildSignature(fset, name, ft, ft.Results)
}

func (e *Extractor) buildStructFieldSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return e.buildSignature(fset, name+" func", ft, ft.Results)
}
