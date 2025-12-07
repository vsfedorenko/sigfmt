package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSettings(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected Settings
	}{
		{
			name:  "DefaultSettings",
			input: nil,
			expected: Settings{
				MaxLineLen:           DefaultMaxLineLen,
				TabWidth:             DefaultTabWidth,
				PackStructFields:     true,
				PackInterfaceMethods: true,
			},
		},
		{
			name:  "EmptyMap",
			input: map[string]interface{}{},
			expected: Settings{
				MaxLineLen:           DefaultMaxLineLen,
				TabWidth:             DefaultTabWidth,
				PackStructFields:     true,
				PackInterfaceMethods: true,
			},
		},
		{
			name: "CustomSettings",
			input: map[string]interface{}{
				"max-line-len":           float64(100),
				"tab-width":              float64(4),
				"pack-struct-fields":     false,
				"pack-interface-methods": true,
				"param-groups": []interface{}{
					[]interface{}{"context.Context", "*sql.Tx"},
					[]interface{}{"io.Reader"},
				},
			},
			expected: Settings{
				MaxLineLen:           100,
				TabWidth:             4,
				PackStructFields:     false,
				PackInterfaceMethods: true,
				ParamGroups:          [][]string{{"context.Context", "*sql.Tx"}, {"io.Reader"}},
			},
		},
		{
			name: "InvalidTypes",
			input: map[string]interface{}{
				"max-line-len":           "not-a-number",
				"tab-width":              float64(-1),
				"pack-struct-fields":     "not-a-bool",
				"param-groups":           "not-a-list",
				"pack-interface-methods": nil,
			},
			expected: Settings{
				MaxLineLen:           DefaultMaxLineLen,
				TabWidth:             DefaultTabWidth,
				PackStructFields:     true,
				PackInterfaceMethods: true,
			},
		},
		{
			name: "MixedParamGroups",
			input: map[string]interface{}{
				"param-groups": []interface{}{
					[]interface{}{"context.Context"},
					"invalid-group-format",
					[]interface{}{"io.Writer"},
				},
			},
			expected: Settings{
				MaxLineLen:           DefaultMaxLineLen,
				TabWidth:             DefaultTabWidth,
				PackStructFields:     true,
				PackInterfaceMethods: true,
				ParamGroups:          [][]string{{"context.Context"}, {"io.Writer"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, New(tt.input))
		})
	}
}
