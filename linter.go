// Package linters provides a golangci-lint plugin for formatting Go function signatures.
package linters

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

const (
	defaultMaxLineLen         = 120
	defaultTabWidth           = 8
	analyzerName              = "sigfmt"
	analyzerDoc               = "Checks if multi-line function signatures can be collapsed to one line or reformatted more compactly"
	diagnosticMessage         = "Multi-line signature can be collapsed to one line"
	diagnosticMessageReformat = "Signature can be reformatted more compactly"
	fixMessage                = "Collapse to one line"
	fixMessageReformat        = "Reformat with grouped parameters"

	// Action types for signature formatting
	actionCollapse = "collapse"
	actionReformat = "reformat"
)

func init() {
	register.Plugin(analyzerName, New)
}

// Settings contains linter configuration
type Settings struct {
	MaxLineLen           int
	TabWidth             int
	PackStructFields     bool
	PackInterfaceMethods bool
}

// PluginLineWrap implements register.LinterPlugin interface.
type PluginLineWrap struct {
	settings Settings
}

// signatureInfo contains signature information for checking
type signatureInfo struct {
	start             token.Pos      // start of replacement
	end               token.Pos      // end of replacement
	diagPos           token.Pos      // position for diagnostic
	oneLineText       string         // text of one-line version
	reformattedText   string         // text of improved multi-line version (if doesn't fit on one line)
	funcType          *ast.FuncType  // function type for generating formatted version
	receiver          *ast.FieldList // receiver for methods (can be nil)
	name              string         // function/method name
	isStructField     bool           // true if this is a struct field with func type
	isInterfaceMethod bool           // true if this is an interface method
}

// New returns a new instance of the sigfmt linter plugin.
func New(settings any) (register.LinterPlugin, error) {
	p := &PluginLineWrap{}
	// Set defaults
	p.settings.MaxLineLen = defaultMaxLineLen
	p.settings.TabWidth = defaultTabWidth
	p.settings.PackStructFields = true
	p.settings.PackInterfaceMethods = true

	if s, ok := settings.(map[string]interface{}); ok {
		if v, ok := s["max-line-len"].(float64); ok && v > 0 {
			p.settings.MaxLineLen = int(v)
		}
		if v, ok := s["tab-width"].(float64); ok && v > 0 {
			p.settings.TabWidth = int(v)
		}
		if v, ok := s["pack-struct-fields"].(bool); ok {
			p.settings.PackStructFields = v
		}
		if v, ok := s["pack-interface-methods"].(bool); ok {
			p.settings.PackInterfaceMethods = v
		}
	}
	return p, nil
}

// BuildAnalyzers returns the analysis.Analyzer for the sigfmt plugin.
func (p *PluginLineWrap) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{{
		Name: analyzerName,
		Doc:  analyzerDoc,
		Run:  p.run,
	}}, nil
}

// GetLoadMode returns the LoadMode for the analyzer.
func (p *PluginLineWrap) GetLoadMode() string {
	return register.LoadModeSyntax
}

func (p *PluginLineWrap) run(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				p.checkFuncDecl(pass, x)
			case *ast.FuncLit:
				p.checkFuncLit(pass, x)
			case *ast.TypeSpec:
				if iface, ok := x.Type.(*ast.InterfaceType); ok {
					p.checkInterface(pass, iface)
				}
				if structType, ok := x.Type.(*ast.StructType); ok {
					p.checkStruct(pass, structType)
				}
			}
			return true
		})
	}
	return nil, nil
}

func (p *PluginLineWrap) checkFuncDecl(pass *analysis.Pass, decl *ast.FuncDecl) {
	if decl.Type == nil || decl.Type.Params == nil {
		return
	}

	sig := p.extractFuncDeclSignature(pass.Fset, decl)
	if sig == nil {
		return
	}

	action := p.checkSignature(pass, sig)
	if action != "" {
		p.report(pass, sig, action)
	}
}

func (p *PluginLineWrap) checkFuncLit(pass *analysis.Pass, lit *ast.FuncLit) {
	if lit.Type == nil || lit.Type.Params == nil {
		return
	}

	sig := p.extractFuncLitSignature(pass.Fset, lit)
	if sig == nil {
		return
	}

	action := p.checkSignature(pass, sig)
	if action != "" {
		p.report(pass, sig, action)
	}
}

func (p *PluginLineWrap) checkInterface(pass *analysis.Pass, iface *ast.InterfaceType) {
	if iface.Methods == nil {
		return
	}

	for _, m := range iface.Methods.List {
		if len(m.Names) == 0 {
			continue
		}

		ft, ok := m.Type.(*ast.FuncType)
		if !ok || ft.Params == nil {
			continue
		}

		sig := p.extractMethodSignature(pass.Fset, m.Names[0], ft)
		if sig == nil {
			continue
		}

		action := p.checkSignature(pass, sig)
		if action != "" {
			p.report(pass, sig, action)
		}
	}
}

func (p *PluginLineWrap) checkStruct(pass *analysis.Pass, structType *ast.StructType) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		ft, ok := field.Type.(*ast.FuncType)
		if !ok || ft.Params == nil {
			continue
		}

		// Struct fields can be unnamed (anonymous), skip them
		if len(field.Names) == 0 {
			continue
		}

		sig := p.extractStructFieldSignature(pass.Fset, field.Names[0], ft)
		if sig == nil {
			continue
		}

		action := p.checkSignature(pass, sig)
		if action != "" {
			p.report(pass, sig, action)
		}
	}
}

// checkSignature checks if signature formatting needs to be changed.
// Returns actionCollapse if needs to be collapsed to one line,
// actionReformat if multi-line formatting needs improvement,
// or "" if no changes are required
func (p *PluginLineWrap) checkSignature(pass *analysis.Pass, sig *signatureInfo) string {
	fset := pass.Fset
	startLine := fset.Position(sig.start).Line
	endLine := fset.Position(sig.end).Line

	// If one-line version of signature fits in MaxLineLen
	if p.visualLength(sig.oneLineText) <= p.settings.MaxLineLen {
		// If signature is currently multi-line but fits on one, suggest collapsing
		if startLine != endLine {
			return actionCollapse
		}
		// Otherwise (already on one line and fits), do nothing
		return ""
	}

	// If doesn't fit on one line, check if formatting can be improved
	if p.shouldReformat(pass.Fset, sig) {
		sig.reformattedText = p.buildReformattedSignature(pass.Fset, sig)
		return actionReformat
	}

	return ""
}

// visualLength calculates visual length of string considering tabs
func (p *PluginLineWrap) visualLength(s string) int {
	length := 0
	for _, c := range s {
		if c == '\t' {
			length += p.settings.TabWidth
		} else {
			length++
		}
	}
	return length
}

// shouldReformat checks if multi-line signature needs reformatting.
// Returns true if signature can be packed more compactly.
func (p *PluginLineWrap) shouldReformat(fset *token.FileSet, sig *signatureInfo) bool {
	if sig.funcType == nil || sig.funcType.Params == nil {
		return false
	}

	// Don't reformat single-line signatures
	if fset.Position(sig.start).Line == fset.Position(sig.end).Line {
		return false
	}

	params := sig.funcType.Params.List
	if len(params) <= 1 {
		return false
	}

	// For interface methods and struct fields, apply aggressive formatting
	// (parameter packing) if enabled in settings
	if sig.isInterfaceMethod && p.settings.PackInterfaceMethods {
		return true
	}
	if sig.isStructField && p.settings.PackStructFields {
		return true
	}

	// For regular functions: reformat only if parameters are NOT on separate lines
	// (i.e. if they're already neatly split line by line, leave as is)
	return !p.areParamsOnSeparateLines(fset, params)
}

// areParamsOnSeparateLines checks if each parameter is on a separate line
func (p *PluginLineWrap) areParamsOnSeparateLines(fset *token.FileSet, params []*ast.Field) bool {
	if len(params) <= 1 {
		return true
	}

	prevLine := fset.Position(params[0].Pos()).Line
	for i := 1; i < len(params); i++ {
		currentLine := fset.Position(params[i].Pos()).Line
		if currentLine == prevLine {
			return false
		}
		prevLine = currentLine
	}
	return true
}

// buildReformattedSignature generates improved multi-line formatting
func (p *PluginLineWrap) buildReformattedSignature(fset *token.FileSet, sig *signatureInfo) string {
	if sig.funcType == nil {
		return ""
	}

	var sb strings.Builder

	// Prefix (func, name, receiver, etc.)
	switch {
	case sig.isStructField:
		sb.WriteString(sig.name)
		sb.WriteString(" func")
	case sig.isInterfaceMethod:
		sb.WriteString(sig.name)
	default:
		sb.WriteString("func ")
		if sig.receiver != nil {
			sb.WriteString(p.renderFieldList(fset, sig.receiver, "(", ")"))
			sb.WriteString(" ")
		}
		if sig.name != "" {
			sb.WriteString(sig.name)
		}
	}

	// Type parameters (generics)
	if sig.funcType.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, sig.funcType.TypeParams, "[", "]"))
	}

	// Group parameters
	sb.WriteString(p.renderFieldListGrouped(fset, sig.funcType.Params, sig, "(", ")"))

	// Results
	if sig.funcType.Results != nil {
		sb.WriteString(p.renderResults(fset, sig.funcType.Results))
	}

	return sb.String()
}

// renderFieldListGrouped renders parameters with grouping by lines
func (p *PluginLineWrap) renderFieldListGrouped(fset *token.FileSet, fl *ast.FieldList, sig *signatureInfo, openBracket, closeBracket string) string {
	if fl == nil || len(fl.List) == 0 {
		return openBracket + closeBracket
	}

	// Calculate prefix (how much space the beginning of the line takes)
	prefixLen := len(sig.oneLineText) - len(p.renderFieldList(fset, fl, openBracket, closeBracket))
	if sig.funcType.Results != nil {
		prefixLen -= len(p.renderResults(fset, sig.funcType.Results))
	}

	// Get indent for continuation lines
	indent := "\t" // Standard Go indentation

	var lines []string
	var currentLine strings.Builder
	currentLine.WriteString(openBracket)
	currentLineLen := prefixLen + 1 // +1 for opening bracket

	for i, field := range fl.List {
		// Render current field
		var fieldStr string
		for j, name := range field.Names {
			if j > 0 {
				fieldStr += ", "
			}
			fieldStr += name.Name
		}
		if len(field.Names) > 0 {
			fieldStr += " "
		}
		fieldStr += p.renderNode(fset, field.Type)

		// Check if field fits on current line
		testLen := currentLineLen
		if currentLine.Len() > len(openBracket) {
			testLen += 2 // for ", "
		}
		testLen += len(fieldStr)

		// If not first parameter and doesn't fit, start new line
		if i > 0 && testLen > p.settings.MaxLineLen && currentLine.Len() > len(openBracket) {
			currentLine.WriteString(",")
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(indent)
			currentLine.WriteString(fieldStr)
			currentLineLen = p.settings.TabWidth + len(fieldStr)
		} else {
			// Add to current line
			if currentLine.Len() > len(openBracket) {
				currentLine.WriteString(", ")
			}
			currentLine.WriteString(fieldStr)
			currentLineLen = testLen
		}
	}

	// Add closing bracket on same line as last parameter
	currentLine.WriteString(closeBracket)
	lines = append(lines, currentLine.String())

	return strings.Join(lines, "\n")
}

// computeDiagPos calculates position for diagnostic (closing bracket of params or results)
func computeDiagPos(params, results *ast.FieldList) token.Pos {
	diagPos := params.Closing
	if results != nil && results.Closing.IsValid() {
		diagPos = results.Closing
	}
	return diagPos
}

// extractFuncDeclSignature extracts function signature information
func (p *PluginLineWrap) extractFuncDeclSignature(fset *token.FileSet, decl *ast.FuncDecl) *signatureInfo {
	start := decl.Type.Func
	end := decl.Type.Params.End()
	if decl.Type.Results != nil {
		end = decl.Type.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:         start,
		end:           end,
		diagPos:       computeDiagPos(decl.Type.Params, decl.Type.Results),
		oneLineText:   p.buildFuncDeclSignature(fset, decl),
		funcType:      decl.Type,
		receiver:      decl.Recv,
		name:          decl.Name.Name,
		isStructField: false,
	}
}

// extractFuncLitSignature extracts function literal signature information
func (p *PluginLineWrap) extractFuncLitSignature(fset *token.FileSet, lit *ast.FuncLit) *signatureInfo {
	start := lit.Type.Func
	end := lit.Type.Params.End()
	if lit.Type.Results != nil {
		end = lit.Type.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:         start,
		end:           end,
		diagPos:       computeDiagPos(lit.Type.Params, lit.Type.Results),
		oneLineText:   p.buildFuncLitSignature(fset, lit.Type),
		funcType:      lit.Type,
		receiver:      nil,
		name:          "",
		isStructField: false,
	}
}

// extractMethodSignature extracts interface method signature information
func (p *PluginLineWrap) extractMethodSignature(fset *token.FileSet, name *ast.Ident, ft *ast.FuncType) *signatureInfo {
	start := name.Pos()
	end := ft.Params.End()
	if ft.Results != nil {
		end = ft.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:             start,
		end:               end,
		diagPos:           computeDiagPos(ft.Params, ft.Results),
		oneLineText:       p.buildMethodSignature(fset, name.Name, ft),
		funcType:          ft,
		receiver:          nil,
		name:              name.Name,
		isStructField:     false,
		isInterfaceMethod: true,
	}
}

// extractStructFieldSignature extracts struct field signature information (func type)
func (p *PluginLineWrap) extractStructFieldSignature(fset *token.FileSet, name *ast.Ident, ft *ast.FuncType) *signatureInfo {
	start := name.Pos()
	end := ft.Params.End()
	if ft.Results != nil {
		end = ft.Results.End()
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:         start,
		end:           end,
		diagPos:       computeDiagPos(ft.Params, ft.Results),
		oneLineText:   p.buildStructFieldSignature(fset, name.Name, ft),
		funcType:      ft,
		receiver:      nil,
		name:          name.Name,
		isStructField: true,
	}
}

// buildSignature - universal function for building one-line signatures
func (p *PluginLineWrap) buildSignature(fset *token.FileSet, prefix string, ft *ast.FuncType, results *ast.FieldList) string {
	var sb strings.Builder
	sb.WriteString(prefix)

	// Type parameters (generics)
	if ft.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, ft.TypeParams, "[", "]"))
	}

	// Parameters
	sb.WriteString(p.renderFieldList(fset, ft.Params, "(", ")"))

	// Results
	if results != nil {
		sb.WriteString(p.renderResults(fset, results))
	}

	return sb.String()
}

// buildFuncDeclSignature builds one-line version of function signature
func (p *PluginLineWrap) buildFuncDeclSignature(fset *token.FileSet, decl *ast.FuncDecl) string {
	var prefix strings.Builder
	prefix.WriteString("func ")

	if decl.Recv != nil {
		prefix.WriteString(p.renderFieldList(fset, decl.Recv, "(", ")"))
		prefix.WriteString(" ")
	}

	prefix.WriteString(decl.Name.Name)
	return p.buildSignature(fset, prefix.String(), decl.Type, decl.Type.Results)
}

// buildFuncLitSignature builds one-line version of function literal signature
func (p *PluginLineWrap) buildFuncLitSignature(fset *token.FileSet, ft *ast.FuncType) string {
	return p.buildSignature(fset, "func", ft, ft.Results)
}

// buildMethodSignature builds one-line version of method signature
func (p *PluginLineWrap) buildMethodSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return p.buildSignature(fset, name, ft, ft.Results)
}

// buildStructFieldSignature builds one-line version of struct field with func type
func (p *PluginLineWrap) buildStructFieldSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return p.buildSignature(fset, name+" func", ft, ft.Results)
}

// renderResults renders return values
func (p *PluginLineWrap) renderResults(fset *token.FileSet, results *ast.FieldList) string {
	if results == nil || len(results.List) == 0 {
		return ""
	}

	// If single result without name, don't wrap in parentheses
	if len(results.List) == 1 && len(results.List[0].Names) == 0 {
		return " " + p.renderNode(fset, results.List[0].Type)
	}

	// Otherwise wrap in parentheses
	return " " + p.renderFieldList(fset, results, "(", ")")
}

// renderNode renders AST node to string
func (p *PluginLineWrap) renderNode(fset *token.FileSet, n ast.Node) string {
	if n == nil {
		return ""
	}

	// Special handling for FieldList
	if fl, ok := n.(*ast.FieldList); ok {
		return p.renderFieldList(fset, fl, "(", ")")
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return ""
	}

	// Collapse all spaces and newlines into single space
	return strings.Join(strings.Fields(buf.String()), " ")
}

// renderFieldList renders field list with given brackets
func (p *PluginLineWrap) renderFieldList(fset *token.FileSet, fl *ast.FieldList, openBracket, closeBracket string) string {
	if fl == nil || len(fl.List) == 0 {
		return openBracket + closeBracket
	}

	var sb strings.Builder
	sb.WriteString(openBracket)

	for i, field := range fl.List {
		if i > 0 {
			sb.WriteString(", ")
		}

		// Render field names
		for j, name := range field.Names {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(name.Name)
		}

		// If there are names, add space before type
		if len(field.Names) > 0 {
			sb.WriteString(" ")
		}

		// Render type
		sb.WriteString(p.renderNode(fset, field.Type))
	}

	sb.WriteString(closeBracket)
	return sb.String()
}

// report reports found issue with suggested fix
func (p *PluginLineWrap) report(pass *analysis.Pass, sig *signatureInfo, action string) {
	var message, fixMsg string
	var newText []byte

	switch action {
	case actionCollapse:
		message = diagnosticMessage
		fixMsg = fixMessage
		newText = []byte(sig.oneLineText)
	case actionReformat:
		message = diagnosticMessageReformat
		fixMsg = fixMessageReformat
		newText = []byte(sig.reformattedText)
	default:
		return
	}

	pass.Report(analysis.Diagnostic{
		Pos:     sig.diagPos,
		End:     sig.diagPos,
		Message: message,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: fixMsg,
			TextEdits: []analysis.TextEdit{{
				Pos:     sig.start,
				End:     sig.end,
				NewText: newText,
			}},
		}},
	})
}
