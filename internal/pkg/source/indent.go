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

	// Scan backwards to the start of the physical line containing offset.
	// A plain scan is used instead of File.LineStart(File.Line(pos)) because
	// //line directives make the token line table sparse: the position's
	// logical line number can exceed the number of physical lines, and
	// LineStart panics on such input (go vet tolerates //line; so must we).
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
