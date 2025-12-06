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
	analyzerName              = "funcwrap"
	analyzerDoc               = "Checks if multi-line function signatures can be collapsed to one line or formatted better"
	diagnosticMessage         = "Signature fits in one line"
	diagnosticMessageReformat = "Signature can be formatted better"
	fixMessage                = "Collapse to one line"
	fixMessageReformat        = "Reformat with grouped parameters"
)

func init() {
	register.Plugin(analyzerName, New)
}

type PluginLineWrap struct {
	settings struct{ MaxLineLen int }
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
	if s, ok := settings.(map[string]any); ok {
		if v, _ := s["max-line-len"].(float64); v > 0 {
			p.settings.MaxLineLen = int(v)
		}
	}
	if p.settings.MaxLineLen == 0 {
		p.settings.MaxLineLen = defaultMaxLineLen
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

	action := p.checkSignature(pass.Fset, sig)
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

	action := p.checkSignature(pass.Fset, sig)
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

		action := p.checkSignature(pass.Fset, sig)
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

		action := p.checkSignature(pass.Fset, sig)
		if action != "" {
			p.report(pass, sig, action)
		}
	}
}

// checkSignature проверяет, нужно ли изменить форматирование сигнатуры
// Возвращает "collapse" если нужно схлопнуть в одну строку,
// "reformat" если нужно улучшить многострочное форматирование,
// или "" если изменений не требуется
func (p *PluginLineWrap) checkSignature(fset *token.FileSet, sig *signatureInfo) string {
	startLine := fset.Position(sig.start).Line
	endLine := fset.Position(sig.end).Line

	// Если уже в одну строку, ничего не делаем
	if startLine == endLine {
		return ""
	}

	// fmt.Printf("DEBUG: %s len=%d max=%d\n", sig.name, len(sig.oneLineText), p.settings.MaxLineLen)

	// Если помещается в одну строку, предлагаем схлопнуть
	if len(sig.oneLineText) <= p.settings.MaxLineLen {
		return "collapse"
	}

	// Если не помещается в одну строку, проверяем, можно ли улучшить форматирование
	if p.shouldReformat(fset, sig) {
		sig.reformattedText = p.buildReformattedSignature(fset, sig)
		return "reformat"
	}

	return ""
}

// shouldReformat проверяет, нужно ли переформатировать многострочную сигнатуру
func (p *PluginLineWrap) shouldReformat(fset *token.FileSet, sig *signatureInfo) bool {
	// Проверяем, что есть funcType для анализа
	if sig.funcType == nil || sig.funcType.Params == nil {
		return false
	}

	// Проверяем, что параметры действительно разбиты по строкам
	// (каждый параметр на отдельной строке)
	params := sig.funcType.Params.List
	if len(params) <= 1 {
		return false
	}

	// Если это метод интерфейса или поле структуры, мы всегда хотим попробовать
	// упаковать их компактнее (агрессивное форматирование).
	if sig.isInterfaceMethod || sig.isStructField {
		return true
	}

	// Для обычных функций (standalone) сохраняем старое поведение:
	// если параметры уже на отдельных строках, не трогаем их.
	firstLine := fset.Position(params[0].Pos()).Line
	allOnSeparateLines := true
	for i := 1; i < len(params); i++ {
		if fset.Position(params[i].Pos()).Line == firstLine {
			allOnSeparateLines = false
			break
		}
		firstLine = fset.Position(params[i].Pos()).Line
	}

	// Если параметры уже на отдельных строках, по умолчанию считаем,
	// что форматирование уже корректное (или пользователь так захотел)
	// и не пытаемся его "улучшить" (сжать).
	return !allOnSeparateLines
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

	// Получаем отступ для продолжения строк (используем табуляцию)
	indent := "\t"

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
			currentLineLen = 8 + len(fieldStr) // 8 for tab width
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

// extractFuncDeclSignature извлекает информацию о сигнатуре функции
func (p *PluginLineWrap) extractFuncDeclSignature(fset *token.FileSet, decl *ast.FuncDecl) *signatureInfo {
	start := decl.Type.Func
	end := decl.Type.Params.End()
	diagPos := decl.Type.Params.Closing

	if decl.Type.Results != nil {
		end = decl.Type.Results.End()
		if decl.Type.Results.Closing.IsValid() {
			diagPos = decl.Type.Results.Closing
		}
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:         start,
		end:           end,
		diagPos:       diagPos,
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
	diagPos := lit.Type.Params.Closing

	if lit.Type.Results != nil {
		end = lit.Type.Results.End()
		if lit.Type.Results.Closing.IsValid() {
			diagPos = lit.Type.Results.Closing
		}
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:         start,
		end:           end,
		diagPos:       diagPos,
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
	diagPos := ft.Params.Closing

	if ft.Results != nil {
		end = ft.Results.End()
		if ft.Results.Closing.IsValid() {
			diagPos = ft.Results.Closing
		}
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:             start,
		end:               end,
		diagPos:           diagPos,
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
	diagPos := ft.Params.Closing

	if ft.Results != nil {
		end = ft.Results.End()
		if ft.Results.Closing.IsValid() {
			diagPos = ft.Results.Closing
		}
	}

	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &signatureInfo{
		start:         start,
		end:           end,
		diagPos:       diagPos,
		oneLineText:   p.buildStructFieldSignature(fset, name.Name, ft),
		funcType:      ft,
		receiver:      nil,
		name:          name.Name,
		isStructField: true,
	}
}

// buildFuncDeclSignature собирает односрочную версию сигнатуры функции
func (p *PluginLineWrap) buildFuncDeclSignature(fset *token.FileSet, decl *ast.FuncDecl) string {
	var sb strings.Builder

	sb.WriteString("func ")

	// Receiver (для методов)
	if decl.Recv != nil {
		sb.WriteString(p.renderFieldList(fset, decl.Recv, "(", ")"))
		sb.WriteString(" ")
	}

	// Имя функции
	sb.WriteString(decl.Name.Name)

	// Type parameters (дженерики)
	if decl.Type.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, decl.Type.TypeParams, "[", "]"))
	}

	// Параметры
	sb.WriteString(p.renderFieldList(fset, decl.Type.Params, "(", ")"))

	// Результаты
	if decl.Type.Results != nil {
		sb.WriteString(p.renderResults(fset, decl.Type.Results))
	}

	return sb.String()
}

// buildFuncLitSignature собирает односрочную версию сигнатуры функционального литерала
func (p *PluginLineWrap) buildFuncLitSignature(fset *token.FileSet, ft *ast.FuncType) string {
	var sb strings.Builder

	sb.WriteString("func")

	// Type parameters (дженерики)
	if ft.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, ft.TypeParams, "[", "]"))
	}

	// Параметры
	sb.WriteString(p.renderFieldList(fset, ft.Params, "(", ")"))

	// Результаты
	if ft.Results != nil {
		sb.WriteString(p.renderResults(fset, ft.Results))
	}

	return sb.String()
}

// buildMethodSignature собирает односрочную версию сигнатуры метода
func (p *PluginLineWrap) buildMethodSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	var sb strings.Builder

	sb.WriteString(name)

	// Параметры
	sb.WriteString(p.renderFieldList(fset, ft.Params, "(", ")"))

	// Результаты
	if ft.Results != nil {
		sb.WriteString(p.renderResults(fset, ft.Results))
	}

	return sb.String()
}

// buildStructFieldSignature собирает односрочную версию поля структуры с типом func
func (p *PluginLineWrap) buildStructFieldSignature(fset *token.FileSet, name string, ft *ast.FuncType) string {
	var sb strings.Builder

	sb.WriteString(name)
	sb.WriteString(" func")

	// Type parameters (дженерики)
	if ft.TypeParams != nil {
		sb.WriteString(p.renderFieldList(fset, ft.TypeParams, "[", "]"))
	}

	// Параметры
	sb.WriteString(p.renderFieldList(fset, ft.Params, "(", ")"))

	// Результаты
	if ft.Results != nil {
		sb.WriteString(p.renderResults(fset, ft.Results))
	}

	return sb.String()
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
	case "collapse":
		message = diagnosticMessage
		fixMsg = fixMessage
		newText = []byte(sig.oneLineText)
	case "reformat":
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
