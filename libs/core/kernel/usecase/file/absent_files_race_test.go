package file_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// savingReaders opens vaults whose files save a new version of themselves when the first read of a file
// starts, as an editor does while the hash of the file is being made.
type savingReaders struct {
	vaultport.VaultReaders
	save func()
	once sync.Once
}

func (r *savingReaders) Open(v domain.Vault) (vaultport.VaultReader, error) {
	opened, err := r.VaultReaders.Open(v)
	return &savingReader{VaultReader: opened, readers: r}, err
}

type savingReader struct {
	vaultport.VaultReader
	readers *savingReaders
}

func (r *savingReader) Open(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	file, err := r.VaultReader.Open(ctx, path)
	return &savingFile{ReadSeekCloser: file, readers: r.readers}, err
}

type savingFile struct {
	io.ReadSeekCloser
	readers *savingReaders
}

func (f *savingFile) Read(p []byte) (int, error) {
	f.readers.once.Do(f.readers.save)
	return f.ReadSeekCloser.Read(p)
}

func replaceOnDisk(t *testing.T, p phone, path, content string) {
	t.Helper()
	full := filepath.Join(p.vault.Path, filepath.FromSlash(path))
	temp := full + ".saving"
	if err := os.WriteFile(temp, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	later := time.Date(2100, time.January, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(temp, later, later); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temp, full); err != nil {
		t.Fatal(err)
	}
}

func TestEvictFileKeepsAFileSavedWhileItsHashIsMade(t *testing.T) {
	p := newPhone(t, map[string]string{guidePath: guideContent})
	annotated := guideContent + ", annotated"
	p.readers = &savingReaders{VaultReaders: p.readers, save: func() { replaceOnDisk(t, p, guidePath, annotated) }}

	err := p.newEvict().Execute(t.Context(), p.vault, guidePath, getHash(guideContent), "tools-sync")

	if !errors.Is(err, vault.ErrChanged) {
		t.Fatalf("got %v, want ErrChanged", err)
	}
	got, readErr := os.ReadFile(filepath.Join(p.vault.Path, "Books", "Guide.pdf"))
	if readErr != nil || string(got) != annotated {
		t.Fatalf("got %q, %v, want the file saved during the hash", got, readErr)
	}
	if entries := p.getEntries(t); len(entries) != 0 {
		t.Fatalf("got %+v, want no entry for a file that was kept", entries)
	}
}

// failingOnce is the registry that cannot delete an entry the first time it is asked to.
type failingOnce struct {
	vaultport.AbsentFiles
	failed bool
}

var errRegistry = errors.New("the registry is locked")

func (r *failingOnce) RemoveAbsentFiles(ctx context.Context, v domain.Vault, paths []string) error {
	if !r.failed {
		r.failed = true
		return errRegistry
	}
	return r.AbsentFiles.RemoveAbsentFiles(ctx, v, paths)
}

func TestFillFileHealsTheEntryOfAFileItPlacedWhenTheRegistryFailed(t *testing.T) {
	p := newPhone(t, nil)
	p.putEntry(t, getGuideEntry(guideContent))
	p.absent = &failingOnce{AbsentFiles: p.absent}
	store := writeStoreFile(t, guideContent)
	if err := p.newFill().Execute(t.Context(), p.vault, guidePath, store); !errors.Is(err, errRegistry) {
		t.Fatalf("got %v, want the failure of the registry", err)
	}

	if err := p.newFill().Execute(t.Context(), p.vault, guidePath, store); err != nil {
		t.Fatalf("the second fill failed: %v", err)
	}

	if entries := p.getEntries(t); len(entries) != 0 {
		t.Fatalf("got %+v, want no entry for a file that is on the disk", entries)
	}
	if got, err := os.ReadFile(filepath.Join(p.vault.Path, "Books", "Guide.pdf")); err != nil || string(got) != guideContent {
		t.Fatalf("got %q, %v, want the placed file", got, err)
	}
}
