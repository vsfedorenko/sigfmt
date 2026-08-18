package sigfmt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// benchCorpusSeed namespaces the corpus directories so parallel test/bench
// runs never share temp state.
const benchCorpusSeed = 1

// benchKinds are the signature categories the corpus cycles through. They
// mirror the shapes sigfmt handles in real code: collapsible short
// multi-line signatures, long signatures that must stay multi-line but can be
// packed, interface method sets, and struct fields with function types.
var benchKinds = []string{"collapse", "pack", "iface", "structfield"}

// finalizeCorpusFile normalizes the trailing newlines of a generated corpus
// file to gofmt's convention (exactly one newline at EOF), so the corpus
// itself is a gofmt fixed point.
func finalizeCorpusFile(sb *strings.Builder) string {
	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// writeBenchCorpus generates the violation corpus: every file contains
// benchDeclsPerFile signatures that need formatting, cycling through
// benchKinds. Output is a pure function of the constants — no randomness.
func writeBenchCorpus(dir string) error {
	for i := 0; i < benchFilesPerCategory; i++ {
		var sb strings.Builder
		sb.WriteString("package corpus\n\n")
		for j := 0; j < benchDeclsPerFile; j++ {
			kind := benchKinds[(i*benchDeclsPerFile+j)%len(benchKinds)]
			switch kind {
			case "collapse":
				fmt.Fprintf(&sb, "func Collapsible%d%d(\n\ta int,\n\tb string,\n\tc bool,\n) {\n\t_ = a\n\t_ = b\n\t_ = c\n}\n\n", i, j)
			case "pack":
				fmt.Fprintf(&sb, "func Packed%d%d(ctx context.Context, request *Request, options map[string]string, timeout time.Duration, retries int, logger *Logger) (*Response, error) {\n\treturn nil, nil\n}\n\n", i, j)
			case "iface":
				fmt.Fprintf(&sb, "type Service%d%d interface {\n\tDoSomething(\n\t\tctx context.Context,\n\t\tid string,\n\t) error\n}\n\n", i, j)
			case "structfield":
				fmt.Fprintf(&sb, "type Handler%d%d struct {\n\tProcess func(\n\t\tctx context.Context,\n\t\tevent string,\n\t) error\n}\n\n", i, j)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("gen_%02d.go", i)), []byte(finalizeCorpusFile(&sb)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// writeBenchCorpusClean generates the clean corpus: already-compact
// signatures the analyzer must scan and leave alone (zero diagnostics). This
// is the incremental-rerun profile of a mostly-formatted codebase.
func writeBenchCorpusClean(dir string) error {
	for i := 0; i < benchFilesPerCategory; i++ {
		var sb strings.Builder
		sb.WriteString("package corpus\n\n")
		for j := 0; j < benchDeclsPerFile; j++ {
			fmt.Fprintf(&sb, "func Compact%d%d(a int, b string, c bool) error { return nil }\n\n", i, j)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("clean_%02d.go", i)), []byte(finalizeCorpusFile(&sb)), 0o644); err != nil {
			return err
		}
	}
	return nil
}
