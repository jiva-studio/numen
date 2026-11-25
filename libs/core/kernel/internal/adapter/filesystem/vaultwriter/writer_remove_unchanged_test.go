package vaultwriter_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func writeBook(t *testing.T, root, content string, modified time.Time) domain.FileStat {
	t.Helper()
	full := filepath.Join(root, "Book.pdf")
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(full, modified, modified); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(full)
	if err != nil {
		t.Fatal(err)
	}
	return domain.FileStat{Path: "Book.pdf", Size: info.Size(), ModTime: info.ModTime()}
}

func TestRemoveUnchanged_FileAsDescribed_DeletesIt(t *testing.T) {
	root := t.TempDir()
	writer, err := vaultwriter.OpenForWriting(root, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}
	was := writeBook(t, root, "pages", time.Now().Add(-time.Hour))

	if err := writer.RemoveUnchanged(t.Context(), "Book.pdf", was); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, "Book.pdf")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the file is still there: %v", err)
	}
}

func TestRemoveUnchanged_FileSavedSinceTheStat_KeepsItAndReturnsErrChanged(t *testing.T) {
	root := t.TempDir()
	writer, err := vaultwriter.OpenForWriting(root, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}
	was := writeBook(t, root, "pages", time.Now().Add(-time.Hour))
	writeBook(t, root, "pages, annotated", time.Now())

	err = writer.RemoveUnchanged(t.Context(), "Book.pdf", was)

	if !errors.Is(err, vault.ErrChanged) {
		t.Fatalf("error = %v, want ErrChanged", err)
	}
	if got, readErr := os.ReadFile(filepath.Join(root, "Book.pdf")); readErr != nil || string(got) != "pages, annotated" {
		t.Errorf("file = %q, %v, want the saved version", got, readErr)
	}
}

func TestRemoveUnchanged_NoFile_Succeeds(t *testing.T) {
	writer, err := vaultwriter.OpenForWriting(t.TempDir(), vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.RemoveUnchanged(t.Context(), "Book.pdf", domain.FileStat{}); err != nil {
		t.Errorf("error = %v, want none: a file that is gone is removed", err)
	}
}
