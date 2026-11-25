package vaultwriter_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

type editWhileReading struct {
	source io.Reader
	edit   func()
	isDone bool
}

func (e *editWhileReading) Read(p []byte) (int, error) {
	if !e.isDone {
		e.isDone = true
		e.edit()
	}
	return e.source.Read(p)
}

func TestReplace_KeepsAFileEditedDuringTheReplaceAndIsStale(t *testing.T) {
	writer, root := openWriter(t)
	path := filepath.Join(root, "Doc.pdf")
	if err := os.WriteFile(path, []byte("read by the sync"), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := domain.FileStat{Path: "Doc.pdf", Hash: domain.HashContent([]byte("read by the sync"))}
	content := &editWhileReading{source: strings.NewReader("from the server"), edit: func() {
		if err := os.WriteFile(path, []byte("typed by the person"), 0o644); err != nil {
			t.Error(err)
		}
	}}

	err := writer.Replace(t.Context(), "Doc.pdf", content, baseline)

	if !errors.Is(err, edit.ErrStale) {
		t.Fatalf("error = %v, want ErrStale", err)
	}
	if got := readFile(t, path); got != "typed by the person" {
		t.Errorf("file = %q, want the edit of the person", got)
	}
	leftovers, err := filepath.Glob(filepath.Join(root, ".Doc.pdf.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Errorf("temporary files left: %v", leftovers)
	}
}

func TestReplace_AFileThatIsStillTheBaselineIsReplaced(t *testing.T) {
	writer, root := openWriter(t)
	path := filepath.Join(root, "Doc.pdf")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := domain.FileStat{Path: "Doc.pdf", Hash: domain.HashContent([]byte("old"))}

	if err := writer.Replace(t.Context(), "Doc.pdf", strings.NewReader("new"), baseline); err != nil {
		t.Fatal(err)
	}

	if got := readFile(t, path); got != "new" {
		t.Errorf("file = %q, want the new content", got)
	}
}

func TestImport_KeepsAPathCreatedDuringTheImportAndRefuses(t *testing.T) {
	writer, root := openWriter(t)
	path := filepath.Join(root, "Doc.pdf")
	content := &editWhileReading{source: strings.NewReader("from the server"), edit: func() {
		if err := os.WriteFile(path, []byte("made by the person"), 0o644); err != nil {
			t.Error(err)
		}
	}}

	err := writer.Import(t.Context(), "Doc.pdf", content)

	if !errors.Is(err, vault.ErrAlreadyExists) {
		t.Fatalf("error = %v, want ErrAlreadyExists", err)
	}
	if got := readFile(t, path); got != "made by the person" {
		t.Errorf("file = %q, want the file of the person", got)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
