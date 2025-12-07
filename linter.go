// Package sigfmt provides a golangci-lint plugin for formatting Go function signatures.
// It enforces consistent line breaking and parameter grouping rules to improve code readability
// and reduce diff noise in version control systems.
package sigfmt

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
	// defaultMaxLineLen is the default maximum line length allowed before wrapping is suggested.
	defaultMaxLineLen = 120
	// defaultTabWidth is the default number of spaces a tab character represents for length calculation.
	defaultTabWidth = 8
	analyzerName    = "sigfmt"
	analyzerDoc     = "Checks if multi-line function signatures can be collapsed to one line or reformatted more compactly"

	diagnosticMessage         = "Multi-line signature can be collapsed to one line"
	diagnosticMessageReformat = "Signature can be reformatted more compactly"
	fixMessage                = "Collapse to one line"
	fixMessageReformat        = "Reformat with grouped parameters"

	// Action types for signature formatting decision logic.
	actionCollapse = "collapse"
	actionReformat = "reformat"
)

func init() {
	register.Plugin(analyzerName, New)
}

// Settings contains configuration parameters for the linter.
// These settings are populated from the .golangci.yml configuration file.
type Settings struct {
	// MaxLineLen is the maximum allowed length of a line (including indentation).
	MaxLineLen int
	// TabWidth is the visual width of a tab character used for length calculations.
	TabWidth int
	// PackStructFields enables aggressive packing of function types within structs.
	// If true, multiple parameters will be placed on the same line if they fit.
	PackStructFields bool
	// PackInterfaceMethods enables aggressive packing of method signatures within interfaces.
	// If true, multiple parameters will be placed on the same line if they fit.
	PackInterfaceMethods bool
}

// PluginLineWrap implements the register.LinterPlugin interface.
// It holds the configuration state for the analyzer instance.
type PluginLineWrap struct {
	settings Settings
}

// signatureInfo contains detailed metadata about a function signature being analyzed.
// It stores both the original AST information and the calculated reformatted text.
type signatureInfo struct {
	start             token.Pos      // The starting position of the signature (e.g., 'func' keyword or function name).
	end               token.Pos      // The ending position of the signature (usually the closing brace or return type).
	diagPos           token.Pos      // The position where the diagnostic message should be reported.
	oneLineText       string         // The generated text representation of the signature if collapsed to a single line.
	reformattedText   string         // The generated text representation of the signature if reformatted/packed (multi-line).
	funcType          *ast.FuncType  // The underlying AST node for the function type.
	receiver          *ast.FieldList // The receiver field list (nil for functions, non-nil for methods).
	name              string         // The name of the function or method.
	isStructField     bool           // Indicates if this signature belongs to a struct field.
	isInterfaceMethod bool           // Indicates if this signature belongs to an interface method.
}

// New returns a new instance of the sigfmt linter plugin with the provided settings.
// It parses the raw settings map and applies default values where necessary.
func New(settings any) (register.LinterPlugin, error) {
	p := &PluginLineWrap{}
	// Set defaults
	p.settings.MaxLineLen = defaultMaxLineLen
	p.settings.TabWidth = defaultTabWidth
	p.settings.PackStructFields = true
	p.settings.PackInterfaceMethods = true

	// Parse settings from the generic map
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

// BuildAnalyzers returns the analysis.Analyzer definition for the sigfmt plugin.
// This is the entry point used by golangci-lint to register the analyzer.
func (p *PluginLineWrap) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{{
		Name: analyzerName,
		Doc:  analyzerDoc,
		Run:  p.run,
	}}, nil
}

// GetLoadMode returns the LoadMode required for the analyzer.
// This linter only requires syntax information (AST), not full type checking,
// making it significantly faster.
func (p *PluginLineWrap) GetLoadMode() string {
	return register.LoadModeSyntax
}

// run executes the analysis pass on the package files.
// It iterates through all files and inspects relevant AST nodes (FuncDecl, FuncLit, TypeSpec).
func (p *PluginLineWrap) run(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				p.checkFuncDecl(pass, x)
			case *ast.FuncLit:
				p.checkFuncLit(pass, x)
			case *ast.TypeSpec:
				// Handle interfaces and structs
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

// checkFuncDecl inspects a top-level function declaration.
// It validates regular functions and methods attached to types.
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

// checkFuncLit inspects an anonymous function literal (closure).
// Example: var f = func(a int) { ... }
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

// checkInterface inspects method signatures defined within an interface.
func (p *PluginLineWrap) checkInterface(pass *analysis.Pass, iface *ast.InterfaceType) {
	if iface.Methods == nil {
		return
	}

	for _, m := range iface.Methods.List {
		// Skip embedded interfaces or erroneous nodes without names
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

// checkStruct inspects function type fields defined within a struct.
// Example: type Handler struct { Handle func(ctx context.Context) error }
func (p *PluginLineWrap) checkStruct(pass *analysis.Pass, structType *ast.StructType) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		ft, ok := field.Type.(*ast.FuncType)
		if !ok || ft.Params == nil {
			continue
		}

		// Struct fields can be unnamed (anonymous embedding), skip them as they act as mixins
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

// checkSignature is the core logic that determines if a signature needs formatting changes.
//
// Returns:
//   - actionCollapse: if the signature should be collapsed to one line.
//   - actionReformat: if the signature should remain multi-line but needs better packing.
//   - "": if no changes are required.
func (p *PluginLineWrap) checkSignature(pass *analysis.Pass, sig *signatureInfo) string {
	fset := pass.Fset
	startLine := fset.Position(sig.start).Line
	endLine := fset.Position(sig.end).Line

	// 1. Check if the signature can fit on a single line
	if p.visualLength(sig.oneLineText) <= p.settings.MaxLineLen {
		// If it fits, and it is currently split across multiple lines, suggest collapsing.
		if startLine != endLine {
			return actionCollapse
		}
		// If it fits and is already on one line, it is correct.
		return ""
	}

	// 2. If it doesn't fit on one line, check if the multi-line formatting can be improved.
	if p.shouldReformat(pass.Fset, sig) {
		sig.reformattedText = p.buildReformattedSignature(pass.Fset, sig)
		return actionReformat
	}

	return ""
}

// visualLength calculates the visual length of a string, accounting for tab expansion.
// This ensures that the linter respects the user's editor settings for tab width.
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

// shouldReformat determines if a multi-line signature is eligible for reformatting (packing).
//
// Reformatting is suggested if:
//   - It's an interface method or struct field (and packing is enabled).
//   - It's a regular function where parameters are not cleanly split one-per-line.
func (p *PluginLineWrap) shouldReformat(fset *token.FileSet, sig *signatureInfo) bool {
	if sig.funcType == nil || sig.funcType.Params == nil {
		return false
	}

	// Don't reformat single-line signatures (handled by collapse logic)
	if fset.Position(sig.start).Line == fset.Position(sig.end).Line {
		return false
	}

	params := sig.funcType.Params.List
	if len(params) <= 1 {
		return false
	}

	// Strategy: Aggressive packing for definition types (Interfaces, Structs)
	// These often contain many small signatures where vertical space is premium.
	if sig.isInterfaceMethod && p.settings.PackInterfaceMethods {
		return true
	}
	if sig.isStructField && p.settings.PackStructFields {
		return true
	}

	// Strategy: Conservative packing for Implementations (Funcs, Methods)
	// We only interfere if the user has a "mixed" style (e.g. some args on same line, some on new).
	// If the user has explicitly placed every arg on a new line, we respect that choice.
	return !p.areParamsOnSeparateLines(fset, params)
}

// areParamsOnSeparateLines checks if every parameter in the list starts on a new line.
// This is used to detect the "staircase" or "one-arg-per-line" style.
func (p *PluginLineWrap) areParamsOnSeparateLines(fset *token.FileSet, params []*ast.Field) bool {
	if len(params) <= 1 {
		return true
	}

	prevLine := fset.Position(params[0].Pos()).Line
	for i := 1; i < len(params); i++ {
		currentLine := fset.Position(params[i].Pos()).Line
		if currentLine == prevLine {
			// Found two parameters on the same line
			return false
		}
		prevLine = currentLine
	}
	return true
}

// buildReformattedSignature generates the source code for a multi-line signature
// with optimized parameter packing (grouping multiple parameters on a line if they fit).
func (p *PluginLineWrap) buildReformattedSignature(fset *token.FileSet, sig *signatureInfo) string {
	if sig.funcType == nil {
		return ""
	}

	var sb strings.Builder

	// 1. Build the prefix (everything before the parameters)
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

	// 2. Handle Type Parameters (Generics) - e.g. [T any]
	if sig.funcType.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, sig.funcType.TypeParams, "[", "]"))
	}

	// 3. Build the grouped parameter list
	sb.WriteString(p.renderFieldListGrouped(fset, sig.funcType.Params, sig, "(", ")"))

	// 4. Append Return Results
	if sig.funcType.Results != nil {
		sb.WriteString(p.renderResults(fset, sig.funcType.Results))
	}

	return sb.String()
}

// renderFieldListGrouped renders a list of fields (parameters), automatically grouping them
// onto lines based on the MaxLineLen setting.
func (p *PluginLineWrap) renderFieldListGrouped(fset *token.FileSet, fl *ast.FieldList, sig *signatureInfo, openBracket, closeBracket string) string {
	if fl == nil || len(fl.List) == 0 {
		return openBracket + closeBracket
	}

	// Calculate the available space on the first line after the function declaration prefix.
	prefixLen := len(sig.oneLineText) - len(p.renderFieldList(fset, fl, openBracket, closeBracket))
	if sig.funcType.Results != nil {
		prefixLen -= len(p.renderResults(fset, sig.funcType.Results))
	}

	// Standard Go indentation for continuation lines
	indent := "\t"

	var lines []string
	var currentLine strings.Builder
	currentLine.WriteString(openBracket)

	// Track the current visual length of the line being built
	currentLineLen := prefixLen + 1 // +1 for opening bracket

	for i, field := range fl.List {
		// Render the individual field (e.g., "ctx context.Context" or "a, b int")
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

		// Calculate length if we add this field to the current line
		testLen := currentLineLen
		if currentLine.Len() > len(openBracket) {
			testLen += 2 // for ", " separator
		}
		testLen += len(fieldStr)

		// Logic: If it's not the first param and adding it exceeds the limit, wrap to new line.
		if i > 0 && testLen > p.settings.MaxLineLen && currentLine.Len() > len(openBracket) {
			currentLine.WriteString(",") // Add comma to previous line
			lines = append(lines, currentLine.String())

			// Start new line
			currentLine.Reset()
			currentLine.WriteString(indent)
			currentLine.WriteString(fieldStr)
			currentLineLen = p.settings.TabWidth + len(fieldStr)
		} else {
			// Append to current line
			if currentLine.Len() > len(openBracket) {
				currentLine.WriteString(", ")
			}
			currentLine.WriteString(fieldStr)
			currentLineLen = testLen
		}
	}

	// Add closing bracket to the last line
	currentLine.WriteString(closeBracket)
	lines = append(lines, currentLine.String())

	return strings.Join(lines, "\n")
}

// computeDiagPos calculates the best position to place the diagnostic warning.
// It prefers the closing bracket of the parameters or results to avoid visual clutter.
func computeDiagPos(params, results *ast.FieldList) token.Pos {
	diagPos := params.Closing
	if results != nil && results.Closing.IsValid() {
		diagPos = results.Closing
	}
	return diagPos
}

// extractFuncDeclSignature extracts relevant information from a *ast.FuncDecl.
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

// extractFuncLitSignature extracts relevant information from a *ast.FuncLit (anonymous function).
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

// extractMethodSignature extracts relevant information from an interface method definition.
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

// extractStructFieldSignature extracts relevant information from a struct field with a function type.
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

// buildSignature is a helper to construct the one-line string representation of any function type.
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

// buildFuncDeclSignature reconstructs the signature of a function declaration.
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

// buildFuncLitSignature reconstructs the signature of an anonymous function.
func (p *PluginLineWrap) buildFuncLitSignature(fset *token.FileSet, ft *ast.FuncType) string {
	return p.buildSignature(fset, "func", ft, ft.Results)
}

// buildMethodSignature reconstructs the signature of an interface method.
func (p *PluginLineWrap) buildMethodSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return p.buildSignature(fset, name, ft, ft.Results)
}

// buildStructFieldSignature reconstructs the signature of a struct field function.
func (p *PluginLineWrap) buildStructFieldSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return p.buildSignature(fset, name+" func", ft, ft.Results)
}

// renderResults converts the results list (return values) to a string.
// It handles cases with and without parentheses (e.g. `error` vs `(int, error)`).
func (p *PluginLineWrap) renderResults(fset *token.FileSet, results *ast.FieldList) string {
	if results == nil || len(results.List) == 0 {
		return ""
	}

	// Special case: Single unnamed result doesn't need parentheses
	// e.g. "func Foo() int" vs "func Foo() (int, error)"
	if len(results.List) == 1 && len(results.List[0].Names) == 0 {
		return " " + p.renderNode(fset, results.List[0].Type)
	}

	return " " + p.renderFieldList(fset, results, "(", ")")
}

// renderNode converts an AST node to its string representation.
// It uses a custom approach for whitespace collapsing to ensure consistent output.
func (p *PluginLineWrap) renderNode(fset *token.FileSet, n ast.Node) string {
	if n == nil {
		return ""
	}

	// Optimization: Direct dispatch for FieldList to avoid printer overhead
	if fl, ok := n.(*ast.FieldList); ok {
		return p.renderFieldList(fset, fl, "(", ")")
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return ""
	}

	// Normalize whitespace: collapse all sequences of whitespace/newlines into a single space
	return strings.Join(strings.Fields(buf.String()), " ")
}

// renderFieldList converts a list of fields (parameters, results, generics) to a comma-separated string.
// It adds the specified opening and closing brackets.
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

		// Render parameter names (e.g. "a, b" in "a, b int")
		for j, name := range field.Names {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(name.Name)
		}

		// Add space between names and type if names exist
		if len(field.Names) > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(p.renderNode(fset, field.Type))
	}

	sb.WriteString(closeBracket)
	return sb.String()
}

// report sends the diagnostic and suggested fix to the analysis pass.
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
