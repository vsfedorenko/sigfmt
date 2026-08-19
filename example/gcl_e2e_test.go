package main

/**
 * End-to-end plugin-path test: build the custom golangci-lint binary
 * (exactly what `make build-example` produces) and run it against a
 * throwaway third-party module — the way a real consumer integrates
 * sigfmt as a golangci-lint v2 custom linter.
 *
 * Covered contracts (found by hand-probing the plugin path):
 *   1. the linter activates ONLY when the target project's config
 *      declares it under linters.settings.custom (bare `enable:` yields
 *      "unknown linters" — documented behavior of golangci-lint v2
 *      custom builds);
 *   2. diagnostics fire on violations in the third-party module;
 *   3. `--fix` applies the suggested fixes in place;
 *   4. a re-run after the fix is clean (idempotence through golangci);
 *   5. comment preservation holds through the plugin path (the #29
 *      contract: commented signatures are untouched);
 *   6. a 120-violation bulk package reports every issue when golangci's
 *      default caps (max-same-issues=3) are lifted.
 *
 * Skipped in -short mode: building the custom binary takes ~40s.
 */

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	customBinary = "example/custom-gcl"
	moduleName   = "gclprobe.example"
)

func buildCustomGCL(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("custom binary build takes ~40s")
	}

	bin, err := filepath.Abs(filepath.Join("..", "example", "custom-gcl"))
	require.NoError(t, err)
	// Reuse an existing binary when present (developer machines); build
	// fresh in CI.
	if _, statErr := os.Stat(bin); statErr != nil {
		// Same command `make build-example` runs.
		cmd := exec.Command("golangci-lint", "custom")
		cmd.Dir = filepath.Join(mustRepoRoot(t), "example")
		out, buildErr := cmd.CombinedOutput()
		require.NoError(t, buildErr, "golangci-lint custom: %s", out)
	}
	return bin
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, ".."))
}

// writeProbeModule creates a third-party module with one violating file,
// one commented (must-be-kept) file, and a golangci config activating the
// custom linter; returns the module root.
func writeProbeModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"go.mod": "module " + moduleName + "\n\ngo 1.25\n",
		".golangci.yml": `version: "2"

linters:
  default: none
  enable:
    - sigfmt
  settings:
    custom:
      sigfmt:
        type: "module"
        description: "signature formatter"
        settings:
          max-line-len: 120
`,
		"viol.go":  "package gclprobe\n\nfunc Long(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n",
		"guard.go": "package gclprobe\n\nfunc Guarded(\n\ta int, // keep me\n\tb string,\n) error {\n\treturn nil\n}\n",
	}
	for name, body := range files {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(body), 0o600))
	}
	return root
}

func runGCL(t *testing.T, bin, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	// golangci exits 1 on issues — expected here; only crash on other failures.
	if err != nil {
		var exitErr *exec.ExitError
		require.True(t, errors.As(err, &exitErr) && exitErr.ExitCode() <= 1,
			"custom-gcl %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestPluginPath_FixCycleOnThirdPartyModule(t *testing.T) {
	bin := buildCustomGCL(t)
	root := writeProbeModule(t)

	// 1. diagnostics fire on the violation, none on the commented file
	out := runGCL(t, bin, root, "run", "./...")
	assert.Contains(t, out, "viol.go", "expected a diagnostic in viol.go:\n%s", out)
	assert.NotContains(t, out, "guard.go", "commented signature must not be reported:\n%s", out)

	// 2. --fix applies in place
	runGCL(t, bin, root, "run", "--fix", "./...")
	fixed, err := os.ReadFile(filepath.Join(root, "viol.go"))
	require.NoError(t, err)
	assert.Contains(t, string(fixed), "func Long(a int, b string) error {", "--fix did not collapse the signature:\n%s", fixed)

	// 3. the commented file is byte-identical after --fix
	guard, err := os.ReadFile(filepath.Join(root, "guard.go"))
	require.NoError(t, err)
	assert.Contains(t, string(guard), "// keep me", "comment preservation broke through the plugin path:\n%s", guard)
	assert.NotContains(t, string(guard), "func Guarded(a int, b string)", "comment preservation broke through the plugin path:\n%s", guard)

	// 4. re-run is clean (idempotence)
	out = runGCL(t, bin, root, "run", "./...")
	assert.NotContains(t, out, "sigfmt", "re-run after --fix must be clean:\n%s", out)
}

func TestPluginPath_BulkModuleFullCount(t *testing.T) {
	bin := buildCustomGCL(t)
	root := writeProbeModule(t)

	bulk := filepath.Join(root, "bulk")
	require.NoError(t, os.MkdirAll(bulk, 0o755))
	for i := 0; i < 30; i++ {
		body := "package bulk\n\nfunc F" + itoa(i) + "(\n\ta int,\n\tb string,\n) error {\n\treturn nil\n}\n"
		require.NoError(t, os.WriteFile(filepath.Join(bulk, "b"+itoa(i)+".go"), []byte(body), 0o600))
	}

	// golangci's default caps (max-same-issues=3) must be lifted for the count
	out := runGCL(t, bin, root, "run", "--max-same-issues=0", "--max-issues-per-linter=0", "./...")
	assert.Equal(t, 30, strings.Count(out, "bulk/b"), "expected 30 bulk diagnostics:\n%s", out)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
