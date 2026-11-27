package vaultwriter_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

type cancelAfterChecks struct {
	context.Context
	cancel        context.CancelFunc
	cancelOnCheck int
	checks        int
}

func (c *cancelAfterChecks) Err() error {
	c.checks++
	if c.checks == c.cancelOnCheck {
		c.cancel()
	}
	return c.Context.Err()
}

func TestMoveFileWithHashKeepsTheSourceWhenCanceled(t *testing.T) {
	for _, example := range []struct {
		name          string
		content       string
		cancelOnCheck int
	}{
		{name: "between reads", content: strings.Repeat("text", 32*1024), cancelOnCheck: 3},
		{name: "before rename", content: "text", cancelOnCheck: 4},
	} {
		t.Run(example.name, func(t *testing.T) {
			writer, root := openWriter(t)
			if err := writer.Create(t.Context(), "n.md", []byte(example.content)); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			canceling := &cancelAfterChecks{Context: ctx, cancel: cancel, cancelOnCheck: example.cancelOnCheck}

			err := writer.MoveFileWithHash(canceling, "n.md", "next.md", domain.HashContent([]byte(example.content)))

			if !errors.Is(err, context.Canceled) {
				t.Errorf("move returned %v, want context.Canceled", err)
			}
			if got, err := os.ReadFile(filepath.Join(root, "n.md")); err != nil || string(got) != example.content {
				t.Errorf("source read returned %d bytes, %v, want %d bytes", len(got), err, len(example.content))
			}
			if _, err := os.Stat(filepath.Join(root, "next.md")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("target stat returned %v, want no file", err)
			}
		})
	}
}

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
