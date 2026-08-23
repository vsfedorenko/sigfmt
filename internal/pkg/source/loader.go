package source

import (
	"go/token"
	"sync"
)

// File is a parsed file's identity: its token file, its raw content, and
// whether the content could be loaded at all. The analyzer walk loads a
// File once per *ast.File and hands it to every consumer that needs source
// bytes — indentation lookups and range reads share the same copy instead
// of hitting the disk per question.
type File struct {
	fset      *token.FileSet
	TokenFile *token.File
	Content   []byte
	OK        bool // false when the content could not be loaded
}

// Fset returns the FileSet the file's positions resolve into.
func (f *File) Fset() *token.FileSet { return f.fset }

// NewFile builds a File directly from its parts. Production code goes
// through Loader, which owns caching; NewFile serves tests that wire
// synthetic FileSets and one-off inspection of already-loaded bytes.
func NewFile(fset *token.FileSet, tf *token.File, content []byte) *File {
	return &File{fset: fset, TokenFile: tf, Content: content, OK: content != nil}
}

// Indent returns the leading whitespace of the physical line containing pos.
// See Indent for the line-table caveats //line directives impose.
func (f *File) Indent(pos token.Pos) string {
	if !f.OK || f.TokenFile == nil {
		return ""
	}
	offset := f.TokenFile.Offset(pos)
	if offset < 0 || offset >= len(f.Content) {
		return ""
	}
	return indentFromOffset(f.Content, offset)
}

// Range returns the source text between start and end offsets. ok is false
// when the range falls outside the content (the caller keeps its
// fail-open behavior).
func (f *File) Range(start, end token.Pos) (string, bool) {
	if !f.OK || f.TokenFile == nil {
		return "", false
	}
	so := f.TokenFile.Offset(start)
	eo := f.TokenFile.Offset(end)
	if eo < so || so < 0 || eo > len(f.Content) {
		return "", false
	}
	return string(f.Content[so:eo]), true
}

// Loader loads the source File for a position's filename, caching per
// filename: one analyzer pass touches the same file's bytes many times
// (indent lookups in strategies, the no-op guard, the comment guard), and
// re-reading from disk each time dominated the analyzer's profile (50% of
// CPU in syscalls, 78% of allocated bytes). The zero Loader reads from
// disk directly; NewLoader wires it over an analysis.Pass.ReadFile.
type Loader struct {
	readFile func(filename string) ([]byte, error)
	fset     *token.FileSet
	mu       sync.Mutex
	files    map[string]*File
}

// NewLoader returns a Loader resolving positions through fset and reading
// bytes through readFile (analysis.Pass.ReadFile in production, an
// in-memory map in tests).
func NewLoader(fset *token.FileSet, readFile func(filename string) ([]byte, error)) *Loader {
	return &Loader{readFile: readFile, fset: fset, files: map[string]*File{}}
}

// Load returns the File for pos, loading and caching it on first access.
// A file that cannot be read yields File{OK:false} — callers fail open.
func (l *Loader) Load(pos token.Pos) *File {
	tf := l.fset.File(pos)
	if tf == nil {
		return &File{}
	}
	name := tf.Name()

	l.mu.Lock()
	defer l.mu.Unlock()
	if cached, hit := l.files[name]; hit {
		return cached
	}

	loaded := &File{fset: l.fset, TokenFile: tf}
	content, err := l.readFile(name)
	if err == nil {
		loaded.Content = content
		loaded.OK = true
	}
	l.files[name] = loaded
	return loaded
}
