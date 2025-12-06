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
    defaultMaxLineLen = 120
    analyzerName      = "linewrap"
    analyzerDoc       = "Checks if multi-line function signatures can be collapsed to one line"
    diagnosticMessage = "Signature fits in one line"
    fixMessage        = "Collapse to one line"
)

type PluginLineWrap struct {
    settings struct{ MaxLineLen int }
}

// signatureInfo содержит информацию о сигнатуре для проверки
type signatureInfo struct {
    start       token.Pos  // начало замены
    end         token.Pos  // конец замены
    diagPos     token.Pos  // позиция для диагностики
    oneLineText string     // текст односрочной версии
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

    if p.shouldCollapse(pass.Fset, sig) {
        p.report(pass, sig)
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

    if p.shouldCollapse(pass.Fset, sig) {
        p.report(pass, sig)
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

        if p.shouldCollapse(pass.Fset, sig) {
            p.report(pass, sig)
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

        if p.shouldCollapse(pass.Fset, sig) {
            p.report(pass, sig)
        }
    }
}

// shouldCollapse проверяет, должна ли сигнатура быть схлопнута
func (p *PluginLineWrap) shouldCollapse(fset *token.FileSet, sig *signatureInfo) bool {
    // Проверяем длину односрочной версии
    if len(sig.oneLineText) > p.settings.MaxLineLen {
        return false
    }

    // Проверяем, что сигнатура не в одну строку
    startLine := fset.Position(sig.start).Line
    endLine := fset.Position(sig.end).Line
    return startLine != endLine
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
        start:       start,
        end:         end,
        diagPos:     diagPos,
        oneLineText: p.buildFuncDeclSignature(fset, decl),
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
        start:       start,
        end:         end,
        diagPos:     diagPos,
        oneLineText: p.buildFuncLitSignature(fset, lit.Type),
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
        start:       start,
        end:         end,
        diagPos:     diagPos,
        oneLineText: p.buildMethodSignature(fset, name.Name, ft),
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
        start:       start,
        end:         end,
        diagPos:     diagPos,
        oneLineText: p.buildStructFieldSignature(fset, name.Name, ft),
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
func (p *PluginLineWrap) report(pass *analysis.Pass, sig *signatureInfo) {
    pass.Report(analysis.Diagnostic{
        Pos:     sig.diagPos,
        End:     sig.diagPos,
        Message: diagnosticMessage,
        SuggestedFixes: []analysis.SuggestedFix{{
            Message: fixMessage,
            TextEdits: []analysis.TextEdit{{
                Pos:     sig.start,
                End:     sig.end,
                NewText: []byte(sig.oneLineText),
            }},
        }},
    })
}
