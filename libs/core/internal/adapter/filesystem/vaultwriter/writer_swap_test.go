package vaultwriter_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A folder inside the vault replaced by a link to somewhere else, while a write
// through it is on its way.
//
// A sync client owns the folder it syncs and may put a link where a folder was
// at any moment. A rule asked of the name before the write is a rule about a
// vault that has since changed; the write itself is what has to stay inside.
func TestAFolderSwappedForALinkConsumesNothingOutOfTheVault(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	writer, err := vaultwriter.OpenForWriting(root, vaultconfig.Options{})
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
	var swapFailures int
	go func() {
		defer swapping.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			for _, err := range []error{
				os.RemoveAll(notes),
				os.Symlink(elsewhere, notes),
				os.Remove(notes),
				os.Mkdir(notes, 0o755),
			} {
				if err != nil {
					swapFailures++
				}
			}
		}
	}()

	ctx := t.Context()
	var writeFailures int
	// Enough turns that the swap lands between a rule and the write it was read
	// for, many times over.
	for range 800 {
		_, writeErr := writer.Write(ctx, "notes/written.md", []byte("bytes"), domain.FileStat{})
		for _, err := range []error{
			writer.MakeFolder(ctx, "notes/inside"),
			writer.Import(ctx, "notes/landed.md", strings.NewReader("bytes")),
			writer.Create(ctx, "notes/created.md", []byte("bytes")),
			writeErr,
			writer.Move(ctx, "notes/landed.md", "notes/moved.md"),
			writer.Remove(ctx, "notes/moved.md"),
		} {
			if err != nil {
				writeFailures++
			}
		}
	}
	close(stop)
	swapping.Wait()
	t.Logf("%d writes failed and %d swap steps failed while the folder was swapped", writeFailures, swapFailures)

	remaining, err := os.ReadDir(elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		names := make([]string, 0, len(remaining))
		for _, each := range remaining {
			names = append(names, each.Name())
		}
		t.Errorf("a write landed outside the vault: %v", names)
	}
}
