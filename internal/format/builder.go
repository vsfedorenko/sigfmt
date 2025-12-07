package format

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/vsfedorenko/sigfmt/internal/config"
	"github.com/vsfedorenko/sigfmt/internal/domain"
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

	baseIndent := b.renderer.GetIndent(fset, sig.Start)
	indentUnit := "\t"
	if strings.Contains(baseIndent, " ") && !strings.Contains(baseIndent, "\t") {
		indentUnit = "    "
	}
	paramIndent := baseIndent + indentUnit
	baseIndentLen := b.renderer.VisualLength(baseIndent)

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

	prefixLen := baseIndentLen + b.renderer.VisualLength(sig.OneLineText) - b.renderer.VisualLength(b.renderer.FieldList(fset, fl, openBracket, closeBracket))
	if sig.FuncType.Results != nil {
		prefixLen -= b.renderer.VisualLength(b.renderer.Results(fset, sig.FuncType.Results))
	}
	prefixLen += b.renderer.VisualLength(openBracket)

	writer := b.newBuilderLineWriter(indent, openBracket, prefixLen)

	i := 0
	for i < len(fl.List) {
		matchedGroupLen := 0
		for _, group := range b.config.ParamGroups {
			if b.matchesGroup(fset, fl.List, i, group) {
				matchedGroupLen = len(group)
				break
			}
		}

		fieldsToProcess := 1
		isGroupMatch := false
		if matchedGroupLen > 0 {
			fieldsToProcess = matchedGroupLen
			isGroupMatch = true
		}

		var fieldStrs []string
		for k := 0; k < fieldsToProcess; k++ {
			field := fl.List[i+k]
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
			fieldStr += b.renderer.Node(fset, field.Type)
			fieldStrs = append(fieldStrs, fieldStr)
		}

		for _, s := range fieldStrs {
			writer.add(s)
		}

		if isGroupMatch && (i+fieldsToProcess < len(fl.List)) {
			writer.forceBreak()
		}

		i += fieldsToProcess
	}

	return writer.String(closeBracket)
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
		indentVisLen:  b.renderer.VisualLength(indent),
		openBracket:   openBracket,
		currentVisLen: initialVisLen,
	}
	l.currentLine.WriteString(openBracket)
	return l
}

func (lb *builderLineWriter) add(text string) {
	textVisLen := lb.b.renderer.VisualLength(text)
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

	lb.currentLine.WriteString(text)
	lb.currentVisLen += textVisLen
}

func (lb *builderLineWriter) forceBreak() {
	lb.currentLine.WriteString(",")
	lb.lines = append(lb.lines, lb.currentLine.String())
	lb.currentLine.Reset()
	lb.currentLine.WriteString(lb.indent)
	lb.currentVisLen = lb.indentVisLen
}

func (lb *builderLineWriter) String(closeBracket string) string {
	lb.currentLine.WriteString(closeBracket)
	lb.lines = append(lb.lines, lb.currentLine.String())
	return strings.Join(lb.lines, "\n")
}