package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(binDir) })
	bin := filepath.Join(binDir, "sigfmt-test")
	abs, err := filepath.Abs(bin)
	if err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", abs, ".")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	build.Dir = wd
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	return abs
}

// writePkg creates a Go package with the given source files under a
// module root and returns the root path.
func writePkg(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module smoke.test\n\ngo 1.25\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runCLI(t *testing.T, bin, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run CLI: %v\n%s", err, out)
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
	if code != 0 {
		t.Fatalf("clean package must exit 0, got %d\n%s", code, out)
	}
	if strings.Contains(strings.ToLower(out), "signature") {
		t.Fatalf("clean package must produce no diagnostics:\n%s", out)
	}
}

func TestCLI_ViolationsExitNonZero(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"viol/viol.go": srcViolations})

	out, code := runCLI(t, bin, root, "./viol")
	if code == 0 {
		t.Fatalf("violations must exit non-zero:\n%s", out)
	}
	if !strings.Contains(out, "Signature can be formatted more compactly") {
		t.Fatalf("expected the diagnostic in output:\n%s", out)
	}
}

func TestCLI_DiffDoesNotModifyFiles(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"viol/viol.go": srcViolations})
	path := filepath.Join(root, "viol", "viol.go")

	out, code := runCLI(t, bin, root, "-fix", "-diff", "./viol")
	// Contract (documented in the editor-integration recipes): -diff
	// prints the proposed changes without applying them and exits 0 —
	// it is a preview mode, not a check.
	if code != 0 {
		t.Fatalf("-diff must exit 0 (preview mode), got %d\n%s", code, out)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != srcViolations {
		t.Fatalf("-diff must not modify the file:\n%s", after)
	}
	if !strings.Contains(out, "func Long(a int, b string) error {") {
		t.Fatalf("-diff must print the proposed fix:\n%s", out)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "func Long(a int, b string) error {") {
		t.Fatalf("-fix must collapse the signature in place:\n%s", after)
	}

	// Re-run on the fixed file: clean, exit 0.
	if _, code := runCLI(t, bin, root, "./viol"); code != 0 {
		t.Fatalf("fixed package must pass the re-check")
	}
}

func TestCLI_CommentedSignatureLeftAlone(t *testing.T) {
	// Comment preservation (#29): the CLI must not report or rewrite a
	// signature whose range contains comments.
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"guard/guard.go": srcCommented})
	path := filepath.Join(root, "guard", "guard.go")

	out, code := runCLI(t, bin, root, "./guard")
	if code != 0 {
		t.Fatalf("commented signature must not be a violation:\n%s", out)
	}

	if _, code := runCLI(t, bin, root, "-fix", "./guard"); code != 0 {
		t.Fatalf("-fix on commented signature must be a no-op")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != srcCommented {
		t.Fatalf("file with commented signature must stay byte-identical:\n%s", after)
	}
}

func TestCLI_UnknownFlagFails(t *testing.T) {
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"clean/clean.go": srcClean})

	out, code := runCLI(t, bin, root, "--totally-bogus-flag")
	if code == 0 {
		t.Fatalf("unknown flag must exit non-zero:\n%s", out)
	}
	if !strings.Contains(out, "flag provided but not defined") {
		t.Fatalf("expected flag error, got:\n%s", out)
	}
}

func TestCLI_FlagForwarding(t *testing.T) {
	// -max-line-len must reach the analyzer through singlechecker: with
	// limit 20 the collapsed form (30 chars) no longer fits, the collapse
	// strategy stops applying, and the violation disappears — proving the
	// flag changed analyzer behavior end-to-end.
	bin := buildCLI(t)
	root := writePkg(t, map[string]string{"viol/viol.go": srcViolations})

	out20, code20 := runCLI(t, bin, root, "-max-line-len", "20", "./viol")
	if code20 != 0 || strings.Contains(out20, "Signature") {
		t.Fatalf("tight limit must disable the collapse diagnostic, got exit %d:\n%s", code20, out20)
	}

	outDefault, codeDefault := runCLI(t, bin, root, "./viol")
	if codeDefault == 0 || !strings.Contains(outDefault, "Signature can be formatted more compactly") {
		t.Fatalf("default limit must report the violation, got exit %d:\n%s", codeDefault, outDefault)
	}
}
