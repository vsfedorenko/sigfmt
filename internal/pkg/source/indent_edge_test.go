package source

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetIndentEdgeCases covers the defensive branches of GetIndent that the
// happy-path table test does not reach: unknown position (file missing from
// the FileSet), unreadable file, position past EOF, and an indent prefix
// that stops at the first non-whitespace byte.
func TestGetIndentEdgeCases(t *testing.T) {
	writeFixture := func(t *testing.T, src string) (string, *token.FileSet, *ast.File) {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "fixture.go")
		require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, err)
		return path, fset, f
	}

	t.Run("UnknownPosition", func(t *testing.T) {
		// A FileSet with no files cannot resolve any position.
		fset := token.NewFileSet()
		assert.Empty(t, GetIndent(fset, token.Pos(1000)), "GetIndent(unknown pos) must be empty")
	})

	t.Run("UnreadableFile", func(t *testing.T) {
		path, fset, f := writeFixture(t, "package main\n\ntype S struct {\n\tField int\n}\n")
		require.NoError(t, os.Remove(path))
		assert.Empty(t, GetIndent(fset, f.Pos()), "GetIndent(unreadable file) must be empty")
	})

	t.Run("PositionPastEOF", func(t *testing.T) {
		// Shrink the file after parsing so a late position's offset is
		// beyond the (now shorter) content length.
		path, fset, f := writeFixture(t, "package main\n\ntype S struct {\n\tVeryLongFieldName int\n}\n")
		require.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))
		end := f.End() // last valid offset of the ORIGINAL content
		assert.Empty(t, GetIndent(fset, end), "GetIndent(past EOF) must be empty")
	})

	t.Run("IndentStopsAtNonWhitespace", func(t *testing.T) {
		// The prefix between the line start and the position contains a
		// non-whitespace byte (a label on the same line); the indent loop
		// must stop at it rather than copying it into the indent.
		_, fset, f := writeFixture(t, "package main\n\nfunc main() {\nloop:\n	for {\n		break loop\n	}\n}\n")
		var pos token.Pos
		ast.Inspect(f, func(n ast.Node) bool {
			if bs, ok := n.(*ast.BranchStmt); ok && bs.Tok == token.BREAK {
				pos = bs.Pos()
				return false
			}
			return true
		})
		require.True(t, pos.IsValid(), "break statement not found")
		assert.Equal(t, "		", GetIndent(fset, pos), "GetIndent(break indent)")
	})
}
