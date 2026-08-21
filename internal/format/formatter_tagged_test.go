package format

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vsfedorenko/sigfmt/internal/astinfo"
	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

// formatter_tagged_test.go pins the SuffixText branch of (*Formatter).check
// (the struct-tag budget of sigfmt 1.4.0) at the unit layer: a tagged field
// already declared as a compact one-liner is left alone regardless of the
// tag's own width, while a tagged multi-line field keeps flowing through the
// strategies. Before this file the branch was reachable only through
// analysistest fixtures, which do not count toward the package's unit
// coverage.

// parseFirstTaggedFieldSig parses src and extracts the first struct field
// with a func type and a tag, the way the linter's field walker does.
func parseFirstTaggedFieldSig(t *testing.T, src string) (*token.FileSet, *domain.Signature, string) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, 0)
	require.NoError(t, err)

	var sig *domain.Signature
	// Find the field directly: first TypeSpec -> StructType -> tagged field.
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range st.Fields.List {
				ft, ok := field.Type.(*ast.FuncType)
				if !ok || field.Tag == nil {
					continue
				}
				cfg := config.New(nil)
				renderer := render.New(cfg.TabWidth)
				extractor := astinfo.New(renderer)
				sig = extractor.StructField(fset, field.Names, ft)
				// Mirror the linter's SuffixText wiring.
				sig.SuffixText = " " + field.Tag.Value
				return fset, sig, src
			}
		}
	}
	t.Fatal("no tagged func-typed struct field found in fixture")
	return nil, nil, ""
}

// A tagged one-liner within the limit (signature alone fits) is maximally
// compact: no diagnostic, even when rewriting could theoretically move the
// tag. The overflow in the too-long case below is the TAG's doing.
func TestCheckTagged_CompactOneLinerUntouched(t *testing.T) {
	src := "package p\n\ntype S struct {\n\tH func(a int, b string) error `json:\"h\" validate:\"required,min=1,max=64,oneof=a b c d e f g h i j k l m n o p\"`\n}\n"
	fset, sig, source := parseFirstTaggedFieldSig(t, src)
	cfg := config.New(nil)
	f := New(cfg, render.New(cfg.TabWidth))

	got := f.Check(readFileOf(source), fset, sig)
	assert.Empty(t, got, "compact tagged one-liner must stay untouched, got fix:\n%s", got)
}

// A tagged field split across lines still flows through the strategies
// (collapse or pack) — the early return must not swallow real diagnostics.
func TestCheckTagged_MultiLineStillReported(t *testing.T) {
	src := "package p\n\ntype S struct {\n\tH func(\n\t\ta int,\n\t\tb string,\n\t) error `json:\"h\"`\n}\n"
	fset, sig, source := parseFirstTaggedFieldSig(t, src)
	cfg := config.New(nil)
	f := New(cfg, render.New(cfg.TabWidth))

	got := f.Check(readFileOf(source), fset, sig)
	assert.NotEmpty(t, got, "tagged multi-line field must produce a fix")
}

// A tagged one-liner whose SIGNATURE alone busts the limit is not compact:
// the early return must not apply, the strategies unpack it.
func TestCheckTagged_HugeOneLinerStillReported(t *testing.T) {
	src := "package p\n\ntype S struct {\n\tH func(ctx interface{}, request interface{}, response interface{}, options map[string]interface{}, extra []byte, verbose bool, timeout int, retries int) (result string, err error) `json:\"h\"`\n}\n"
	fset, sig, source := parseFirstTaggedFieldSig(t, src)
	cfg := config.New(nil)
	f := New(cfg, render.New(cfg.TabWidth))

	got := f.Check(readFileOf(source), fset, sig)
	assert.NotEmpty(t, got, "huge tagged one-liner must produce a fix")
}

// mustParseFsetSig parses src and returns its fset and first FuncDecl
// signature (collapse/pack probe helper).
func mustParseFsetSig(t *testing.T, src string) (*token.FileSet, *domain.Signature) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, 0)
	require.NoError(t, err)
	fd, ok := f.Decls[0].(*ast.FuncDecl)
	require.True(t, ok, "fixture must start with a FuncDecl")
	cfg := config.New(nil)
	extractor := astinfo.New(render.New(cfg.TabWidth))
	return fset, extractor.FuncDecl(fset, fd)
}

const packableSrc = "package p\n\nfunc B(\n\tfirstParameterWithAVeryLongNameOne string,\n\tsecondParameterWithAVeryLongNameTwo string,\n\tthirdParameterWithAVeryLongNameThree string,\n\tfourthParameterWithAVeryLongNameFour int,\n) error { return nil }\n"

// Strategy Name() methods feed diagnostics and debugging surfaces; pin the
// concrete strings so a rename cannot slip in silently.
func TestStrategyNames(t *testing.T) {
	cfg := config.New(nil)
	assert.Equal(t, "Collapse", (&CollapseStrategy{}).Name())
	assert.Equal(t, "ParamGroup", (&ParamGroupStrategy{config: cfg}).Name())
	assert.Equal(t, "DefinitionPacking", (&DefinitionPackingStrategy{}).Name())
	assert.Equal(t, "Consistency", (&ConsistencyStrategy{}).Name())
}
