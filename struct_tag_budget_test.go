package sigfmt

import (
	"fmt"
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tagCorpus pins struct-tag awareness: a func-typed struct field carrying
// a tag (`json:"…"`) must keep the WHOLE construct — signature + tag —
// within the line limit, whether collapsed to one line or packed across
// several. The tag itself is never part of the rewrite, so it must also
// survive verbatim.
var tagCorpus = []string{
	// Short signature + short tag: collapses, tag keeps the line under the
	// limit trivially.
	"package g\n\ntype S struct {\n\tHandler func(\n\t\tw int,\n\t\tr string,\n\t) error `json:\"handler\"`\n}\n",

	// Short signature + LONG tag: without tag awareness the collapsed line
	// would exceed 120 (signature ~30 + tag ~95); it must stay split.
	"package g\n\ntype S struct {\n\tHandler func(\n\t\tw int,\n\t\tr string,\n\t) error `json:\"handler\" validate:\"required,min=1,max=64,oneof=start stop restart\" yaml:\"handler_key_for_serialization\"`\n}\n",

	// Many parameters + tag: packing must budget the tag (and results)
	// into the last line's width.
	"package g\n\ntype Registry struct {\n\tProcessOrderEntry func(\n\t\tfirstParameterName string,\n\t\tsecondParameterName int64,\n\t\tthirdParameterName int,\n\t\tfourthParameterName string,\n\t) (result string, err error) `json:\"process_order_entry\" validate:\"required\"`\n}\n",

	// Multi-name field + tag: every name and the tag must survive.
	"package g\n\ntype Multi struct {\n\tOnEvent, OnSignal func(\n\t\tid string,\n\t) error `json:\"on_event\"`\n}\n",

	// Already-shaped one-liner with a tag: fits — must be left alone
	// (idempotence on clean input).
	"package g\n\ntype Clean struct {\n\tSimple func(id string) error `json:\"simple\"`\n}\n",

	// Tagged field inside a struct whose neighbours also carry tags (the
	// aligned-table case): the rewrite must not fight the layout.
	"package g\n\ntype Table struct {\n\tAddr    string `json:\"addr\"`\n\tHandler func(\n\t\tw int,\n\t\tr string,\n\t) error `json:\"handler\"`\n\tTimeout int `json:\"timeout\"`\n}\n",
}

// TestStructTagLineBudget runs the public analyzer over the corpus and
// asserts, for every fix applied: (1) sigfmt never makes a line longer
// than the limit unless the input already overflowed at least as much
// (an unshrinkable tag is the input's problem, not the fix's), (2) the
// tag survives verbatim, (3) gofmt's only possible disagreement is
// whitespace (tag re-alignment), never code characters, (4) a second
// pass is a no-op.
func TestStructTagLineBudget(t *testing.T) {
	for i, src := range tagCorpus {
		t.Run(fmt.Sprintf("case%02d", i), func(t *testing.T) {
			fixed := applyAllFixes(t, src)

			// (1) line budget: no NEW overflow. When the input already
			// overflows (an unshrinkable tag), the fix must not make the
			// worst line any longer.
			inMax, outMax := 0, 0
			for _, line := range strings.Split(src, "\n") {
				inMax = max(inMax, visualLenForTest(line))
			}
			for _, line := range strings.Split(fixed, "\n") {
				outMax = max(outMax, visualLenForTest(line))
			}
			assert.LessOrEqualf(t, outMax, max(inMax, 120),
				"fix produced a %d-wide line (input max %d):\n%s", outMax, inMax, fixed)

			// (2) tag survival: count of backtick tags never drops.
			assert.GreaterOrEqualf(t, strings.Count(fixed, "`"), strings.Count(src, "`"),
				"tag lost in rewrite:\n--- src ---\n%s\n--- fixed ---\n%s", src, fixed)

			// (3) gofmt may only disagree on whitespace (it re-aligns
			// neighbouring tags after a collapse — expected, not a bug):
			// collapsing all runs of spaces/tabs inside lines must make
			// both outputs identical.
			formatted, err := format.Source([]byte(fixed))
			require.NoError(t, err, "gofmt rejects fixed output:\n--- fixed ---\n%s", fixed)
			assert.Equal(t, squashSpaces(string(formatted)), squashSpaces(fixed),
				"gofmt disagrees beyond whitespace alignment")

			// (4) idempotence.
			second := applyAllFixes(t, fixed)
			assert.Equal(t, fixed, second, "second pass is not a no-op")
		})
	}
}

// TestStructTagSecondPassAfterGofmt pins the full editor round-trip:
// sigfmt rewrites, then gofmt re-aligns the struct tags; sigfmt must not
// fight back on the second pass (no edit-pong with the formatter).
func TestStructTagSecondPassAfterGofmt(t *testing.T) {
	for i, src := range tagCorpus {
		t.Run(fmt.Sprintf("case%02d", i), func(t *testing.T) {
			fixed := applyAllFixes(t, src)
			formatted, err := format.Source([]byte(fixed))
			require.NoError(t, err)

			// sigfmt over gofmt's output must propose nothing new.
			second := applyAllFixes(t, string(formatted))
			assert.Equal(t, string(formatted), second, "sigfmt fights gofmt's tag alignment")
		})
	}
}

// visualLenForTest expands tabs the way the linter budgets lines (8-wide
// tab stops), matching internal/pkg/text.VisualLength semantics.
func visualLenForTest(s string) int {
	col := 0
	for _, r := range s {
		switch r {
		case '	':
			col += 8 - col%8
		default:
			col++
		}
	}
	return col
}

// squashSpaces collapses every run of spaces/tabs to a single space and
// trims line-leading whitespace, making two texts equal whenever they
// differ only in whitespace alignment.
func squashSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.Join(lines, "\n")
}
