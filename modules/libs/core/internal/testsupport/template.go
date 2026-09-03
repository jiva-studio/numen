package testsupport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Template is a database built once for a whole test binary and handed out as
// copies. Each copy is a file of its own, so tests that run in parallel share
// nothing.
type Template struct {
	build func(ctx context.Context, path string) error
	once  sync.Once
	body  []byte
	err   error
}

// NewTemplate takes the way the database is built. The build runs when the
// first copy is asked for.
func NewTemplate(build func(ctx context.Context, path string) error) *Template {
	return &Template{build: build}
}

// Path is a copy of the template under the test's own temporary folder.
func (t *Template) Path(tb testing.TB) string {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "index.db")
	t.CopyTo(tb, path)
	return path
}

// CopyTo writes a copy of the template at the path given.
func (t *Template) CopyTo(tb testing.TB, path string) {
	tb.Helper()
	t.once.Do(t.make)
	if t.err != nil {
		tb.Fatal(t.err)
	}
	if err := os.WriteFile(path, t.body, 0o600); err != nil {
		tb.Fatal(err)
	}
}

// make builds the database once and keeps the file in memory. A database closed
// cleanly has checkpointed, so the one file is the whole of it; the sidecars are
// checked rather than trusted.
func (t *Template) make() {
	dir, err := os.MkdirTemp("", "numen-template")
	if err != nil {
		t.err = err
		return
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "template.db")
	if err := t.build(context.Background(), path); err != nil {
		t.err = err
		return
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		info, err := os.Stat(path + suffix)
		if err == nil && info.Size() > 0 {
			t.err = fmt.Errorf("the template still holds %d bytes in %s", info.Size(), suffix)
			return
		}
	}
	t.body, t.err = os.ReadFile(path)
}
