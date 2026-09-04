package filesystem_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A folder inside the vault replaced by a link to somewhere else, while a write
// through it is on its way.
//
// A sync client owns the folder it syncs and may put a link where a folder was
// at any moment. A rule asked of the name before the write is a rule about a
// vault that has since changed; the write itself is what has to stay inside.
func TestAFolderSwappedForALinkTakesNothingOutOfTheVault(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	writer, err := filesystem.OpenForWriting(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	notes := filepath.Join(root, "notes")
	if err := os.Symlink(elsewhere, notes); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}
	if err := os.Remove(notes); err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	var swapping sync.WaitGroup
	swapping.Add(1)
	go func() {
		defer swapping.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = os.RemoveAll(notes)
			_ = os.Symlink(elsewhere, notes)
			_ = os.Remove(notes)
			_ = os.Mkdir(notes, 0o755)
		}
	}()

	ctx := t.Context()
	// Enough turns that the swap lands between a rule and the write it was read
	// for, many times over.
	for range 800 {
		_ = writer.MakeFolder(ctx, "notes/inside")
		_ = writer.Bring(ctx, "notes/landed.md", strings.NewReader("bytes"))
		_ = writer.Create(ctx, "notes/created.md", []byte("bytes"))
		_, _ = writer.Write(ctx, "notes/written.md", []byte("bytes"), domain.Fingerprint{})
		_ = writer.Move(ctx, "notes/landed.md", "notes/moved.md")
		_ = writer.Remove(ctx, "notes/moved.md")
	}
	close(stop)
	swapping.Wait()

	left, err := os.ReadDir(elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		names := make([]string, 0, len(left))
		for _, each := range left {
			names = append(names, each.Name())
		}
		t.Errorf("a write landed outside the vault: %v", names)
	}
}
