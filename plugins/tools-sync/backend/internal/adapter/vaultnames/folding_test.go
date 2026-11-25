package vaultnames_test

import (
	"context"
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/vaultnames"
)

// foldingDisk is a vault on a disk that finds a file by a name of either form, as the disk of a Mac does, and lists the
// name the file was made with.
type foldingDisk struct {
	vault *hosttest.Vault
	names map[string]string
}

func createDisk(vault *hosttest.Vault) *foldingDisk {
	return &foldingDisk{vault: vault, names: map[string]string{}}
}

func (f *foldingDisk) put(name, text string) {
	f.names[norm.NFC.String(name)] = name
	f.vault.WriteFile(name, text)
}

func (f *foldingDisk) find(path string) string {
	if name, isKnown := f.names[norm.NFC.String(path)]; isKnown {
		return name
	}
	return path
}

func (f *foldingDisk) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return f.vault.StatFile(ctx, f.find(path))
}

func (f *foldingDisk) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	return f.vault.Read(ctx, f.find(path))
}

func (f *foldingDisk) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	return f.vault.Write(ctx, f.find(path), content, baseline)
}

func (f *foldingDisk) Create(ctx context.Context, path string, content []byte) error {
	if _, isKnown := f.names[norm.NFC.String(path)]; !isKnown {
		f.names[norm.NFC.String(path)] = path
	}
	return f.vault.Create(ctx, f.find(path), content)
}

func (f *foldingDisk) ListFileChanges(ctx context.Context, q host.FileChangeQuery) (host.FileChangePage, error) {
	return f.vault.ListFileChanges(ctx, q)
}

func TestFiles_OnADiskThatFoldsNamesListsOneNoteAndWritesOneFile(t *testing.T) {
	vault := hosttest.New(string(vaultID))
	disk := createDisk(vault)
	disk.put(decomposed, "one")
	files := vaultnames.New().GetFiles(vaultID, disk)

	paths := listPaths(t, files)
	_, stat, err := files.Read(t.Context(), composed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := files.Write(t.Context(), composed, []byte("two"), stat); err != nil {
		t.Fatal(err)
	}

	if len(paths) != 1 || paths[0] != composed {
		t.Errorf("listed %q, want only %q", paths, composed)
	}
	if got, _ := vault.ReadContent(decomposed); got != "two" {
		t.Errorf("the file holds %q, want two", got)
	}
	if _, isThere := vault.ReadContent(composed); isThere {
		t.Error("a second file was made under the composed name")
	}
}
