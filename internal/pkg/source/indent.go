// Package source answers questions about a file's raw bytes that the AST
// alone cannot: indentation of a position and the text of a range. All
// functions are pure over content that the caller already holds; nothing
// here touches the filesystem.
package source

import (
	"go/token"
	"strings"
)

// Indent returns the leading whitespace (spaces and tabs) of the physical
// line containing pos. The content is the file's bytes the position
// resolves into — the caller loads them once per file and reuses them for
// every signature inside.
func Indent(fset *token.FileSet, content []byte, pos token.Pos) string {
	f := fset.File(pos)
	if f == nil {
		return ""
	}

	offset := f.Offset(pos)
	if offset >= len(content) {
		return ""
	}

	return indentFromOffset(content, offset)
}

// indentFromOffset scans backwards from offset to the start of the physical
// line and returns the line's leading whitespace prefix before offset.
//
// A plain scan is used instead of File.LineStart(File.Line(pos)) because
// //line directives make the token line table sparse: the position's
// logical line number can exceed the number of physical lines, and
// LineStart panics on such input (go vet tolerates //line; so must we).
func indentFromOffset(content []byte, offset int) string {
	lineStart := offset
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}
	prefix := content[lineStart:offset]

	var indent strings.Builder
	for _, b := range prefix {
		if b == ' ' || b == '\t' {
			indent.WriteByte(b)
		} else {
			break
		}
	}
	return indent.String()
}
