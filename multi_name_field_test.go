package sigfmt

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// multiNameFieldCorpus pins the destructive-fix class found by hand-probing
// the released CLI: a func-typed struct field declared with SEVERAL names
// (`Handler, Fallback func(...)`) used to lose every name but the first when
// sigfmt rewrote it — the fix silently deleted struct fields. Every entry
// here must survive a full fix cycle with all names intact.
var multiNameFieldCorpus = []struct {
	name string
	src  string
}{
	{
		// Collapse path: the multi-line signature fits on one line.
		name: "Collapse",
		src: "package p\n\n" +
			"type T struct {\n" +
			"\tHandler, Fallback func(\n" +
			"\t\tw string,\n" +
			"\t\tr string,\n" +
			"\t) error\n" +
			"}\n",
	},
	{
		// Packing path: too long for one line even after packing.
		name: "Packing",
		src: "package p\n\n" +
			"type T struct {\n" +
			"\tPrimary, Secondary func(\n" +
			"\t\trequestIdentifier string,\n" +
			"\t\tuserIdentifier string,\n" +
			"\t\ttraceIdentifier string,\n" +
			"\t\tspanIdentifier string,\n" +
			"\t\textraIdentifier string,\n" +
			"\t\tmoreIdentifier string,\n" +
			"\t) error\n" +
			"}\n",
	},
	{
		// Tag after the results list: the tag is outside the rewritten span
		// and must survive untouched next to every name.
		name: "WithTag",
		src: "package p\n\n" +
			"type T struct {\n" +
			"\tHandler, Fallback func(\n" +
			"\t\tw string,\n" +
			"\t\tr string,\n" +
			"\t) error `json:\"hf\"`\n" +
			"}\n",
	},
	{
		// Three names, one of which would fit alone — all must survive.
		name: "ThreeNames",
		src: "package p\n\n" +
			"type T struct {\n" +
			"\tA, B, C func(\n" +
			"\t\tx int,\n" +
			"\t) error\n" +
			"}\n",
	},
}

// TestMultiNameStructFieldFixKeepsAllNames: applying the suggested fixes to
// a multi-name func field must keep every declared field name. The fixed
// output must parse and a second pass must be a no-op (idempotence).
func TestMultiNameStructFieldFixKeepsAllNames(t *testing.T) {
	for _, tc := range multiNameFieldCorpus {
		t.Run(tc.name, func(t *testing.T) {
			fixed := applyAllFixes(t, tc.src)

			if _, err := parser.ParseFile(token.NewFileSet(), "fixed.go", fixed, parser.ParseComments); err != nil {
				t.Fatalf("fixed output does not parse: %v\n%s", err, fixed)
			}

			// Every declared name must still be present in the fixed text.
			// Names are declared in the corpus entries themselves; extract
			// them by requiring the exact joined form the fix must render.
			wantNames := multiNameWant(t, tc.src)
			for _, name := range wantNames {
				if !strings.Contains(fixed, name) {
					t.Errorf("field name %q was dropped by the fix:\noriginal:\n%s\nfixed:\n%s", name, tc.src, fixed)
				}
			}

			// Idempotence: a second pass over the fixed text must not
			// change it further.
			again := applyAllFixes(t, fixed)
			if again != fixed {
				t.Errorf("fix is not idempotent:\nfirst:\n%s\nsecond:\n%s", fixed, again)
			}
		})
	}
}

// multiNameWant returns every identifier of the multi-name func field
// declared in the fixture (the names on the line that also has "func(").
func multiNameWant(t *testing.T, src string) []string {
	t.Helper()
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "func(") {
			continue
		}
		head := trimmed[:strings.Index(trimmed, " func(")]
		if strings.Contains(head, ",") {
			parts := strings.Split(head, ",")
			names := make([]string, 0, len(parts))
			for _, p := range parts {
				names = append(names, strings.TrimSpace(p))
			}
			return names
		}
		return strings.Fields(head)
	}
	t.Fatalf("fixture declares no func-typed field:\n%s", src)
	return nil
}
