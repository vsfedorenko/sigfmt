package format

import (
	"go/ast"
	"go/token"

	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/pkg/source"
	"github.com/vsfedorenko/sigfmt/internal/pkg/text"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

// Strategy defines an algorithm for determining if and how a signature should be formatted.
type Strategy interface {
	// Name returns the name of the strategy (for debugging/logging).
	Name() string
	// Apply checks if this strategy applies to the given signature.
	// If it applies, it returns the new formatted text and true.
	// If not, it returns empty string and false.
	Apply(fset *token.FileSet, sig *domain.Signature) (string, bool)
}

// --- Concrete Strategies ---

// CollapseStrategy checks if the signature fits on a single line.
type CollapseStrategy struct {
	config   config.Settings
	renderer *render.Renderer
}

func NewCollapseStrategy(cfg config.Settings, r *render.Renderer) *CollapseStrategy {
	return &CollapseStrategy{config: cfg, renderer: r}
}

func (s *CollapseStrategy) Name() string { return "Collapse" }

func (s *CollapseStrategy) Apply(fset *token.FileSet, sig *domain.Signature) (string, bool) {
	startLine := fset.Position(sig.Start).Line
	endLine := fset.Position(sig.End).Line

	baseIndent := source.GetIndent(fset, sig.Start)
	baseIndentLen := text.VisualLength(baseIndent, s.config.TabWidth)
	oneLineVisualLen := text.VisualLength(sig.OneLineText, s.config.TabWidth)
	// The suffix (struct tag) stays on the collapsed line — its width and
	// its separating space count against the budget.
	suffixLen := text.VisualLength(sig.SuffixText, s.config.TabWidth)

	if baseIndentLen+oneLineVisualLen+suffixLen <= s.config.MaxLineLen {
		if startLine != endLine {
			return sig.OneLineText, true
		}
		return "", true
	}
	return "", false
}

// ParamGroupStrategy enforces parameter grouping defined in settings.
type ParamGroupStrategy struct {
	config  config.Settings
	builder *Builder
}

func NewParamGroupStrategy(cfg config.Settings, b *Builder) *ParamGroupStrategy {
	return &ParamGroupStrategy{config: cfg, builder: b}
}

func (s *ParamGroupStrategy) Name() string { return "ParamGroup" }

func (s *ParamGroupStrategy) Apply(fset *token.FileSet, sig *domain.Signature) (string, bool) {
	if len(s.config.ParamGroups) > 0 {
		return s.builder.BuildReformattedSignature(fset, sig), true
	}
	return "", false
}

// DefinitionPackingStrategy handles packing for interfaces and structs.
type DefinitionPackingStrategy struct {
	config  config.Settings
	builder *Builder
}

func NewDefinitionPackingStrategy(cfg config.Settings, b *Builder) *DefinitionPackingStrategy {
	return &DefinitionPackingStrategy{config: cfg, builder: b}
}

func (s *DefinitionPackingStrategy) Name() string { return "DefinitionPacking" }

func (s *DefinitionPackingStrategy) Apply(fset *token.FileSet, sig *domain.Signature) (string, bool) {
	if sig.IsInterfaceMethod && s.config.PackInterfaceMethods {
		return s.builder.BuildReformattedSignature(fset, sig), true
	}
	if sig.IsStructField && s.config.PackStructFields {
		return s.builder.BuildReformattedSignature(fset, sig), true
	}
	return "", false
}

// ConsistencyStrategy enforces standard formatting for regular functions (staircase check).
type ConsistencyStrategy struct {
	builder *Builder
}

func NewConsistencyStrategy(b *Builder) *ConsistencyStrategy {
	return &ConsistencyStrategy{builder: b}
}

func (s *ConsistencyStrategy) Name() string { return "Consistency" }

func (s *ConsistencyStrategy) Apply(fset *token.FileSet, sig *domain.Signature) (string, bool) {
	// Only applies if parameters are NOT on separate lines (mixed style).
	if !areParamsOnSeparateLines(fset, sig.FuncType.Params.List) {
		return s.builder.BuildReformattedSignature(fset, sig), true
	}
	return "", false
}

// Helper for ConsistencyStrategy
func getFieldStartPos(f *ast.Field) token.Pos {
	if len(f.Names) > 0 {
		return f.Names[0].Pos()
	}
	// Fallback for unnamed fields, though parameters are typically named.
	return f.Type.Pos()
}

func areParamsOnSeparateLines(fset *token.FileSet, params []*ast.Field) bool {
	if len(params) <= 1 {
		return true
	}
	prevLine := fset.Position(getFieldStartPos(params[0])).Line
	for i := 1; i < len(params); i++ {
		currentLine := fset.Position(getFieldStartPos(params[i])).Line
		if currentLine == prevLine {
			return false
		}
		prevLine = currentLine
	}
	return true
}
