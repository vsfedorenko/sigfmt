package linters

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestLineWrap_Features проверяет корректность работы линтера с различными языковыми конструкциями
// (функции, интерфейсы, структуры) при стандартной длине строки (120).
func TestLineWrap_Features(t *testing.T) {
	runTest(t, 120, "features")
}

// TestLineWrap_Limits проверяет работу линтера с различными ограничениями длины строки.
func TestLineWrap_Limits(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		pkg    string
	}{
		{"Limit_80", 80, "limits/length80"},
		{"Limit_100", 100, "limits/length100"},
		{"Limit_120", 120, "limits/length120"},
		{"Limit_140", 140, "limits/length140"},
		{"Limit_160", 160, "limits/length160"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTest(t, tt.limit, tt.pkg)
		})
	}
}

// helper для запуска тестов
func runTest(t *testing.T, maxLineLen int, pkgs ...string) {
	t.Helper()

	plugin := &PluginLineWrap{
		settings: struct{ MaxLineLen int }{MaxLineLen: maxLineLen},
	}

	analyzers, err := plugin.BuildAnalyzers()
	require.NoError(t, err, "Failed to build analyzers")
	require.Len(t, analyzers, 1, "Expected exactly one analyzer")
	
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzers[0], pkgs...)
}