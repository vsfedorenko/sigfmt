package source

import (
	"go/token"
	"os"
	"strings"
)

// GetIndent reads the source file to find the indentation of the line containing pos.
// Returns the leading whitespace (spaces and tabs) at the beginning of the line.
func GetIndent(fset *token.FileSet, pos token.Pos) string {
	f := fset.File(pos)
	if f == nil {
		return ""
	}

	content, err := os.ReadFile(f.Name())
	if err != nil {
		return ""
	}

	offset := f.Offset(pos)
	if offset >= len(content) {
		return ""
	}

	lineStart := f.LineStart(f.Line(pos))
	lineStartOffset := f.Offset(lineStart)

	if lineStartOffset > offset {
		return ""
	}

	prefix := content[lineStartOffset:offset]

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
