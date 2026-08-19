package main

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

// CLI smoke tests exercise the released binary surface: exit codes,
// -fix, -diff, and flag forwarding. They build the real binary and run
// it against fixture packages, mirroring what a consumer does.

// binDir is inside the repo tree: t.TempDir() can be mounted noexec,
// which makes the freshly built binary unrunnable.
const binDir = "../../.testbin"

// buildCLI builds the sigfmt binary once for the whole test run.
func buildCLI(t *testing.T) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	t.Cleanup(func() { _ = os.RemoveAll(binDir) })
	bin := filepath.Join(binDir, "sigfmt-test")
	abs, err := filepath.Abs(bin)
	require.NoError(t, err)
	build := exec.Command("go", "build", "-o", abs, ".")
	wd, err := os.Getwd()
	require.NoError(t, err)
	build.Dir = wd
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build CLI: %s", out)
	return abs
}

// writePkg creates a Go package with the given source files under a
// module root and returns the root path.
func writePkg(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module smoke.test\n\ngo 1.25\n"), 0o600))
	for name, body := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	}
	return root
}

func runCLI(t *testing.T, bin, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else {
		require.NoError(t, err, "run CLI: %s", out)
	}
	return string(out), code
}

const srcViolations = `package smoke

func Long(
	a int,
	b string,
) error {
	return nil
}
`

const srcClean = `package smoke

func Fine(a int, b string) error { return nil }
`

const srcCommented = `package smoke

func Guarded(
	a int, // keep me
	b string,
) error {
	return nil
}
`

func TestCLI_CleanPackageExitsZero(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"clean/clean.go": srcClean})

	out, code := runCLI(t, bin, root, "./clean")
	require.Zero(t, code, "clean package must exit 0:\n%s", out)
	assert.NotContains(t, strings.ToLower(out), "signature", "clean package must produce no diagnostics:\n%s", out)
}

func TestCLI_ViolationsExitNonZero(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"viol/viol.go": srcViolations})

	out, code := runCLI(t, bin, root, "./viol")
	require.NotZero(t, code, "violations must exit non-zero:\n%s", out)
	assert.Contains(t, out, "Signature can be formatted more compactly", "expected the diagnostic in output:\n%s", out)
}

func TestCLI_DiffDoesNotModifyFiles(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"viol/viol.go": srcViolations})
	path := filepath.Join(root, "viol", "viol.go")

	out, code := runCLI(t, bin, root, "-fix", "-diff", "./viol")
	// Contract (documented in the editor-integration recipes): -diff
	// prints the proposed changes without applying them and exits 0 —
	// it is a preview mode, not a check.
	require.Zero(t, code, "-diff must exit 0 (preview mode):\n%s", out)

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, srcViolations, string(after), "-diff must not modify the file:\n%s", after)
	assert.Contains(t, out, "func Long(a int, b string) error {", "-diff must print the proposed fix:\n%s", out)
}

func TestCLI_FixRewritesFile(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"viol/viol.go": srcViolations})
	path := filepath.Join(root, "viol", "viol.go")

	out, code := runCLI(t, bin, root, "-fix", "./viol")
	// singlechecker exits 0 after applying fixes when nothing remains.
	if code != 0 && !strings.Contains(out, "fixed") {
		// exit 3/1 acceptable only if it reports remaining diagnostics;
		// after a successful fix the re-run must be clean either way.
		t.Logf("fix run exited %d:\n%s", code, out)
	}

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(after), "func Long(a int, b string) error {", "-fix must collapse the signature in place:\n%s", after)

	// Re-run on the fixed file: clean, exit 0.
	_, code = runCLI(t, bin, root, "./viol")
	require.Zero(t, code, "fixed package must pass the re-check")
}

func TestCLI_CommentedSignatureLeftAlone(t *testing.T) {
	// Comment preservation (#29): the CLI must not report or rewrite a
	// signature whose range contains comments.
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"guard/guard.go": srcCommented})
	path := filepath.Join(root, "guard", "guard.go")

	out, code := runCLI(t, bin, root, "./guard")
	require.Zero(t, code, "commented signature must not be a violation:\n%s", out)

	_, code = runCLI(t, bin, root, "-fix", "./guard")
	require.Zero(t, code, "-fix on commented signature must be a no-op")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, srcCommented, string(after), "file with commented signature must stay byte-identical:\n%s", after)
}

func TestCLI_UnknownFlagFails(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"clean/clean.go": srcClean})

	out, code := runCLI(t, bin, root, "--totally-bogus-flag")
	require.NotZero(t, code, "unknown flag must exit non-zero:\n%s", out)
	assert.Contains(t, out, "flag provided but not defined", "expected flag error, got:\n%s", out)
}

func TestCLI_FlagForwarding(t *testing.T) {
	// -max-line-len must reach the analyzer through singlechecker: with
	// limit 20 the collapsed form (30 chars) no longer fits, the collapse
	// strategy stops applying, and the violation disappears — proving the
	// flag changed analyzer behavior end-to-end.
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"viol/viol.go": srcViolations})

	out20, code20 := runCLI(t, bin, root, "-max-line-len", "20", "./viol")
	require.Zero(t, code20, "tight limit must disable the collapse diagnostic:\n%s", out20)
	assert.NotContains(t, out20, "Signature", "tight limit must disable the collapse diagnostic:\n%s", out20)

	outDefault, codeDefault := runCLI(t, bin, root, "./viol")
	require.NotZero(t, codeDefault, "default limit must report the violation:\n%s", outDefault)
	assert.Contains(t, outDefault, "Signature can be formatted more compactly", "default limit must report the violation:\n%s", outDefault)
}
