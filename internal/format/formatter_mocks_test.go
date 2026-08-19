package format

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vsfedorenko/sigfmt/internal/domain"
)

// The Formatter's contract with its strategies: first applied wins, the
// rest are not consulted. Pinned with mockery-generated mocks (the concrete
// strategies are tested elsewhere; here we test the ORCHESTRER).
func TestFormatter_FirstAppliedStrategyWins(t *testing.T) {
	first := NewMockStrategy(t)
	second := NewMockStrategy(t)

	first.EXPECT().Name().Return("first").Maybe()
	first.EXPECT().Apply(mock.Anything, mock.Anything).Return("formatted", true)
	// second.Apply must never be called: no EXPECT registered

	f := &Formatter{strategies: []Strategy{first, second}}
	out := f.check(testFset(t), testSig(t))
	assert.Equal(t, "formatted", out)
}

func TestFormatter_NoStrategyApplies(t *testing.T) {
	s := NewMockStrategy(t)
	s.EXPECT().Apply(mock.Anything, mock.Anything).Return("", false)

	f := &Formatter{strategies: []Strategy{s}}
	out := f.check(testFset(t), testSig(t))
	assert.Empty(t, out, "no strategy applied must yield empty text")
}

func TestFormatter_NilParamsShortCircuits(t *testing.T) {
	s := NewMockStrategy(t)
	// no EXPECT: Apply must not be called when FuncType.Params is nil

	f := &Formatter{strategies: []Strategy{s}}
	sig := &domain.Signature{FuncType: &ast.FuncType{}}
	assert.Empty(t, f.check(testFset(t), sig))
}

// --- helpers ---

func testFset(t *testing.T) *token.FileSet {
	t.Helper()
	fset := token.NewFileSet()
	src := "package p\n\nfunc F(a int) error { return nil }\n"
	_, err := parser.ParseFile(fset, "a.go", src, 0)
	require.NoError(t, err)
	return fset
}

func testSig(t *testing.T) *domain.Signature {
	t.Helper()
	return &domain.Signature{
		FuncType: &ast.FuncType{Params: &ast.FieldList{}},
	}
}
