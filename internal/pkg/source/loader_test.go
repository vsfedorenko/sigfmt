package source_test

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vsfedorenko/sigfmt/internal/pkg/source"
)

const probeSrc = "package p\n\nfunc f() {\n\tx := 1\n}\n"

// countingReader records how many times each filename was read — the cache
// contract (one disk read per file per pass) is the reason Loader exists.
// override, when set, replaces the returned bytes (shorter than the
// FileSet-registered size) to exercise the range guards.
type countingReader struct {
	reads    map[string]int
	err      error
	override []byte
}

func (r *countingReader) read(name string) ([]byte, error) {
	r.reads[name]++
	if r.err != nil {
		return nil, r.err
	}
	if r.override != nil {
		return r.override, nil
	}
	return []byte(probeSrc), nil
}

type readError struct{}

func (readError) Error() string { return "boom" }

func newTestLoader(t *testing.T) (*source.Loader, *token.File, *countingReader, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	tf := fset.AddFile("/probe/a.go", -1, len(probeSrc))
	reader := &countingReader{reads: map[string]int{}}
	return source.NewLoader(fset, reader.read), tf, reader, fset
}

func TestLoaderLoadCachesPerFile(t *testing.T) {
	ld, tf, reader, fset := newTestLoader(t)

	first := ld.Load(tf.Pos(0))
	second := ld.Load(tf.Pos(len(probeSrc) - 1))

	// want: same *File back, exactly one read regardless of position count.
	assert.Same(t, first, second, "repeated Load must return the cached File")
	assert.Equal(t, 1, reader.reads["/probe/a.go"], "one disk read per file per Loader")
	assert.True(t, first.OK)
	assert.Equal(t, []byte(probeSrc), first.Content)
	assert.Equal(t, fset, first.Fset())
	assert.Equal(t, tf, first.TokenFile)
}

func TestLoaderLoadDistinctFiles(t *testing.T) {
	ld, tf, reader, fset := newTestLoader(t)
	tfB := fset.AddFile("/probe/b.go", -1, len(probeSrc))

	a := ld.Load(tf.Pos(0))
	b := ld.Load(tfB.Pos(0))

	assert.NotSame(t, a, b, "different files must not share a cache entry")
	assert.True(t, b.OK)
	assert.Equal(t, 1, reader.reads["/probe/a.go"])
	assert.Equal(t, 1, reader.reads["/probe/b.go"])
}

func TestLoaderLoadUnknownPos(t *testing.T) {
	ld, _, reader, _ := newTestLoader(t)

	got := ld.Load(token.Pos(1 << 20))

	// want: fail-open empty File, no disk access for an unresolvable position.
	assert.False(t, got.OK)
	assert.Nil(t, got.TokenFile)
	assert.Empty(t, got.Content)
	assert.Empty(t, reader.reads)
}

func TestLoaderLoadReadError(t *testing.T) {
	fset := token.NewFileSet()
	tf := fset.AddFile("/probe/err.go", -1, len(probeSrc))
	reader := &countingReader{reads: map[string]int{}, err: readError{}}
	ld := source.NewLoader(fset, reader.read)

	first := ld.Load(tf.Pos(0))
	second := ld.Load(tf.Pos(1))

	// want: failed loads are cached too — a broken file must not be re-read
	// on every position (the pass touches each file many times).
	assert.False(t, first.OK)
	assert.Same(t, first, second, "failed load must be cached, not retried")
	assert.Equal(t, 1, reader.reads["/probe/err.go"])
}

func TestFileIndent(t *testing.T) {
	ld, tf, _, _ := newTestLoader(t)
	// "package p\n\nfunc f() {\n\tx := 1\n}\n": the 'x' sits at offset 23,
	// its line starts with a single tab.
	xPos := tf.Pos(len("package p\n\nfunc f() {\n\t"))

	tests := []struct {
		name string
		file *source.File
		pos  token.Pos
		want string
	}{
		{"indented line at first code rune", ld.Load(tf.Pos(0)), xPos, "\t"},
		{"line start has no prefix yet", ld.Load(tf.Pos(0)), tf.Pos(0), ""},
		{"not-OK file", &source.File{}, token.Pos(1), ""},
		{"nil token file", &source.File{Content: []byte(" x")}, token.Pos(1), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.file.Indent(tt.pos))
		})
	}
}

func TestFileRange(t *testing.T) {
	ld, tf, _, _ := newTestLoader(t)
	f := ld.Load(tf.Pos(0))

	// A file whose on-disk content is SHORTER than the size registered in the
	// FileSet (stale cache, mid-edit read) makes the past-the-end guard fire:
	// token.File.Offset clamps positions to the registered size, so only the
	// content length can exceed.
	fsetShort := token.NewFileSet()
	tfShort := fsetShort.AddFile("/probe/short.go", -1, len(probeSrc))
	shortReader := &countingReader{reads: map[string]int{}}
	shortReader.override = []byte("package p\n") // 10 bytes < 35 registered
	short := source.NewLoader(fsetShort, shortReader.read).Load(tfShort.Pos(0))

	tests := []struct {
		name   string
		file   *source.File
		start  token.Pos
		end    token.Pos
		want   string
		wantOK bool
	}{
		{"valid range", f, tf.Pos(0), tf.Pos(8), "package ", true},
		{"full file", f, tf.Pos(0), tf.Pos(len(probeSrc)), probeSrc, true},
		{"inverted range", f, tf.Pos(8), tf.Pos(0), "", false},
		{"content shorter than registered size", short, tfShort.Pos(0), tfShort.Pos(len(probeSrc)), "", false},
		{"not-OK file", &source.File{}, token.Pos(1), token.Pos(2), "", false},
		{"nil token file", &source.File{Content: []byte("xy")}, token.Pos(1), token.Pos(2), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.file.Range(tt.start, tt.end)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewFile(t *testing.T) {
	fset := token.NewFileSet()
	tf := fset.AddFile("/probe/n.go", -1, len("x"))

	tests := []struct {
		name    string
		content []byte
		wantOK  bool
	}{
		{"non-nil content", []byte("x"), true},
		{"empty non-nil slice", []byte{}, true},
		{"nil content", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := source.NewFile(fset, tf, tt.content)
			assert.Equal(t, tt.wantOK, f.OK)
			assert.Equal(t, fset, f.Fset())
			assert.Equal(t, tf, f.TokenFile)
		})
	}
}
