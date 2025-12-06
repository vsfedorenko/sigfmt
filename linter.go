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
	analyzerName              = "funcwrap"
	analyzerDoc               = "Checks if multi-line function signatures can be collapsed to one line or formatted better"
	diagnosticMessage         = "Signature fits in one line"
	diagnosticMessageReformat = "Signature can be formatted better"
	fixMessage                = "Collapse to one line"
	fixMessageReformat        = "Reformat with grouped parameters"

	// Action types for signature formatting
	actionCollapse = "collapse"
	actionReformat = "reformat"
)

func init() {
	register.Plugin(analyzerName, New)
}

// Settings содержит настройки линтера
type Settings struct {
	MaxLineLen           int
	TabWidth             int
	PackStructFields     bool
	PackInterfaceMethods bool
}

type PluginLineWrap struct {
	settings Settings
}

// signatureInfo содержит информацию о сигнатуре для проверки
type signatureInfo struct {
	start             token.Pos      // начало замены
	end               token.Pos      // конец замены
	diagPos           token.Pos      // позиция для диагностики
	oneLineText       string         // текст односрочной версии
	reformattedText   string         // текст улучшенной многострочной версии (если не помещается в одну строку)
	funcType          *ast.FuncType  // тип функции для генерации форматированной версии
	receiver          *ast.FieldList // receiver для методов (может быть nil)
	name              string         // имя функции/метода
	isStructField     bool           // true если это поле структуры с типом func
	isInterfaceMethod bool           // true если это метод интерфейса
}

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

func (p *PluginLineWrap) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{{
		Name: analyzerName,
		Doc:  analyzerDoc,
		Run:  p.run,
	}}, nil
}

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

		// Поля структуры могут быть без имени (анонимные), пропускаем их
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

// checkSignature проверяет, нужно ли изменить форматирование сигнатуры.
// Возвращает actionCollapse если нужно схлопнуть в одну строку,
// actionReformat если нужно улучшить многострочное форматирование,
// или "" если изменений не требуется
func (p *PluginLineWrap) checkSignature(pass *analysis.Pass, sig *signatureInfo) string {
	fset := pass.Fset
	startLine := fset.Position(sig.start).Line
	endLine := fset.Position(sig.end).Line

	// Если однострочная версия сигнатуры помещается в MaxLineLen
	if p.visualLength(sig.oneLineText) <= p.settings.MaxLineLen {
		// Если сигнатура сейчас многострочная, но помещается в одну, предлагаем схлопнуть
		if startLine != endLine {
			return actionCollapse
		}
		// Иначе (уже в одну строку и помещается), ничего не делаем
		return ""
	}

	// Если не помещается в одну строку, проверяем, можно ли улучшить форматирование
	if p.shouldReformat(pass.Fset, sig) {
		sig.reformattedText = p.buildReformattedSignature(pass.Fset, sig)
		return actionReformat
	}

	return ""
}

// visualLength вычисляет визуальную длину строки с учетом табуляции
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

// shouldReformat проверяет, нужно ли переформатировать многострочную сигнатуру.
// Возвращает true, если сигнатуру можно упаковать компактнее.
func (p *PluginLineWrap) shouldReformat(fset *token.FileSet, sig *signatureInfo) bool {
	if sig.funcType == nil || sig.funcType.Params == nil {
		return false
	}

	// Не реформатируем однострочные сигнатуры
	if fset.Position(sig.start).Line == fset.Position(sig.end).Line {
		return false
	}

	params := sig.funcType.Params.List
	if len(params) <= 1 {
		return false
	}

	// Для методов интерфейсов и полей структур применяем агрессивное форматирование
	// (упаковка параметров), если это разрешено настройками
	if sig.isInterfaceMethod && p.settings.PackInterfaceMethods {
		return true
	}
	if sig.isStructField && p.settings.PackStructFields {
		return true
	}

	// Для обычных функций: реформатируем только если параметры НЕ на отдельных строках
	// (т.е. если они уже аккуратно разбиты построчно, оставляем как есть)
	return !p.areParamsOnSeparateLines(fset, params)
}

// areParamsOnSeparateLines проверяет, находится ли каждый параметр на отдельной строке
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

// buildReformattedSignature генерирует улучшенное многострочное форматирование
func (p *PluginLineWrap) buildReformattedSignature(fset *token.FileSet, sig *signatureInfo) string {
	if sig.funcType == nil {
		return ""
	}

	var sb strings.Builder

	// Префикс (func, имя, receiver и т.д.)
	if sig.isStructField {
		sb.WriteString(sig.name)
		sb.WriteString(" func")
	} else if sig.isInterfaceMethod {
		sb.WriteString(sig.name)
	} else {
		sb.WriteString("func ")
		if sig.receiver != nil {
			sb.WriteString(p.renderFieldList(fset, sig.receiver, "(", ")"))
			sb.WriteString(" ")
		}
		if sig.name != "" {
			sb.WriteString(sig.name)
		}
	}

	// Type parameters (дженерики)
	if sig.funcType.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, sig.funcType.TypeParams, "[", "]"))
	}

	// Группируем параметры
	sb.WriteString(p.renderFieldListGrouped(fset, sig.funcType.Params, sig, "(", ")"))

	// Результаты
	if sig.funcType.Results != nil {
		sb.WriteString(p.renderResults(fset, sig.funcType.Results))
	}

	return sb.String()
}

// renderFieldListGrouped рендерит параметры с группировкой по строкам
func (p *PluginLineWrap) renderFieldListGrouped(fset *token.FileSet, fl *ast.FieldList, sig *signatureInfo, openBracket, closeBracket string) string {
	if fl == nil || len(fl.List) == 0 {
		return openBracket + closeBracket
	}

	// Рассчитываем префикс (сколько места занимает начало строки)
	prefixLen := len(sig.oneLineText) - len(p.renderFieldList(fset, fl, openBracket, closeBracket))
	if sig.funcType.Results != nil {
		prefixLen -= len(p.renderResults(fset, sig.funcType.Results))
	}

	// Получаем отступ для продолжения строк
	indent := "\t" // Standard Go indentation

	var lines []string
	var currentLine strings.Builder
	currentLine.WriteString(openBracket)
	currentLineLen := prefixLen + 1 // +1 для открывающей скобки

	for i, field := range fl.List {
		// Рендерим текущее поле
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

		// Проверяем, поместится ли поле на текущей строке
		testLen := currentLineLen
		if currentLine.Len() > len(openBracket) {
			testLen += 2 // для ", "
		}
		testLen += len(fieldStr)

		// Если не первый параметр и не помещается, начинаем новую строку
		if i > 0 && testLen > p.settings.MaxLineLen && currentLine.Len() > len(openBracket) {
			currentLine.WriteString(",")
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(indent)
			currentLine.WriteString(fieldStr)
			currentLineLen = p.settings.TabWidth + len(fieldStr)
		} else {
			// Добавляем на текущую строку
			if currentLine.Len() > len(openBracket) {
				currentLine.WriteString(", ")
			}
			currentLine.WriteString(fieldStr)
			currentLineLen = testLen
		}
	}

	// Добавляем закрывающую скобку на той же строке что и последний параметр
	currentLine.WriteString(closeBracket)
	lines = append(lines, currentLine.String())

	return strings.Join(lines, "\n")
}

// computeDiagPos вычисляет позицию для диагностики (закрывающая скобка параметров или результатов)
func computeDiagPos(params *ast.FieldList, results *ast.FieldList) token.Pos {
	diagPos := params.Closing
	if results != nil && results.Closing.IsValid() {
		diagPos = results.Closing
	}
	return diagPos
}

// extractFuncDeclSignature извлекает информацию о сигнатуре функции
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

// extractFuncLitSignature извлекает информацию о сигнатуре функционального литерала
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

// extractMethodSignature извлекает информацию о сигнатуре метода интерфейса
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

// extractStructFieldSignature извлекает информацию о сигнатуре поля структуры с типом func
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

// buildSignature - универсальная функция для построения односрочных сигнатур
func (p *PluginLineWrap) buildSignature(fset *token.FileSet, prefix string, ft *ast.FuncType, results *ast.FieldList) string {
	var sb strings.Builder
	sb.WriteString(prefix)

	// Type parameters (дженерики)
	if ft.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, ft.TypeParams, "[", "]"))
	}

	// Параметры
	sb.WriteString(p.renderFieldList(fset, ft.Params, "(", ")"))

	// Результаты
	if results != nil {
		sb.WriteString(p.renderResults(fset, results))
	}

	return sb.String()
}

// buildFuncDeclSignature собирает односрочную версию сигнатуры функции
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

// buildFuncLitSignature собирает односрочную версию сигнатуры функционального литерала
func (p *PluginLineWrap) buildFuncLitSignature(fset *token.FileSet, ft *ast.FuncType) string {
	return p.buildSignature(fset, "func", ft, ft.Results)
}

// buildMethodSignature собирает односрочную версию сигнатуры метода
func (p *PluginLineWrap) buildMethodSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return p.buildSignature(fset, name, ft, ft.Results)
}

// buildStructFieldSignature собирает односрочную версию поля структуры с типом func
func (p *PluginLineWrap) buildStructFieldSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	return p.buildSignature(fset, name+" func", ft, ft.Results)
}

// renderResults рендерит возвращаемые значения
func (p *PluginLineWrap) renderResults(fset *token.FileSet, results *ast.FieldList) string {
	if results == nil || len(results.List) == 0 {
		return ""
	}

	// Если результат один и без имени, не оборачиваем в скобки
	if len(results.List) == 1 && len(results.List[0].Names) == 0 {
		return " " + p.renderNode(fset, results.List[0].Type)
	}

	// Иначе оборачиваем в скобки
	return " " + p.renderFieldList(fset, results, "(", ")")
}

// renderNode рендерит AST узел в строку
func (p *PluginLineWrap) renderNode(fset *token.FileSet, n ast.Node) string {
	if n == nil {
		return ""
	}

	// Специальная обработка для FieldList
	if fl, ok := n.(*ast.FieldList); ok {
		return p.renderFieldList(fset, fl, "(", ")")
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return ""
	}

	// Схлопываем все пробелы и переносы строк в один пробел
	return strings.Join(strings.Fields(buf.String()), " ")
}

// renderFieldList рендерит список полей с заданными скобками
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

		// Рендерим имена полей
		for j, name := range field.Names {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(name.Name)
		}

		// Если есть имена, добавляем пробел перед типом
		if len(field.Names) > 0 {
			sb.WriteString(" ")
		}

		// Рендерим тип
		sb.WriteString(p.renderNode(fset, field.Type))
	}

	sb.WriteString(closeBracket)
	return sb.String()
}

// report сообщает о найденной проблеме с предложением исправления
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
