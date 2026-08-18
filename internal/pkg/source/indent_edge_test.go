package source

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
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
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		return path, fset, f
	}

	t.Run("UnknownPosition", func(t *testing.T) {
		// A FileSet with no files cannot resolve any position.
		fset := token.NewFileSet()
		if got := GetIndent(fset, token.Pos(1000)); got != "" {
			t.Errorf("GetIndent(unknown pos) = %q, want empty", got)
		}
	})

	t.Run("UnreadableFile", func(t *testing.T) {
		path, fset, f := writeFixture(t, "package main\n\ntype S struct {\n\tField int\n}\n")
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if got := GetIndent(fset, f.Pos()); got != "" {
			t.Errorf("GetIndent(unreadable file) = %q, want empty", got)
		}
	})

	t.Run("PositionPastEOF", func(t *testing.T) {
		// Shrink the file after parsing so a late position's offset is
		// beyond the (now shorter) content length.
		path, fset, f := writeFixture(t, "package main\n\ntype S struct {\n\tVeryLongFieldName int\n}\n")
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		end := f.End() // last valid offset of the ORIGINAL content
		if got := GetIndent(fset, end); got != "" {
			t.Errorf("GetIndent(past EOF) = %q, want empty", got)
		}
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
		if !pos.IsValid() {
			t.Fatal("break statement not found")
		}
		if got := GetIndent(fset, pos); got != "		" {
			t.Errorf("GetIndent(break indent) = %q, want %q", got, "		")
		}
	})
}
