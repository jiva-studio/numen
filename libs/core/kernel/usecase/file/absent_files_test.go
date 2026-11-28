package file_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/vaultabsent"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const (
	guidePath    = "Books/Guide.pdf"
	guideContent = "the guide, page one"
)

// phone is a vault on disk with the registry and the readers of a phone.
type phone struct {
	vault   domain.Vault
	absent  vaultport.AbsentFiles
	readers vaultport.VaultReaders
}

func newPhone(t *testing.T, held map[string]string) phone {
	t.Helper()
	v := domain.Vault{ID: "01VAULT0000000000000000000", Path: vaulttest.CreateVault(t, held, nil)}
	db, err := index.Open(t.Context(), indexfile.GetPath(t), index.DisableSync())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return phone{
		vault: v, absent: db.AbsentFiles(),
		readers: vaultabsent.NewReaders(vaultreader.Readers{Options: vaultconfig.Options{}}, db.AbsentFiles()),
	}
}

func getHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func getGuideEntry(content string) vault.AbsentFile {
	return vault.AbsentFile{
		Path: guidePath, Size: int64(len(content)), Mtime: time.Date(2026, time.November, 20, 9, 30, 0, 0, time.UTC),
		Hash: getHash(content), Provider: "tools-sync",
	}
}

func (p phone) putEntry(t *testing.T, entry vault.AbsentFile) {
	t.Helper()
	if err := file.NewPutAbsentFiles(p.readers, p.absent).Execute(t.Context(), p.vault, []vault.AbsentFile{entry}); err != nil {
		t.Fatal(err)
	}
}

func (p phone) getEntries(t *testing.T) []vault.AbsentFile {
	t.Helper()
	entries, err := p.absent.ListAbsentFiles(t.Context(), p.vault)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func (p phone) hasBytes(path string) bool {
	_, err := os.Stat(filepath.Join(p.vault.Path, filepath.FromSlash(path)))
	return err == nil
}

func writeStoreFile(t *testing.T, content string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "guide.part")
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestStoreAbsentFilesListsTheFileInTheRegistry(t *testing.T) {
	p := newPhone(t, nil)
	p.putEntry(t, getGuideEntry(guideContent))
	if got := p.getEntries(t); len(got) != 1 || got[0].Path != guidePath {
		t.Fatalf("got %+v, want the entry of %s", got, guidePath)
	}
}

func TestStoreAbsentFilesRefusesAPathWithAFileOfAnotherHash(t *testing.T) {
	p := newPhone(t, map[string]string{guidePath: "my own notes in a file"})
	err := file.NewPutAbsentFiles(p.readers, p.absent).Execute(t.Context(), p.vault, []vault.AbsentFile{getGuideEntry(guideContent)})
	if !errors.Is(err, vault.ErrFilePresent) {
		t.Fatalf("got %v, want ErrFilePresent", err)
	}
	if got := p.getEntries(t); len(got) != 0 {
		t.Fatalf("got %+v, want no entry beside a file that differs", got)
	}
}

func TestStoreAbsentFilesAcceptsAPathWithTheSameContent(t *testing.T) {
	p := newPhone(t, map[string]string{guidePath: guideContent})
	p.putEntry(t, getGuideEntry(guideContent))
}

func TestFillFilePlacesTheBytesAndRemovesTheEntry(t *testing.T) {
	p := newPhone(t, nil)
	p.putEntry(t, getGuideEntry(guideContent))
	if err := p.newFill().Execute(t.Context(), p.vault, guidePath, writeStoreFile(t, guideContent)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(p.vault.Path, "Books", "Guide.pdf"))
	if err != nil || string(got) != guideContent {
		t.Fatalf("got %q, %v, want the bytes of the store file", got, err)
	}
	if entries := p.getEntries(t); len(entries) != 0 {
		t.Fatalf("got %+v, want the entry removed once the file is held", entries)
	}
}

func TestFillFileWithOtherBytesPlacesNothing(t *testing.T) {
	p := newPhone(t, nil)
	p.putEntry(t, getGuideEntry(guideContent))
	err := p.newFill().Execute(t.Context(), p.vault, guidePath, writeStoreFile(t, "damaged bytes"))
	if !errors.Is(err, plugin.ErrHashMismatch) {
		t.Fatalf("got %v, want ErrHashMismatch", err)
	}
	if p.hasBytes(guidePath) {
		t.Fatal("a file is at the path after a mismatch")
	}
	if entries := p.getEntries(t); len(entries) != 1 {
		t.Fatalf("got %+v, want the entry kept", entries)
	}
}

func TestFillFileOfAPathWithoutAnEntryPlacesNothing(t *testing.T) {
	p := newPhone(t, nil)
	if err := p.newFill().Execute(t.Context(), p.vault, guidePath, writeStoreFile(t, guideContent)); err == nil {
		t.Fatal("got nil, want an error for a path that is not listed as absent")
	}
	if p.hasBytes(guidePath) {
		t.Fatal("a file is at a path nothing listed")
	}
}

func TestEvictFileDeletesTheBytesAndListsTheFile(t *testing.T) {
	p := newPhone(t, map[string]string{guidePath: guideContent})
	if err := p.newEvict().Execute(t.Context(), p.vault, guidePath, getHash(guideContent), "tools-sync"); err != nil {
		t.Fatal(err)
	}
	if p.hasBytes(guidePath) {
		t.Fatal("the bytes are still on the disk")
	}
	entries := p.getEntries(t)
	if len(entries) != 1 || entries[0].Path != guidePath || entries[0].Hash != getHash(guideContent) || entries[0].Provider != "tools-sync" {
		t.Fatalf("got %+v, want an entry of the file with its hash and the provider", entries)
	}
	if _, err := os.Stat(filepath.Join(p.vault.Path, vaultconfig.DefaultServiceDir, "trash")); err == nil {
		t.Fatal("an eviction made a trash folder")
	}
}

func TestEvictFileRefusesAFileWithChanges(t *testing.T) {
	p := newPhone(t, map[string]string{guidePath: guideContent + ", then my notes"})
	err := p.newEvict().Execute(t.Context(), p.vault, guidePath, getHash(guideContent), "tools-sync")
	if !errors.Is(err, vault.ErrChanged) {
		t.Fatalf("got %v, want ErrChanged", err)
	}
	if !p.hasBytes(guidePath) {
		t.Fatal("a file with changes lost its bytes")
	}
	if entries := p.getEntries(t); len(entries) != 0 {
		t.Fatalf("got %+v, want no entry for a file that was kept", entries)
	}
}

func TestEvictThenFillRestoresTheFile(t *testing.T) {
	p := newPhone(t, map[string]string{guidePath: guideContent})
	if err := p.newEvict().Execute(t.Context(), p.vault, guidePath, getHash(guideContent), "tools-sync"); err != nil {
		t.Fatal(err)
	}
	if err := p.newFill().Execute(t.Context(), p.vault, guidePath, writeStoreFile(t, guideContent)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(p.vault.Path, "Books", "Guide.pdf"))
	if err != nil || string(got) != guideContent {
		t.Fatalf("got %q, %v, want the file as it was", got, err)
	}
}

func (p phone) newFill() *file.FillFile {
	noIndex := func(context.Context, domain.Vault, []string) error { return nil }
	return file.NewFillFile(p.readers, p.absent, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, filesystem.ImportedFiles{}, noIndex)
}

func (p phone) newEvict() *file.EvictFile {
	return file.NewEvictFile(p.readers, p.absent, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()})
}
