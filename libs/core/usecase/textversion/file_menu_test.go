package textversion_test

import (
	"context"
	"io/fs"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// fileReaders opens a vault whose files are the map, as they are when read.
type fileReaders map[string][]byte

func (r fileReaders) Open(domain.Vault) (vaultport.VaultReader, error) {
	return fileReader{files: r}, nil
}

type fileReader struct {
	vaultport.VaultReader
	files fileReaders
}

func (r fileReader) Read(_ context.Context, path string) ([]byte, error) {
	raw, ok := r.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func hasProofreading(entries []textversion.MenuEntry) bool {
	return slices.ContainsFunc(entries, func(e textversion.MenuEntry) bool {
		return e.Action == textversion.ActionCreateText && e.PluginID == proofreader
	})
}

func TestFileMenuUsesCurrentFileHash(t *testing.T) {
	s := newScene(t)
	menu, err := textversion.NewFileMenu(s.versions, fileReaders(s.files))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.versions.RequestText(t.Context(), s.vault, scanPath, s.fileHash()); err != nil {
		t.Fatal(err)
	}

	entries, err := menu.GetMenu(t.Context(), s.vault, scanPath)
	if err != nil || !hasProofreading(entries) {
		t.Fatalf("a file with a live version is offered %+v (%v), want a proofreading", entries, err)
	}
	s.files[scanPath] = []byte("%PDF scan, second printing")
	entries, err = menu.GetMenu(t.Context(), s.vault, scanPath)
	if err != nil || hasProofreading(entries) {
		t.Fatalf("a file replaced since its version was made is offered %+v (%v), want no proofreading", entries, err)
	}
	if _, err := menu.GetMenu(t.Context(), s.vault, "books/gone.pdf"); err == nil {
		t.Fatal("a file that is not there was given a menu")
	}
}

func TestFileMenuRunIsUserStarted(t *testing.T) {
	s := newScene(t)
	menu, err := textversion.NewFileMenu(s.versions, fileReaders(s.files))
	if err != nil {
		t.Fatal(err)
	}
	if err := menu.RunMenuEntry(t.Context(), s.vault, scanPath, textversion.MenuEntry{Action: textversion.ActionCreateText, PluginID: ocr}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(s.ocr.isUserStarted, []bool{true}) {
		t.Fatalf("the recogniser was asked as %v, want user-started", s.ocr.isUserStarted)
	}
}

func TestNewFileMenuRequiresVersionsAndReaders(t *testing.T) {
	if _, err := textversion.NewFileMenu(nil, fileReaders{}); err == nil {
		t.Error("a file menu was made without versions")
	}
	if _, err := textversion.NewFileMenu(newScene(t).versions, nil); err == nil {
		t.Error("a file menu was made without readers")
	}
}
