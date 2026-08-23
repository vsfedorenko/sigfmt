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

// parseFixture writes src to a temp file and parses it, mirroring how the
// analyzer sees positions (a FileSet backed by real file offsets).
func parseFixture(t *testing.T, src string) (string, *token.FileSet, *ast.File) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.go")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)
	return path, fset, f
}

func TestGetIndent(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "NoIndent",
			source: "package main\n\nfunc main() {}",
			want:   "",
		},
		{
			name:   "OneTab",
			source: "package main\n\ntype S struct {\n\tField int\n}",
			want:   "\t",
		},
		{
			name:   "TwoTabs",
			source: "package main\n\ntype S struct {\n\t\tField int\n}",
			want:   "\t\t",
		},
		{
			name:   "FourSpaces",
			source: "package main\n\ntype S struct {\n    Field int\n}",
			want:   "    ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, fset, f := parseFixture(t, tt.source)
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			require.NotEmpty(t, f.Decls, "No declarations found")

			// For the func fixture use the decl position; for struct
			// fixtures find the first field position.
			var pos token.Pos
			if tt.name == "NoIndent" {
				pos = f.Decls[0].Pos()
			} else {
				if typeDecl, ok := f.Decls[0].(*ast.GenDecl); ok && len(typeDecl.Specs) > 0 {
					if typeSpec, ok := typeDecl.Specs[0].(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok && len(structType.Fields.List) > 0 {
							pos = structType.Fields.List[0].Pos()
						}
					}
				}
			}

			require.True(t, pos.IsValid(), "Could not find valid position")

			assert.Equal(t, tt.want, Indent(fset, content, pos), "Indent()")
		})
	}
}

// TestIndentEdgeCases covers the defensive branches of Indent and Loader:
// unknown position (file missing from the FileSet), unreadable file,
// position past EOF, and an indent prefix that stops at the first
// non-whitespace byte.
func TestGetIndentEdgeCases(t *testing.T) {
	t.Run("UnknownPosition", func(t *testing.T) {
		// A FileSet with no files cannot resolve any position.
		fset := token.NewFileSet()
		assert.Empty(t, Indent(fset, nil, token.Pos(100)), "Indent(unknown position) must be empty")
	})

	t.Run("UnreadableFile", func(t *testing.T) {
		// The Loader caches the read failure as File{OK:false}; indent
		// lookups must degrade to "" exactly like a nil content.
		path, fset, f := parseFixture(t, "package main\n\ntype S struct {\n\tField int\n}\n")
		loader := NewLoader(fset, func(string) ([]byte, error) { return nil, os.ErrNotExist })
		file := loader.Load(f.Pos())
		require.False(t, file.OK, "loader must mark the file unreadable")
		assert.Empty(t, file.Indent(f.Pos()), "Indent(unreadable file) must be empty")
		assert.Empty(t, Indent(fset, file.Content, f.Pos()), "Indent(no content) must be empty")
		_ = path
	})

	t.Run("PositionPastEOF", func(t *testing.T) {
		// A position whose offset is beyond the (shorter) content length.
		_, fset, f := parseFixture(t, "package main\n\ntype S struct {\n\tVeryLongFieldName int\n}\n")
		content := []byte("package main\n")
		end := f.End() // last valid offset of the ORIGINAL content
		assert.Empty(t, Indent(fset, content, end), "Indent(past EOF) must be empty")
	})

	t.Run("IndentStopsAtNonWhitespace", func(t *testing.T) {
		// The prefix between the line start and the position contains a
		// non-whitespace byte (a label on the same line); the indent loop
		// must stop at it rather than copying it into the indent.
		_, fset, f := parseFixture(t, "package main\n\nfunc main() {\nloop:\n\tfor {\n\t\tbreak loop\n\t}\n}\n")
		var pos token.Pos
		ast.Inspect(f, func(n ast.Node) bool {
			if bs, ok := n.(*ast.BranchStmt); ok && bs.Tok == token.BREAK {
				pos = bs.Pos()
				return false
			}
			return true
		})
		require.True(t, pos.IsValid(), "break statement not found")
		content, err := os.ReadFile(fset.Position(pos).Filename)
		require.NoError(t, err)
		assert.Equal(t, "\t\t", Indent(fset, content, pos), "Indent(break indent)")
	})
}
