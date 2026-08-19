package astinfo

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/pkg/field"
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
	return e.createFuncSignature(
		fset,
		decl.Type,
		decl.Type.Func,
		e.buildFuncDeclSignature(fset, decl),
		decl.Recv,
		decl.Name.Name,
	)
}

// FuncLit extracts info from a function literal.
func (e *Extractor) FuncLit(fset *token.FileSet, lit *ast.FuncLit) *domain.Signature {
	return e.createFuncSignature(
		fset,
		lit.Type,
		lit.Type.Func,
		e.buildFuncLitSignature(fset, lit.Type),
		nil,
		"",
	)
}

// createFuncSignature is a helper for creating function signatures.
func (e *Extractor) createFuncSignature(fset *token.FileSet, ft *ast.FuncType, start token.Pos, oneLineText string, receiver *ast.FieldList, name string) *domain.Signature {
	end := ft.Params.End()
	if ft.Results != nil {
		end = ft.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &domain.Signature{
		Start:       start,
		End:         end,
		DiagPos:     computeDiagPos(ft.Params, ft.Results),
		OneLineText: oneLineText,
		FuncType:    ft,
		Receiver:    receiver,
		Name:        name,
	}
}

// Method extracts info from an interface method. Interface methods are
// always declared with a single name.
func (e *Extractor) Method(fset *token.FileSet, names []*ast.Ident, ft *ast.FuncType) *domain.Signature {
	if len(names) == 0 {
		return nil
	}
	name := names[0].Name
	sig := e.createFieldSignature(fset, names[0].Pos(), ft, name, e.buildMethodSignature(fset, name, ft))
	if sig != nil {
		sig.IsInterfaceMethod = true
	}
	return sig
}

// StructField extracts info from a func-typed struct field. A field may be
// declared with several names (`Handler, Fallback func(...)`); all of them
// are rendered and rewritten together — dropping any would silently delete
// a struct field.
func (e *Extractor) StructField(fset *token.FileSet, names []*ast.Ident, ft *ast.FuncType) *domain.Signature {
	if len(names) == 0 {
		return nil
	}
	rendered := field.RenderNames(names)
	sig := e.createFieldSignature(fset, names[0].Pos(), ft, rendered, e.buildStructFieldSignature(fset, rendered, ft))
	if sig != nil {
		sig.IsStructField = true
	}
	return sig
}

// createFieldSignature is a helper for creating signatures from named fields
// (methods/struct fields). name is the rendered field name(s): a single
// identifier for interface methods, the comma-joined list for struct fields.
func (e *Extractor) createFieldSignature(fset *token.FileSet, start token.Pos, ft *ast.FuncType, name, oneLineText string) *domain.Signature {
	end := ft.Params.End()
	if ft.Results != nil {
		end = ft.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &domain.Signature{
		Start:       start,
		End:         end,
		DiagPos:     computeDiagPos(ft.Params, ft.Results),
		OneLineText: oneLineText,
		FuncType:    ft,
		Receiver:    nil,
		Name:        name,
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
