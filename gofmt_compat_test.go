package sigfmt

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	goldenGlob      = "*.golden"
	testdataDirName = "testdata"
	gofmtBinName    = "gofmt"
	goBinName       = "go"
)

// TestGoldenFilesAreGofmtSFixedPoints guards the gofmt -s compatibility
// contract: every golden file (the exact bytes sigfmt's suggested fixes
// produce) must be a `gofmt -s` fixed point. If `gofmt -s` would rewrite a
// golden file, sigfmt's auto-fix output diverges from gofmt -s, and a
// subsequent editor format-on-save would churn the very lines sigfmt just
// touched. The check shells out to the real gofmt binary shipped with the
// current toolchain, so the contract is verified against exactly the
// formatter users run, not a re-implementation.
func TestGoldenFilesAreGofmtSFixedPoints(t *testing.T) {
	gofmt := gofmtPath(t)

	goldenFiles, err := filepath.Glob(filepath.Join(testdataDirName, "src", "*", goldenGlob))
	require.NoError(t, err, "glob golden files")
	require.NotEmpty(t, goldenFiles, "no golden files found — testdata layout changed?")

	for _, golden := range goldenFiles {
		t.Run(filepath.Base(golden), func(t *testing.T) {
			src, err := os.ReadFile(golden)
			require.NoError(t, err, "read golden file")

			cmd := exec.Command(gofmt, "-s", golden)
			out, err := cmd.Output()
			require.NoError(t, err, "run gofmt -s")

			assert.Equal(t, string(src), string(out),
				"%s is not a gofmt -s fixed point: run `gofmt -s -w %s` and commit the result", golden, golden)
		})
	}
}

// gofmtPath locates the gofmt binary that belongs to the toolchain running
// the tests: first next to the `go` binary on PATH (they ship together),
// then gofmt on PATH itself as a fallback.
func gofmtPath(t *testing.T) string {
	t.Helper()

	name := gofmtBinName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	if goPath, err := exec.LookPath(goBinName); err == nil {
		if p := filepath.Join(filepath.Dir(goPath), name); fileExists(p) {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	t.Skipf("gofmt binary not found next to `go` or on PATH — cannot verify gofmt -s compatibility")
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
