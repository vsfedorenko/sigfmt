package sigfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseParamGroupsFlag(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace-only returns nil",
			input: "   ",
			want:  nil,
		},
		{
			name:  "single group single type",
			input: "context.Context",
			want:  [][]string{{"context.Context"}},
		},
		{
			name:  "single group multiple types",
			input: "context.Context,error",
			want:  [][]string{{"context.Context", "error"}},
		},
		{
			name:  "multiple groups",
			input: "context.Context,error;io.Reader,io.Writer",
			want: [][]string{
				{"context.Context", "error"},
				{"io.Reader", "io.Writer"},
			},
		},
		{
			name:  "groups with whitespace are trimmed",
			input: " context.Context , error ; io.Reader , io.Writer ",
			want: [][]string{
				{"context.Context", "error"},
				{"io.Reader", "io.Writer"},
			},
		},
		{
			name:  "trailing semicolon ignored",
			input: "a,b;",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "trailing comma in group ignored",
			input: "a,b,",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "empty groups skipped",
			input: ";;a,b;;",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "single type per group across many groups",
			input: "int;string;bool",
			want: [][]string{
				{"int"},
				{"string"},
				{"bool"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseParamGroupsFlag(tt.input)
			assert.True(t, equalGroups(got, tt.want), "parseParamGroupsFlag(%q) = %v, want %v", tt.input, got, tt.want)
		})
	}
}

func TestNewAnalyzerFlags(t *testing.T) {
	a := NewAnalyzer()

	// Verify all expected flags are registered.
	expectedFlags := []string{
		"max-line-len",
		"tab-width",
		"pack-struct-fields",
		"pack-interface-methods",
		"param-groups",
	}
	for _, name := range expectedFlags {
		assert.NotNil(t, a.Flags.Lookup(name), "Analyzer.Flags missing flag: %s", name)
	}
}

// equalGroups compares two [][]string for deep equality, treating nil and
// empty slice as equivalent.
func equalGroups(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
