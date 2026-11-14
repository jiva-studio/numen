package file_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"

	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
)

func newListHandler(open window, readers vaultport.VaultReaders) *filewindow.Handler {
	fileKinds := file.NewGetFileKinds(readers, nil)
	choose := tabkind.NewChoose(tabKinds{}, notesOnly{}, nil)
	return filewindow.NewHandler(
		filewindow.NewFolders(
			open,
			file.NewListFolder(file.NewGetFolderEntries(readers, nil), file.NewListTabKinds(fileKinds, choose)),
			file.NewListFileKinds(fileKinds, choose), nil,
		),
		filewindow.NewChanges(open, open, nil, nil, nil),
	)
}

// unstamped lists what the real reader lists, with no modification time on
// any entry, as a source that is not a disk does.
type unstamped struct{ vaultreader.Readers }

func (u unstamped) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := u.Readers.Open(v)
	if err != nil {
		return nil, err
	}
	return unstampedReader{reader}, nil
}

type unstampedReader struct{ vaultport.VaultReader }

func (r unstampedReader) List(ctx context.Context, folder string) ([]domain.Entry, error) {
	entries, err := r.VaultReader.List(ctx, folder)
	for i := range entries {
		entries[i].ModTime = time.Time{}
	}
	return entries, err
}

func TestAListingAnswersWhenEachEntryWasModifiedInMilliseconds(t *testing.T) {
	v := vaultfixture.NewVault(t, map[string]string{
		"links/tide.md":     "# Tide\n",
		"links/deeper/a.md": "# A\n",
	})
	at := time.Date(2025, 11, 2, 10, 30, 0, 5_000_000, time.UTC)
	folderAt := time.Date(2024, 1, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(v.Path, "links", "tide.md"), at, at); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(v.Path, "links", "deeper"), folderAt, folderAt); err != nil {
		t.Fatal(err)
	}
	handler := newListHandler(window{vault: v}, vaultreader.Readers{})

	answer, err := handler.ListFiles(t.Context(), connect.NewRequest(&v1.ListFilesRequest{Path: "links"}))
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]int64{}
	for _, one := range answer.Msg.GetEntries() {
		got[one.GetPath()] = one.GetModifiedAtMs()
	}
	if want := at.UnixMilli(); got["links/tide.md"] != want {
		t.Errorf("the note was modified at %d, want %d", got["links/tide.md"], want)
	}
	if want := folderAt.UnixMilli(); got["links/deeper"] != want {
		t.Errorf("the folder was modified at %d, want %d", got["links/deeper"], want)
	}
}

func TestAListingAnswersZeroForAnEntryWithNoModificationTime(t *testing.T) {
	v := vaultfixture.NewVault(t, map[string]string{"links/tide.md": "# Tide\n"})
	handler := newListHandler(window{vault: v}, unstamped{vaultreader.Readers{}})

	answer, err := handler.ListFiles(t.Context(), connect.NewRequest(&v1.ListFilesRequest{Path: "links"}))
	if err != nil {
		t.Fatal(err)
	}

	if len(answer.Msg.GetEntries()) == 0 {
		t.Fatal("the listing is empty")
	}
	for _, one := range answer.Msg.GetEntries() {
		if got := one.GetModifiedAtMs(); got != 0 {
			t.Errorf("%s was modified at %d, want 0", one.GetPath(), got)
		}
	}
}
