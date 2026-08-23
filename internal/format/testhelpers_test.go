package format

import (
	"fmt"
	"go/token"

	"github.com/vsfedorenko/sigfmt/internal/pkg/source"
)

// newSyntheticFile wraps a synthetic FileSet (AddFile, no real bytes) into
// a source.File for strategy and builder tests: positions resolve, indent
// lookups yield empty (no content), the builder falls back to tabs.
func newSyntheticFile(fset *token.FileSet) *source.File {
	var tf *token.File
	fset.Iterate(func(f *token.File) bool {
		if tf == nil {
			tf = f
		}
		return true
	})
	return source.NewFile(fset, tf, []byte("package p\n"))
}

// testLoader wires a Loader over an in-memory source for Check tests.
func testLoader(fset *token.FileSet, src string) *source.Loader {
	return source.NewLoader(fset, func(string) ([]byte, error) { return []byte(src), nil })
}

// testLoaderFailing simulates an unreadable file: guards must fail open.
func testLoaderFailing(fset *token.FileSet) *source.Loader {
	return source.NewLoader(fset, func(string) ([]byte, error) { return nil, errFakeRead })
}

var errFakeRead = fmt.Errorf("fake read failure")
