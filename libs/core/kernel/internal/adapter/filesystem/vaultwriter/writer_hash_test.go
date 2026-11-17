package vaultwriter_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

func TestAWriteOverTheHashOfTheContentReturnsTheHashOfTheNewContent(t *testing.T) {
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	vault := vaulttest.NewVault(t, nil)
	writer, err := writers.Open(vault)
	if err != nil {
		t.Fatal(err)
	}
	first, err := writer.Write(t.Context(), "n.md", []byte("start"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Stat.Hash != domain.HashContent([]byte("start")) {
		t.Fatalf("the first write returned the hash %q", first.Stat.Hash)
	}

	second, err := writer.Write(t.Context(), "n.md", []byte("next"), domain.FileStat{Hash: first.Stat.Hash})

	if err != nil {
		t.Fatal(err)
	}
	if second.Stat.Hash != domain.HashContent([]byte("next")) || second.Version <= first.Version || second.Stat.Version != second.Version {
		t.Errorf("the second write returned %+v after version %d", second, first.Version)
	}
	if got := writers.GetVersion(vault); got != second.Version {
		t.Errorf("the vault counts %d writes, the last write returned %d", got, second.Version)
	}
}

func TestAWriteOverAStaleHashChangesNothingAndCountsNoWrite(t *testing.T) {
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	vault := vaulttest.NewVault(t, nil)
	writer, err := writers.Open(vault)
	if err != nil {
		t.Fatal(err)
	}
	first, err := writer.Write(t.Context(), "n.md", []byte("start"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(t.Context(), "n.md", []byte("other"), domain.FileStat{Hash: first.Stat.Hash}); err != nil {
		t.Fatal(err)
	}
	before := writers.GetVersion(vault)

	_, err = writer.Write(t.Context(), "n.md", []byte("late"), domain.FileStat{Hash: first.Stat.Hash})

	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("a write over a stale hash returned %v", err)
	}
	got, err := os.ReadFile(filepath.Join(vault.Path, "n.md"))
	if err != nil || string(got) != "other" {
		t.Errorf("the note holds %q (%v)", got, err)
	}
	if writers.GetVersion(vault) != before {
		t.Errorf("a refused write took a version")
	}
}

func TestAChangeOfContentThatKeepsSizeAndTimeIsRefusedAndSoIsABaselineWithoutAHash(t *testing.T) {
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	vault := vaulttest.NewVault(t, nil)
	writer, err := writers.Open(vault)
	if err != nil {
		t.Fatal(err)
	}
	first, err := writer.Write(t.Context(), "n.md", []byte("body"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(vault.Path, "n.md")
	if err := os.WriteFile(target, []byte("BODY"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(target, first.Stat.ModTime, first.Stat.ModTime); err != nil {
		t.Fatal(err)
	}
	bySizeAndTime := domain.FileStat{Path: "n.md", Size: first.Stat.Size, ModTime: first.Stat.ModTime}

	_, byHash := writer.Write(t.Context(), "n.md", []byte("mine"), first.Stat)
	_, bySize := writer.Write(t.Context(), "n.md", []byte("mine"), bySizeAndTime)

	if !errors.Is(byHash, edit.ErrStale) {
		t.Errorf("a write over the hash of the old content returned %v", byHash)
	}
	if !errors.Is(bySize, edit.ErrStale) {
		t.Errorf("a baseline without a hash is never the content of a file, and the write returned %v", bySize)
	}
}

func TestAHashBaselineOverAMissingNoteIsStale(t *testing.T) {
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	writer, err := writers.Open(vaulttest.NewVault(t, nil))
	if err != nil {
		t.Fatal(err)
	}

	_, err = writer.Write(t.Context(), "n.md", []byte("late"), domain.FileStat{Hash: domain.HashContent([]byte("start"))})

	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("a write over the hash of a note that is not there returned %v", err)
	}
}
