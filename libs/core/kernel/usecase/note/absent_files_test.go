package note_test

import (
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

func getGuide(path string) vault.AbsentFile {
	return vault.AbsentFile{
		Path: path, Size: 5 << 20, Mtime: time.Date(2026, time.November, 20, 9, 30, 0, 0, time.UTC),
		Hash: "ab12", Provider: "tools-sync",
	}
}

func listPaths(t *testing.T, db *scanIndex, v domain.Vault) []string {
	t.Helper()
	entries, err := db.AbsentFiles().ListAbsentFiles(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, len(entries))
	for i, entry := range entries {
		paths[i] = entry.Path
	}
	return paths
}

func TestRemovingAnAbsentFileDeletesItsEntryAndUpdatesTheIndex(t *testing.T) {
	t.Parallel()
	db := openIndex(t)
	v := domain.Vault{ID: "01VAULT0000000000000000000"}
	if err := db.AbsentFiles().PutAbsentFiles(t.Context(), v, []vault.AbsentFile{getGuide("Books/Guide.pdf")}); err != nil {
		t.Fatal(err)
	}
	tr := newTrashScene()
	tr.writers.writer.move = func(string, string) error { return fs.ErrNotExist }

	got, err := tr.remove().WithAbsent(db.AbsentFiles()).Execute(t.Context(), v, "Books/Guide.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if got.Trashed != "" {
		t.Errorf("trashed %q, want nothing: there were no bytes", got.Trashed)
	}
	if paths := listPaths(t, db, v); len(paths) != 0 {
		t.Errorf("the registry still lists %v", paths)
	}
	if len(*tr.updated) != 1 || (*tr.updated)[0][0] != "Books/Guide.pdf" {
		t.Errorf("the index was updated for %v, want the path of the file, which makes its removal", *tr.updated)
	}
}

func TestRemovingAFolderDeletesTheEntriesBelowIt(t *testing.T) {
	t.Parallel()
	db := openIndex(t)
	v := domain.Vault{ID: "01VAULT0000000000000000000"}
	entries := []vault.AbsentFile{getGuide("Books/Guide.pdf"), getGuide("Books2/Other.pdf"), getGuide("Books/Deep/Atlas.pdf")}
	if err := db.AbsentFiles().PutAbsentFiles(t.Context(), v, entries); err != nil {
		t.Fatal(err)
	}
	tr := newTrashScene()

	if _, err := tr.remove().WithAbsent(db.AbsentFiles()).Execute(t.Context(), v, "Books"); err != nil {
		t.Fatal(err)
	}
	if paths := listPaths(t, db, v); len(paths) != 1 || paths[0] != "Books2/Other.pdf" {
		t.Errorf("the registry lists %v, want only the folder that is not named", paths)
	}
}

func TestRemovingAPathThatNothingHoldsStillFails(t *testing.T) {
	t.Parallel()
	db := openIndex(t)
	tr := newTrashScene()
	tr.writers.writer.move = func(string, string) error { return fs.ErrNotExist }

	_, err := tr.remove().WithAbsent(db.AbsentFiles()).Execute(t.Context(), domain.Vault{ID: "01VAULT0000000000000000000"}, "Books/None.pdf")
	if !errors.Is(err, edit.ErrNoNote) {
		t.Fatalf("got %v, want ErrNoNote", err)
	}
}

func TestMovingAnAbsentFileMovesItsEntry(t *testing.T) {
	t.Parallel()
	f := openFiling(t, map[string]string{"Shopping list.md": "milk"})
	if err := f.db.AbsentFiles().PutAbsentFiles(t.Context(), f.vault, []vault.AbsentFile{getGuide("Books/Guide.pdf")}); err != nil {
		t.Fatal(err)
	}
	moving := tree{f.move().MoveFile.WithAbsent(f.db.AbsentFiles())}

	if _, err := moving.Execute(t.Context(), f.vault, "Books/Guide.pdf", "Reading/Guide.pdf"); err != nil {
		t.Fatal(err)
	}
	entry, isListed, err := f.db.AbsentFiles().GetAbsentFile(t.Context(), f.vault, "Reading/Guide.pdf")
	if err != nil || !isListed || entry.Hash != "ab12" || entry.Provider != "tools-sync" {
		t.Fatalf("got %+v, %v, %v, want the entry at its new path with its hash", entry, isListed, err)
	}
	if paths := listPaths(t, f.db, f.vault); len(paths) != 1 {
		t.Errorf("the registry lists %v, want the one entry", paths)
	}
}

func TestMovingAFolderMovesTheEntriesBelowIt(t *testing.T) {
	t.Parallel()
	f := openFiling(t, map[string]string{"Books/Atlas.md": "maps"})
	if err := f.db.AbsentFiles().PutAbsentFiles(t.Context(), f.vault, []vault.AbsentFile{getGuide("Books/Guide.pdf")}); err != nil {
		t.Fatal(err)
	}
	moving := tree{f.move().MoveFile.WithAbsent(f.db.AbsentFiles())}

	if _, err := moving.Execute(t.Context(), f.vault, "Books", "Reading"); err != nil {
		t.Fatal(err)
	}
	if paths := listPaths(t, f.db, f.vault); len(paths) != 1 || paths[0] != "Reading/Guide.pdf" {
		t.Errorf("the registry lists %v, want Reading/Guide.pdf", paths)
	}
}

func TestMovingAPathThatNothingHoldsStillFails(t *testing.T) {
	t.Parallel()
	f := openFiling(t, map[string]string{"Shopping list.md": "milk"})
	moving := tree{f.move().MoveFile.WithAbsent(f.db.AbsentFiles())}

	if _, err := moving.Execute(t.Context(), f.vault, "Books/None.pdf", "Reading/None.pdf"); !errors.Is(err, edit.ErrNoNote) {
		t.Fatalf("got %v, want ErrNoNote", err)
	}
}
