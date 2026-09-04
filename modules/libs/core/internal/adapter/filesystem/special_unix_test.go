//go:build unix

package filesystem_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// fifo is a named pipe in a vault, with the extension of a note. Nothing is at
// the other end of it.
func fifo(t *testing.T, root, name string) {
	t.Helper()
	if err := syscall.Mkfifo(filepath.Join(root, name), 0o600); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
}

// TestSpecialFileIsNotANote. A FIFO named like a note is not one: a walk passes
// over it, and reading it says so instead of waiting for a writer that will
// never come.
func TestSpecialFileIsNotANote(t *testing.T) {
	root := t.TempDir()
	fifo(t, root, "pipe.md")
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# note\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	src, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		var walked []string
		err := src.Walk(t.Context(), func(r domain.Fingerprint) error {
			walked = append(walked, r.Path)
			return nil
		})
		if err == nil && !slices.Equal(walked, []string{"note.md"}) {
			t.Errorf("walk found %v, want [note.md]", walked)
		}
		if _, err := src.Stat(t.Context(), "pipe.md"); !errors.Is(err, port.ErrNotANote) {
			t.Errorf("stat pipe.md: %v, want ErrNotANote", err)
		}
		if _, err := src.Read(t.Context(), "pipe.md"); !errors.Is(err, port.ErrNotANote) {
			t.Errorf("read pipe.md: %v, want ErrNotANote", err)
		}
		if f, err := src.Open(t.Context(), "pipe.md"); !errors.Is(err, port.ErrNotANote) {
			if err == nil {
				f.Close()
			}
			t.Errorf("open pipe.md: %v, want ErrNotANote", err)
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the vault blocked on a FIFO")
	}
}

// TestLinkToSpecialFileIsNotANote. A link inside the vault is followed, so a
// note kept as a link is read and a link to a FIFO is not.
func TestLinkToSpecialFileIsNotANote(t *testing.T) {
	root := t.TempDir()
	fifo(t, root, "pipe")
	if err := os.WriteFile(filepath.Join(root, "real.md"), []byte("# note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "pipe"), filepath.Join(root, "piped.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real.md"), filepath.Join(root, "linked.md")); err != nil {
		t.Fatal(err)
	}

	got := walkPaths(t, root)
	want := []string{"linked.md", "real.md"}
	if !slices.Equal(got, want) {
		t.Errorf("walk found %v, want %v", got, want)
	}
}

// TestALinkLeadingNowhereCostsTheWalkNothing. A link with nothing at the end of
// it and one that leads round in a circle are files a walk passes over, and the
// rest of the vault is reported.
func TestALinkLeadingNowhereCostsTheWalkNothing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.md"), []byte("# note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "gone.md"), filepath.Join(root, "dangling.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "second.md"), filepath.Join(root, "first.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "first.md"), filepath.Join(root, "second.md")); err != nil {
		t.Fatal(err)
	}

	got := walkPaths(t, root)
	want := []string{"real.md"}
	if !slices.Equal(got, want) {
		t.Errorf("walk found %v, want %v", got, want)
	}
}
