package format

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
	"github.com/vsfedorenko/sigfmt/internal/pkg/field"
	"github.com/vsfedorenko/sigfmt/internal/pkg/source"
	"github.com/vsfedorenko/sigfmt/internal/pkg/text"
	"github.com/vsfedorenko/sigfmt/internal/render"
)

// Builder generates reformatted signatures.
type Builder struct {
	config   config.Settings
	renderer *render.Renderer
}

func NewBuilder(cfg config.Settings, renderer *render.Renderer) *Builder {
	return &Builder{
		config:   cfg,
		renderer: renderer,
	}
}

// BuildReformattedSignature generates a multi-line signature with optimized parameter packing.
func (b *Builder) BuildReformattedSignature(fset *token.FileSet, sig *domain.Signature) string {
	if sig.FuncType == nil {
		return ""
	}

	var sb strings.Builder

	switch {
	case sig.IsStructField:
		sb.WriteString(sig.Name)
		sb.WriteString(" func")
	case sig.IsInterfaceMethod:
		sb.WriteString(sig.Name)
	default:
		sb.WriteString("func ")
		if sig.Receiver != nil {
			sb.WriteString(b.renderer.FieldList(fset, sig.Receiver, "(", ")"))
			sb.WriteString(" ")
		}
		if sig.Name != "" {
			sb.WriteString(sig.Name)
		}
	}

	if sig.FuncType.TypeParams != nil {
		sb.WriteString(b.renderer.FieldList(fset, sig.FuncType.TypeParams, "[", "]"))
	}

	baseIndent := source.GetIndent(fset, sig.Start)
	indentUnit := "\t"
	if strings.Contains(baseIndent, " ") && !strings.Contains(baseIndent, "\t") {
		indentUnit = "    "
	}
	paramIndent := baseIndent + indentUnit
	baseIndentLen := text.VisualLength(baseIndent, b.config.TabWidth)

	sb.WriteString(b.renderFieldListGrouped(fset, sig.FuncType.Params, sig, "(", ")", paramIndent, baseIndentLen))

	if sig.FuncType.Results != nil {
		sb.WriteString(b.renderer.Results(fset, sig.FuncType.Results))
	}

	return sb.String()
}

func (b *Builder) renderFieldListGrouped(fset *token.FileSet, fl *ast.FieldList, sig *domain.Signature, openBracket, closeBracket, indent string, baseIndentLen int) string {
	if fl == nil || len(fl.List) == 0 {
		return openBracket + closeBracket
	}

	prefixLen := baseIndentLen + text.VisualLength(sig.OneLineText, b.config.TabWidth) - text.VisualLength(b.renderer.FieldList(fset, fl, openBracket, closeBracket), b.config.TabWidth)
	if sig.FuncType.Results != nil {
		prefixLen -= text.VisualLength(b.renderer.Results(fset, sig.FuncType.Results), b.config.TabWidth)
	}
	prefixLen += text.VisualLength(openBracket, b.config.TabWidth)

	writer := b.newBuilderLineWriter(indent, openBracket, prefixLen)

	i := 0
	for i < len(fl.List) {
		fieldsToProcess, isGroupMatch := b.getFieldsToProcess(fset, fl.List, i)

		for k := 0; k < fieldsToProcess; k++ {
			writer.add(b.renderField(fset, fl.List[i+k]))
		}

		if isGroupMatch && (i+fieldsToProcess < len(fl.List)) {
			remainingLen := b.calculateRemainingParamsLength(fset, fl.List[i+fieldsToProcess:])
			if !writer.canFitOnCurrentLine(remainingLen) {
				writer.forceBreak()
			}
		}

		i += fieldsToProcess
	}

	return writer.String(closeBracket)
}

// getFieldsToProcess determines how many fields to process together.
// Returns the number of fields to process and whether they match a param group.
func (b *Builder) getFieldsToProcess(fset *token.FileSet, list []*ast.Field, startIdx int) (int, bool) {
	for _, group := range b.config.ParamGroups {
		if b.matchesGroup(fset, list, startIdx, group) {
			return len(group), true
		}
	}
	return 1, false
}

// renderField renders a single field as a string (names + type).
func (b *Builder) renderField(fset *token.FileSet, f *ast.Field) string {
	var sb strings.Builder

	names := field.RenderNames(f.Names)
	if names != "" {
		sb.WriteString(names)
		sb.WriteString(" ")
	}

	sb.WriteString(b.renderer.Node(fset, f.Type))
	return sb.String()
}

func (b *Builder) matchesGroup(fset *token.FileSet, list []*ast.Field, startIdx int, group []string) bool {
	if startIdx+len(group) > len(list) {
		return false
	}

	for i, typeName := range group {
		field := list[startIdx+i]
		actualType := b.renderer.Node(fset, field.Type)
		if actualType != typeName {
			return false
		}
	}
	return true
}

// calculateRemainingParamsLength calculates the visual length of remaining parameters
func (b *Builder) calculateRemainingParamsLength(fset *token.FileSet, fields []*ast.Field) int {
	if len(fields) == 0 {
		return 0
	}

	totalLen := 0
	for i, field := range fields {
		if i > 0 {
			totalLen += 2 // ", "
		}
		totalLen += text.VisualLength(b.renderField(fset, field), b.config.TabWidth)
	}
	return totalLen
}

// builderLineWriter manages line wrapping and formatting for signatures.
type builderLineWriter struct {
	b             *Builder
	lines         []string
	currentLine   strings.Builder
	currentVisLen int
	indent        string
	indentVisLen  int
	openBracket   string
}

func (b *Builder) newBuilderLineWriter(indent, openBracket string, initialVisLen int) *builderLineWriter {
	l := &builderLineWriter{
		b:             b,
		indent:        indent,
		indentVisLen:  text.VisualLength(indent, b.config.TabWidth),
		openBracket:   openBracket,
		currentVisLen: initialVisLen,
	}
	l.currentLine.WriteString(openBracket)
	return l
}

func (lb *builderLineWriter) add(s string) {
	textVisLen := text.VisualLength(s, lb.b.config.TabWidth)
	currentText := lb.currentLine.String()
	needsComma := currentText != lb.openBracket && currentText != lb.indent

	potentialLen := lb.currentVisLen + textVisLen
	if needsComma {
		potentialLen += 2
	}

	shouldWrap := potentialLen > lb.b.config.MaxLineLen && currentText != lb.indent

	if shouldWrap {
		if currentText != lb.openBracket {
			lb.currentLine.WriteString(",")
		}
		lb.lines = append(lb.lines, lb.currentLine.String())
		lb.currentLine.Reset()
		lb.currentLine.WriteString(lb.indent)
		lb.currentVisLen = lb.indentVisLen
		needsComma = false
	}

	if needsComma {
		lb.currentLine.WriteString(", ")
		lb.currentVisLen += 2
	}

	lb.currentLine.WriteString(s)
	lb.currentVisLen += textVisLen
}

func (lb *builderLineWriter) forceBreak() {
	lb.currentLine.WriteString(",")
	lb.lines = append(lb.lines, lb.currentLine.String())
	lb.currentLine.Reset()
	lb.currentLine.WriteString(lb.indent)
	lb.currentVisLen = lb.indentVisLen
}

func (lb *builderLineWriter) canFitOnCurrentLine(additionalLength int) bool {
	// Account for comma and space before the next parameter
	potentialLen := lb.currentVisLen + 2 + additionalLength
	return potentialLen <= lb.b.config.MaxLineLen
}

func (lb *builderLineWriter) String(closeBracket string) string {
	lb.currentLine.WriteString(closeBracket)
	lb.lines = append(lb.lines, lb.currentLine.String())
	return strings.Join(lb.lines, "\n")
}
