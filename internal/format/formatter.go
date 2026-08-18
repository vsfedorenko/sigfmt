package format

import (
	"go/token"
	"strings"

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

// check applies the strategies and returns the winning output, or "" when
// no strategy applies. Guard-free: the no-op guard lives in Check.
func (f *Formatter) check(fset *token.FileSet, sig *domain.Signature) string {
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

// originalText returns the signature's source text between sig.Start and
// sig.End. It is the no-op guard input: a strategy whose output equals the
// existing source must not produce a diagnostic.
func (f *Formatter) originalText(readFile func(filename string) ([]byte, error), fset *token.FileSet, sig *domain.Signature) (string, bool) {
	if readFile == nil {
		return "", false
	}
	pos := fset.Position(sig.Start)
	end := fset.Position(sig.End)
	if end.Offset < pos.Offset || pos.Filename == "" {
		return "", false
	}
	src, err := readFile(pos.Filename)
	if err != nil || len(src) < end.Offset {
		return "", false
	}
	return string(src[pos.Offset:end.Offset]), true
}

// Check determines if a signature needs formatting changes: it applies
// the strategies and returns the new formatted text, or "" when no
// change is warranted. Two guards suppress the diagnostic:
//
//   - comment preservation: the renderer rebuilds the signature from the
//     AST alone, so any comment inside the rewritten range would be
//     silently dropped. Signatures whose original source contains a
//     comment are left untouched.
//   - no-op guard: a strategy whose output equals the existing source
//     (modulo the first line's leading indentation) must not be reported.
//
// readFile is analysis.Pass.ReadFile in production and a file loader in
// tests; when the source cannot be read the guards fail open (the
// diagnostic is kept).
func (f *Formatter) Check(readFile func(filename string) ([]byte, error), fset *token.FileSet, sig *domain.Signature) string {
	newText := f.check(fset, sig)
	if newText == "" {
		return ""
	}
	original, ok := f.originalText(readFile, fset, sig)
	if !ok {
		return newText // cannot verify — keep the diagnostic
	}
	if containsComment(original) {
		return "" // rewriting would drop the comment — leave the signature alone
	}
	if normalizeFirstLineIndent(original) == normalizeFirstLineIndent(newText) {
		return "" // no-op fix — the source is already in the target shape
	}
	return newText
}

// containsComment reports whether the signature's original source text
// contains a line (`//`) or block (`/*`) comment marker. Signatures contain
// only identifiers, types, and punctuation — never string literals — so the
// textual scan cannot false-positive.
func containsComment(src string) bool {
	for i := 0; i+1 < len(src); i++ {
		if src[i] == '/' && (src[i+1] == '/' || src[i+1] == '*') {
			return true
		}
	}
	return false
}

// normalizeFirstLineIndent trims leading whitespace of the first line: the
// builder re-derives indentation from the file, and a signature starting at
// column 0 vs column 1 renders identically apart from that prefix.
func normalizeFirstLineIndent(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimLeft(s[:i], " 	") + s[i:]
	}
	return strings.TrimLeft(s, " 	")
}
