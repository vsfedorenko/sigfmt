package sigfmt

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/vsfedorenko/sigfmt/internal/astinfo"
	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/format"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

const (
	analyzerName = "sigfmt"
	analyzerDoc  = "Checks if multi-line function signatures can be collapsed to one line or reformatted more compactly"
)

func init() {
	register.Plugin(analyzerName, New)
}

// PluginLineWrap implements the register.LinterPlugin interface.
type PluginLineWrap struct {
	settings config.Settings
}

// New returns a new instance of the sigfmt linter plugin.
func New(settings any) (register.LinterPlugin, error) {
	return &PluginLineWrap{
		settings: config.New(settings),
	}, nil
}

// BuildAnalyzers returns the analysis.Analyzer definition.
func (p *PluginLineWrap) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{{
		Name: analyzerName,
		Doc:  analyzerDoc,
		Run:  p.run,
	}}, nil
}

// GetLoadMode returns the LoadMode required for the analyzer.
func (p *PluginLineWrap) GetLoadMode() string {
	return register.LoadModeSyntax
}

// run executes the analysis pass.
func (p *PluginLineWrap) run(pass *analysis.Pass) (any, error) {
	renderer := render.New(p.settings.TabWidth)
	extractor := astinfo.New(renderer)
	formatter := format.New(p.settings, renderer)

	for _, f := range pass.Files {
		if isGenerated(pass, f) {
			continue
		}
		if p.settings.IgnoreTests && strings.HasSuffix(pass.Fset.Position(f.Pos()).Filename, "_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if sig := extractor.FuncDecl(pass.Fset, x); sig != nil {
					p.checkAndReport(pass, sig, formatter)
				}
			case *ast.FuncLit:
				if sig := extractor.FuncLit(pass.Fset, x); sig != nil {
					p.checkAndReport(pass, sig, formatter)
				}
			case *ast.TypeSpec:
				p.handleTypeSpec(pass, x, extractor, formatter)
			}
			return true
		})
	}
	return nil, nil
}

// generatedFilePrefix is the conventional machine-generated marker line
// (https://go.dev/s/generatedcode): a comment appearing before the package
// clause and matching /^ Code generated .* DO NOT EDIT\.$/.
const generatedFilePrefix = "Code generated"

// generatedFileSuffix terminates the required "DO NOT EDIT." sentence.
const generatedFileSuffix = "DO NOT EDIT."

// isGenerated reports whether the file is machine-generated, following the
// same convention as golangci-lint and go/analysis' analysistest: a comment
// before the package clause containing "Code generated ... DO NOT EDIT.".
// Generated files (*.pb.go, mocks, zz_*) are not hand-maintained, so
// reporting formatting diagnostics on them is noise.
func isGenerated(pass *analysis.Pass, f *ast.File) bool {
	for _, group := range f.Comments {
		if group.Pos() >= f.Package {
			break // comments after the package clause are not the header
		}
		for _, c := range group.List {
			text := strings.TrimSpace(c.Text)
			if strings.HasPrefix(text, "//") {
				text = strings.TrimSpace(text[2:])
			}
			if strings.HasPrefix(text, generatedFilePrefix) &&
				strings.HasSuffix(text, generatedFileSuffix) {
				return true
			}
		}
	}
	return false
}

func (p *PluginLineWrap) handleTypeSpec(pass *analysis.Pass, spec *ast.TypeSpec, extractor *astinfo.Extractor, formatter *format.Formatter) {
	switch t := spec.Type.(type) {
	case *ast.InterfaceType:
		p.handleFields(pass, t.Methods, extractor.Method, formatter)
	case *ast.StructType:
		p.handleFields(pass, t.Fields, extractor.StructField, formatter)
	}
}

func (p *PluginLineWrap) handleFields(pass *analysis.Pass, fields *ast.FieldList, extract func(*token.FileSet, *ast.Ident, *ast.FuncType) *domain.Signature, formatter *format.Formatter) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			continue
		}
		ft, ok := field.Type.(*ast.FuncType)
		if !ok || ft.Params == nil {
			continue
		}
		if sig := extract(pass.Fset, field.Names[0], ft); sig != nil {
			p.checkAndReport(pass, sig, formatter)
		}
	}
}

func (p *PluginLineWrap) checkAndReport(pass *analysis.Pass, sig *domain.Signature, formatter *format.Formatter) {
	newText := formatter.CheckWithSource(pass.ReadFile, pass.Fset, sig)
	if newText != "" {
		p.report(pass, sig, newText)
	}
}

func (p *PluginLineWrap) report(pass *analysis.Pass, sig *domain.Signature, newText string) {
	pass.Report(analysis.Diagnostic{
		Pos:     sig.DiagPos,
		End:     sig.DiagPos,
		Message: format.DiagnosticMessage,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: format.FixMessage,
			TextEdits: []analysis.TextEdit{{
				Pos:     sig.Start,
				End:     sig.End,
				NewText: []byte(newText),
			}},
		}},
	})
}
