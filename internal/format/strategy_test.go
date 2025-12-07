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

func newTestFileSet() (*token.FileSet, *token.File) {
	fset := token.NewFileSet()
	file := fset.AddFile("test.go", -1, 1000)
	for i := 0; i < 10; i++ {
		file.AddLine(i * 10)
	}
	return fset, file
}

func TestCollapseStrategy(t *testing.T) {
	cfg := config.Settings{MaxLineLen: 20, TabWidth: 4}
	r := render.New(cfg.TabWidth)
	strategy := NewCollapseStrategy(cfg, r)
	fset, file := newTestFileSet()

	tests := []struct {
		name        string
		sig         *domain.Signature
		wantApplied bool
		wantNewText bool
	}{
		{
			name: "FitsAndNeedsCollapse",
			sig: &domain.Signature{
				OneLineText: "func Foo()",
				Start:       file.Pos(0),
				End:         file.Pos(20),
				FuncType:    &ast.FuncType{Params: &ast.FieldList{}},
			},
			wantApplied: true,
			wantNewText: true,
		},
		{
			name: "FitsAndAlreadyCollapsed",
			sig: &domain.Signature{
				OneLineText: "func Bar()",
				Start:       file.Pos(10),
				End:         file.Pos(15),
				FuncType:    &ast.FuncType{Params: &ast.FieldList{}},
			},
			wantApplied: true,
			wantNewText: false,
		},
		{
			name: "DoesNotFit",
			sig: &domain.Signature{
				OneLineText: "func Baz(veryLongParameterName int) error",
				Start:       file.Pos(0),
				End:         file.Pos(90),
				FuncType:    &ast.FuncType{Params: &ast.FieldList{}},
			},
			wantApplied: false,
			wantNewText: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newText, applied := strategy.Apply(fset, tt.sig)
			assert.Equal(t, tt.wantApplied, applied)
			assert.Equal(t, tt.wantNewText, newText != "")
		})
	}
}

func TestParamGroupStrategy(t *testing.T) {
	fset, _ := newTestFileSet()
	r := render.New(4)

	tests := []struct {
		name        string
		cfg         config.Settings
		wantApplied bool
	}{
		{
			name:        "NoParamGroupsConfigured",
			cfg:         config.Settings{ParamGroups: [][]string{}},
			wantApplied: false,
		},
		{
			name:        "ParamGroupsConfigured",
			cfg:         config.Settings{ParamGroups: [][]string{{"context.Context"}}},
			wantApplied: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := &domain.Signature{FuncType: &ast.FuncType{Params: &ast.FieldList{}}}
			builder := NewBuilder(tt.cfg, r)
			strategy := NewParamGroupStrategy(tt.cfg, builder)

			_, applied := strategy.Apply(fset, sig)
			assert.Equal(t, tt.wantApplied, applied)
		})
	}
}

func TestDefinitionPackingStrategy(t *testing.T) {
	fset, _ := newTestFileSet()
	r := render.New(4)

	tests := []struct {
		name        string
		cfg         config.Settings
		isInterface bool
		isStruct    bool
		wantApplied bool
	}{
		{
			name:        "InterfaceMethod_PackingEnabled",
			cfg:         config.Settings{PackInterfaceMethods: true},
			isInterface: true,
			wantApplied: true,
		},
		{
			name:        "StructField_PackingEnabled",
			cfg:         config.Settings{PackStructFields: true},
			isStruct:    true,
			wantApplied: true,
		},
		{
			name:        "InterfaceMethod_PackingDisabled",
			cfg:         config.Settings{PackInterfaceMethods: false},
			isInterface: true,
			wantApplied: false,
		},
		{
			name:        "RegularFunction",
			cfg:         config.Settings{},
			wantApplied: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := &domain.Signature{
				IsInterfaceMethod: tt.isInterface,
				IsStructField:     tt.isStruct,
				FuncType:          &ast.FuncType{Params: &ast.FieldList{}},
			}
			builder := NewBuilder(tt.cfg, r)
			strategy := NewDefinitionPackingStrategy(tt.cfg, builder)
			_, applied := strategy.Apply(fset, sig)
			assert.Equal(t, tt.wantApplied, applied)
		})
	}
}

func TestConsistencyStrategy(t *testing.T) {
	fset, file := newTestFileSet()
	r := render.New(4)
	builder := NewBuilder(config.Settings{}, r)
	strategy := NewConsistencyStrategy(builder)

	tests := []struct {
		name        string
		params      []*ast.Field
		wantApplied bool
	}{
		{
			name: "MixedParams",
			params: []*ast.Field{
				{Names: []*ast.Ident{{Name: "a", NamePos: file.Pos(0)}}, Type: &ast.Ident{Name: "int", NamePos: file.Pos(1)}},
				{Names: []*ast.Ident{{Name: "b", NamePos: file.Pos(2)}}, Type: &ast.Ident{Name: "int", NamePos: file.Pos(3)}},
				{Names: []*ast.Ident{{Name: "c", NamePos: file.Pos(10)}}, Type: &ast.Ident{Name: "string", NamePos: file.Pos(11)}},
			},
			wantApplied: true,
		},
		{
			name: "ParamsOnSeparateLines",
			params: []*ast.Field{
				{Names: []*ast.Ident{{Name: "a", NamePos: file.Pos(0)}}, Type: &ast.Ident{Name: "int", NamePos: file.Pos(1)}},
				{Names: []*ast.Ident{{Name: "b", NamePos: file.Pos(10)}}, Type: &ast.Ident{Name: "int", NamePos: file.Pos(11)}},
				{Names: []*ast.Ident{{Name: "c", NamePos: file.Pos(20)}}, Type: &ast.Ident{Name: "string", NamePos: file.Pos(21)}},
			},
			wantApplied: false,
		},
		{
			name: "SingleParam",
			params: []*ast.Field{
				{Names: []*ast.Ident{{Name: "a", NamePos: file.Pos(0)}}, Type: &ast.Ident{Name: "int", NamePos: file.Pos(1)}},
			},
			wantApplied: false,
		},
		{
			name:        "NoParams",
			params:      []*ast.Field{},
			wantApplied: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := &domain.Signature{
				FuncType: &ast.FuncType{Params: &ast.FieldList{List: tt.params}},
			}
			_, applied := strategy.Apply(fset, sig)
			assert.Equal(t, tt.wantApplied, applied)
		})
	}
}
