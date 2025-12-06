package linters

import (
	"bytes"
	"flag"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

const (
	goldenFileExt    = ".golden"
	goldenFilePerms  = 0644
	defaultLineLimit = 120
	settingMaxLineLen = "max-line-len"
	settingPackStructFields = "pack-struct-fields"
	settingPackInterfaceMethods = "pack-interface-methods"
)

var update = flag.Bool("update", false, "update golden files")

// fileEdit represents a single text replacement in a file
type fileEdit struct {
	pos     int    // start position
	end     int    // end position
	newText []byte // new text to replace with
}

// TestLineWrap_Features tests the linter with various Go language constructs
// (functions, interfaces, structs) using the default line length (120).
func TestLineWrap_Features(t *testing.T) {
	runTestWithSettings(t, map[string]interface{}{
		settingMaxLineLen: float64(defaultLineLimit),
	}, "features")
}

// TestLineWrap_Limits tests the linter with various line length limits.
func TestLineWrap_Limits(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		pkg   string
	}{
		{"Limit_80", 80, "limits/length80"},
		{"Limit_100", 100, "limits/length100"},
		{"Limit_120", 120, "limits/length120"},
		{"Limit_140", 140, "limits/length140"},
		{"Limit_160", 160, "limits/length160"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTestWithSettings(t, map[string]interface{}{
				settingMaxLineLen: float64(tt.limit),
			}, tt.pkg)
		})
	}
}

// TestLineWrap_Settings tests the linter with different configuration settings.
func TestLineWrap_Settings(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]interface{}
		pkg      string
	}{
		{
			name:     "Defaults",
			settings: map[string]interface{}{settingMaxLineLen: float64(defaultLineLimit)},
			pkg:      "settings/defaults",
		},
		{
			name: "NoPack",
			settings: map[string]interface{}{
				settingPackStructFields:     false,
				settingPackInterfaceMethods: false,
			},
			pkg: "settings/no_pack",
		},
		{
			name: "PackStructsOnly",
			settings: map[string]interface{}{
				settingPackStructFields:     true,
				settingPackInterfaceMethods: false,
			},
			pkg: "settings/pack_structs",
		},
		{
			name: "PackInterfacesOnly",
			settings: map[string]interface{}{
				settingPackStructFields:     false,
				settingPackInterfaceMethods: true,
			},
			pkg: "settings/pack_interfaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTestWithSettings(t, tt.settings, tt.pkg)
		})
	}
}

// runTestWithSettings is a helper function to run tests with specific settings.
func runTestWithSettings(t *testing.T, settings map[string]interface{}, pkgs ...string) {
	t.Helper()

	plugin, err := New(settings)
	require.NoError(t, err, "Failed to create plugin")

	analyzers, err := plugin.BuildAnalyzers()
	require.NoError(t, err, "Failed to build analyzers")
	require.Len(t, analyzers, 1, "Expected exactly one analyzer")

	testData := analysistest.TestData()
	if *update {
		updateGoldenFiles(t, analyzers[0], testData, pkgs...)
		return
	}

	analysistest.RunWithSuggestedFixes(t, testData, analyzers[0], pkgs...)
}

// updateGoldenFiles updates the golden files with suggested fixes from the analyzer.
func updateGoldenFiles(t *testing.T, a *analysis.Analyzer, dir string, pkgs ...string) {
	t.Helper()

	results := analysistest.Run(t, dir, a, pkgs...)
	fileEdits := collectEditsFromResults(results)

	for path, edits := range fileEdits {
		applyEditsToFile(t, path, edits)
	}
}

// collectEditsFromResults groups all text edits by file path.
func collectEditsFromResults(results []*analysistest.Result) map[string][]fileEdit {
	fileEdits := make(map[string][]fileEdit)

	for _, res := range results {
		for _, diag := range res.Diagnostics {
			for _, fix := range diag.SuggestedFixes {
				for _, edit := range fix.TextEdits {
					file := res.Pass.Fset.File(edit.Pos)
					path := file.Name()

					fileEdits[path] = append(fileEdits[path], fileEdit{
						pos:     file.Offset(edit.Pos),
						end:     file.Offset(edit.End),
						newText: edit.NewText,
					})
				}
			}
		}
	}

	return fileEdits
}

// applyEditsToFile applies a list of edits to a file and writes the golden file.
func applyEditsToFile(t *testing.T, path string, edits []fileEdit) {
	t.Helper()

	goldenPath := path + goldenFileExt

	src, err := os.ReadFile(path)
	if err != nil {
		t.Logf("Error reading %s: %v", path, err)
		return
	}

	// Sort edits in reverse order (from end of file) to avoid offset disruption
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].pos > edits[j].pos
	})

	// Apply edits sequentially
	result := src
	for _, edit := range edits {
		result = applyEdit(result, edit)
	}

	if err := os.WriteFile(goldenPath, result, goldenFilePerms); err != nil {
		t.Logf("Error writing %s: %v", goldenPath, err)
		return
	}

	t.Logf("Updated: %s", goldenPath)
}

// applyEdit applies a single edit to the source bytes.
func applyEdit(src []byte, edit fileEdit) []byte {
	var buf bytes.Buffer
	buf.Write(src[:edit.pos])
	buf.Write(edit.newText)
	buf.Write(src[edit.end:])
	return buf.Bytes()
}
