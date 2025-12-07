package format

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

func parseGoCode(t *testing.T, code string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	require.NoError(t, err)
	return fset, file
}

func TestParamGroups_InterfaceMethods(t *testing.T) {
	code := `package test

import (
	"context"
	"database/sql"
)

type Repository interface {
	Create(ctx context.Context, tx *sql.Tx, id int, name string) error
	Update(ctx context.Context, data []byte) error
	Delete(ctx context.Context, tx *sql.Tx) error
}
`
	fset, file := parseGoCode(t, code)

	tests := []struct {
		name         string
		paramGroups  [][]string
		methodName   string
		wantContains []string
	}{
		{
			name: "TwoGroups_ContextAndTx",
			paramGroups: [][]string{
				{"context.Context", "*sql.Tx"},
				{"context.Context"},
			},
			methodName: "Create",
			wantContains: []string{
				"Create(",
				"context.Context",
				"*sql.Tx",
			},
		},
		{
			name: "SingleGroup_Context",
			paramGroups: [][]string{
				{"context.Context"},
			},
			methodName: "Update",
			wantContains: []string{
				"Update(",
				"context.Context",
			},
		},
		{
			name: "NoMatchingGroups",
			paramGroups: [][]string{
				{"io.Reader", "io.Writer"},
			},
			methodName:   "Delete",
			wantContains: []string{"Delete("},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Settings{
				MaxLineLen:           50,
				TabWidth:             4,
				PackInterfaceMethods: true,
				ParamGroups:          tt.paramGroups,
			}

			r := render.New(cfg.TabWidth)
			builder := NewBuilder(cfg, r)
			strategy := NewParamGroupStrategy(cfg, builder)

			var typeSpec *ast.TypeSpec
			for _, decl := range file.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
					typeSpec = genDecl.Specs[0].(*ast.TypeSpec)
					break
				}
			}
			require.NotNil(t, typeSpec, "TypeSpec not found")
			iface := typeSpec.Type.(*ast.InterfaceType)

			var method *ast.Field
			for _, m := range iface.Methods.List {
				if len(m.Names) > 0 && m.Names[0].Name == tt.methodName {
					method = m
					break
				}
			}
			require.NotNil(t, method, "Method %s not found", tt.methodName)

			ft := method.Type.(*ast.FuncType)
			sig := &domain.Signature{
				Name:              method.Names[0].Name,
				Start:             method.Names[0].Pos(),
				End:               ft.Params.End(),
				DiagPos:           ft.Params.Closing,
				OneLineText:       tt.methodName + "(ctx context.Context, tx *sql.Tx, id int, name string) error",
				FuncType:          ft,
				IsInterfaceMethod: true,
			}

			newText, applied := strategy.Apply(fset, sig)

			if len(tt.paramGroups) > 0 {
				assert.True(t, applied, "Strategy should apply when param groups are configured")
				for _, want := range tt.wantContains {
					assert.Contains(t, newText, want, "Result should contain: %s", want)
				}
			}
		})
	}
}

func TestParamGroups_GroupMatching(t *testing.T) {
	code := `package test

import "context"

type Service interface {
	Process(ctx context.Context, id int, name string) error
}
`
	fset, file := parseGoCode(t, code)

	cfg := config.Settings{
		MaxLineLen: 50,
		TabWidth:   4,
		ParamGroups: [][]string{
			{"context.Context"},
		},
	}

	r := render.New(cfg.TabWidth)
	builder := NewBuilder(cfg, r)

	var typeSpec *ast.TypeSpec
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			typeSpec = genDecl.Specs[0].(*ast.TypeSpec)
			break
		}
	}
	require.NotNil(t, typeSpec, "TypeSpec not found")
	iface := typeSpec.Type.(*ast.InterfaceType)
	method := iface.Methods.List[0]
	ft := method.Type.(*ast.FuncType)

	sig := &domain.Signature{
		Name:              "Process",
		Start:             method.Names[0].Pos(),
		End:               ft.Params.End(),
		FuncType:          ft,
		IsInterfaceMethod: true,
	}

	result := builder.BuildReformattedSignature(fset, sig)

	assert.Contains(t, result, "Process(")
	assert.Contains(t, result, "context.Context")
	assert.Contains(t, result, "id int")
	assert.Contains(t, result, "name string")
}

func TestParamGroups_MultipleGroupsBreak(t *testing.T) {
	code := `package test

import (
	"context"
	"database/sql"
)

type Store interface {
	Execute(ctx context.Context, tx *sql.Tx, query string, args []interface{}) error
}
`
	fset, file := parseGoCode(t, code)

	cfg := config.Settings{
		MaxLineLen: 80,
		TabWidth:   4,
		ParamGroups: [][]string{
			{"context.Context", "*sql.Tx"},
		},
	}

	r := render.New(cfg.TabWidth)
	builder := NewBuilder(cfg, r)

	var typeSpec *ast.TypeSpec
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			typeSpec = genDecl.Specs[0].(*ast.TypeSpec)
			break
		}
	}
	require.NotNil(t, typeSpec, "TypeSpec not found")
	iface := typeSpec.Type.(*ast.InterfaceType)
	method := iface.Methods.List[0]
	ft := method.Type.(*ast.FuncType)

	sig := &domain.Signature{
		Name:              "Execute",
		Start:             method.Names[0].Pos(),
		End:               ft.Params.End(),
		FuncType:          ft,
		IsInterfaceMethod: true,
		OneLineText:       "Execute(ctx context.Context, tx *sql.Tx, query string, args []interface{}) error",
	}

	result := builder.BuildReformattedSignature(fset, sig)

	assert.Contains(t, result, "Execute(")
	assert.Contains(t, result, "context.Context")
	assert.Contains(t, result, "*sql.Tx")
}

func TestParamGroups_EmptyInterface(t *testing.T) {
	code := `package test

type Empty interface {
	NoParams() error
}
`
	fset, file := parseGoCode(t, code)

	cfg := config.Settings{
		MaxLineLen: 80,
		TabWidth:   4,
		ParamGroups: [][]string{
			{"context.Context"},
		},
	}

	r := render.New(cfg.TabWidth)
	builder := NewBuilder(cfg, r)

	var typeSpec *ast.TypeSpec
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			typeSpec = genDecl.Specs[0].(*ast.TypeSpec)
			break
		}
	}
	require.NotNil(t, typeSpec, "TypeSpec not found")
	iface := typeSpec.Type.(*ast.InterfaceType)
	method := iface.Methods.List[0]
	ft := method.Type.(*ast.FuncType)

	sig := &domain.Signature{
		Name:              "NoParams",
		Start:             method.Names[0].Pos(),
		End:               ft.Params.End(),
		FuncType:          ft,
		IsInterfaceMethod: true,
		OneLineText:       "NoParams() error",
	}

	result := builder.BuildReformattedSignature(fset, sig)

	assert.Contains(t, result, "NoParams()")
	assert.Contains(t, result, "error")
}
