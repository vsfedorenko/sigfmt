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

// parseFirstFuncSig parses src and extracts the first FuncDecl signature.
func parseFirstFuncSig(t *testing.T, src string) (*token.FileSet, *domain.Signature, string) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, 0)
	require.NoError(t, err)
	fd := f.Decls[0].(*ast.FuncDecl)
	cfg := config.New(nil)
	renderer := render.New(cfg.TabWidth)
	extractor := astinfo.New(renderer)
	return fset, extractor.FuncDecl(fset, fd), src
}

// readFileOf returns a readFile func serving the parsed source.
func readFileOf(src string) func(string) ([]byte, error) {
	return func(string) ([]byte, error) { return []byte(src), nil }
}

// An already-correctly formatted multi-line signature (too long to collapse,
// params packed per the builder's target shape) must NOT produce a diagnostic.
func TestCheckWithSource_NoOpGuard(t *testing.T) {
	src := "package p\n\nfunc F(\n\tfirstParameterWithAVeryLongNameOne, secondParameterWithAVeryLongNameTwo, thirdParameterWithAVeryLongName string,\n\tfourthParameterWithAVeryLongName, fifthParameterWithAVeryLongName int,\n) error {\n\treturn nil\n}\n"
	fset, sig, source := parseFirstFuncSig(t, src)
	cfg := config.New(nil)
	f := New(cfg, render.New(cfg.TabWidth))

	got := f.Check(testLoader(fset, source), sig)
	assert.Empty(t, got, "no-op guard failed: got fix\n%s", got)
}

// A genuinely collapsible signature must still produce the collapsed fix.
func TestCheckWithSource_CollapseStillReported(t *testing.T) {
	src := "package p\n\nfunc F(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n"
	fset, sig, source := parseFirstFuncSig(t, src)
	cfg := config.New(nil)
	f := New(cfg, render.New(cfg.TabWidth))

	got := f.Check(testLoader(fset, source), sig)
	require.NotEmpty(t, got, "expected collapse fix, got none")
	assert.Equal(t, "func F(a int, b string) error", got, "unexpected fix")
}

// When readFile cannot serve the file, the diagnostic must be kept
// (fail-open — never suppress legitimate findings).
func TestCheckWithSource_UnreadableSourceKeepsDiagnostic(t *testing.T) {
	src := "package p\n\nfunc F(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n"
	fset, sig, _ := parseFirstFuncSig(t, src)
	cfg := config.New(nil)
	f := New(cfg, render.New(cfg.TabWidth))

	got := f.Check(testLoaderFailing(fset), sig)
	assert.NotEmpty(t, got, "unreadable source must keep the diagnostic")
}

var errUnavailable = errUnavailableError{}

type errUnavailableError struct{}

func (errUnavailableError) Error() string { return "unavailable" }
