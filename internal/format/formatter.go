package format

import (
	"go/token"

	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

const (
	DiagnosticMessage = "Signature can be formatted more compactly"
	FixMessage        = "Format signature"
)

// Formatter applies formatting strategies to function signatures.
type Formatter struct {
	strategies []Strategy
}

// New creates a new Formatter with configured strategies.
func New(cfg config.Settings, renderer *render.Renderer) *Formatter {
	builder := NewBuilder(cfg, renderer)

	return &Formatter{
		strategies: []Strategy{
			NewCollapseStrategy(cfg, renderer),
			NewParamGroupStrategy(cfg, builder),
			NewDefinitionPackingStrategy(cfg, builder),
			NewConsistencyStrategy(builder),
		},
	}
}

// Check determines if a signature needs formatting changes.
// Returns the new formatted text if changes are needed, or empty string otherwise.
func (f *Formatter) Check(fset *token.FileSet, sig *domain.Signature) string {
	if sig.FuncType == nil || sig.FuncType.Params == nil {
		return ""
	}

	for _, strategy := range f.strategies {
		newText, applied := strategy.Apply(fset, sig)
		if applied {
			return newText
		}
	}

	return ""
}
