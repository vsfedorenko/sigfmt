package linters

import (
    "testing"

    "github.com/stretchr/testify/require"
    "golang.org/x/tools/go/analysis/analysistest"
)

// TestLineWrap_Basic тестирует базовые случаи с настройками по умолчанию (120)
func TestLineWrap_Basic(t *testing.T) {
    plugin := &PluginLineWrap{
        settings: struct{ MaxLineLen int }{MaxLineLen: 120},
    }

    analyzers, err := plugin.BuildAnalyzers()
    require.NoError(t, err, "Failed to build analyzers")
    analyzer := analyzers[0]

    analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, "cases")
}

// TestLineWrap_Length80 тестирует с максимальной длиной 80 символов
func TestLineWrap_Length80(t *testing.T) {
    plugin := &PluginLineWrap{
        settings: struct{ MaxLineLen int }{MaxLineLen: 80},
    }

    analyzers, err := plugin.BuildAnalyzers()
    require.NoError(t, err, "Failed to build analyzers")
    analyzer := analyzers[0]

    analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, "length80")
}

// TestLineWrap_Length100 тестирует с максимальной длиной 100 символов
func TestLineWrap_Length100(t *testing.T) {
    plugin := &PluginLineWrap{
        settings: struct{ MaxLineLen int }{MaxLineLen: 100},
    }

    analyzers, err := plugin.BuildAnalyzers()
    require.NoError(t, err, "Failed to build analyzers")
    analyzer := analyzers[0]

    analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, "length100")
}

// TestLineWrap_Length120 тестирует с максимальной длиной 120 символов
func TestLineWrap_Length120(t *testing.T) {
    plugin := &PluginLineWrap{
        settings: struct{ MaxLineLen int }{MaxLineLen: 120},
    }

    analyzers, err := plugin.BuildAnalyzers()
    require.NoError(t, err, "Failed to build analyzers")
    analyzer := analyzers[0]

    analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, "length120")
}

// TestLineWrap_Length140 тестирует с максимальной длиной 140 символов
func TestLineWrap_Length140(t *testing.T) {
    plugin := &PluginLineWrap{
        settings: struct{ MaxLineLen int }{MaxLineLen: 140},
    }

    analyzers, err := plugin.BuildAnalyzers()
    require.NoError(t, err, "Failed to build analyzers")
    analyzer := analyzers[0]

    analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, "length140")
}

// TestLineWrap_Length160 тестирует с максимальной длиной 160 символов
func TestLineWrap_Length160(t *testing.T) {
    plugin := &PluginLineWrap{
        settings: struct{ MaxLineLen int }{MaxLineLen: 160},
    }

    analyzers, err := plugin.BuildAnalyzers()
    require.NoError(t, err, "Failed to build analyzers")
    analyzer := analyzers[0]

    analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, "length160")
}

// TestLineWrap_AllConfigurations запускает все тесты последовательно
func TestLineWrap_AllConfigurations(t *testing.T) {
    testCases := []struct {
        name      string
        maxLen    int
        pkg       string
    }{
        {"Basic_120", 120, "cases"},
        {"Length_80", 80, "length80"},
        {"Length_100", 100, "length100"},
        {"Length_120", 120, "length120"},
        {"Length_140", 140, "length140"},
        {"Length_160", 160, "length160"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            plugin := &PluginLineWrap{
                settings: struct{ MaxLineLen int }{MaxLineLen: tc.maxLen},
            }

            analyzers, err := plugin.BuildAnalyzers()
            require.NoError(t, err, "Failed to build analyzers")
            analyzer := analyzers[0]

            analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, tc.pkg)
        })
    }
}
