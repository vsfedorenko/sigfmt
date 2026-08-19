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
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.source), 0o644))

			// Parse file
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tmpFile, nil, 0)
			require.NoError(t, err)

			// Use the position of the first declaration
			require.NotEmpty(t, f.Decls, "No declarations found")

			// For first test (func), use func position
			// For others (struct fields), find the field position
			var pos token.Pos
			if tt.name == "NoIndent" {
				pos = f.Decls[0].Pos()
			} else {
				// Find struct field position
				if typeDecl, ok := f.Decls[0].(*ast.GenDecl); ok && len(typeDecl.Specs) > 0 {
					if typeSpec, ok := typeDecl.Specs[0].(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok && len(structType.Fields.List) > 0 {
							pos = structType.Fields.List[0].Pos()
						}
					}
				}
			}

			require.True(t, pos.IsValid(), "Could not find valid position")

			got := GetIndent(fset, pos)

			assert.Equal(t, tt.want, got, "GetIndent()")
		})
	}
}
