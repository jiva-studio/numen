package note_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

type savingSources struct {
	note.SourcesUnder
	save func(context.Context) error
}

func (s savingSources) GetSourcesUnder(ctx context.Context, id domain.VaultID, path string) ([]domain.FileStat, error) {
	if err := s.save(ctx); err != nil {
		return nil, err
	}
	return s.SourcesUnder.GetSourcesUnder(ctx, id, path)
}

func TestRemoveCheckedFile_KeepsAnEditSavedWhileItLooksUpTheFile(t *testing.T) {
	t.Parallel()
	const path = "Map.md"
	c := openChangeFixture(t, map[string]string{path: "original"})
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	writer, err := writers.OpenNoteWriter(c.vault)
	if err != nil {
		t.Fatal(err)
	}
	baseline := domain.FileStat{Path: path, Hash: domain.HashContent([]byte("original"))}
	sources := savingSources{SourcesUnder: c.db.Sources(), save: func(ctx context.Context) error {
		_, err := writer.Write(ctx, path, []byte("edited"), baseline)
		return err
	}}
	remove := note.NewRemove(writers, c.db.NoteQueries(), sources, c.db.Sources(), c.index, nil)
	checked := note.NewRemoveChecked(note.NewRemoveFile(&remove, nil), vaultreader.Readers{})

	if _, err := checked.Execute(t.Context(), c.vault, path, &baseline); !errors.Is(err, edit.ErrStale) {
		t.Errorf("removing a changed file returned %v, want ErrStale", err)
	}
	if got := c.read(t, path); got != "edited" {
		t.Errorf("file = %q, want the edit", got)
	}
	if _, err := os.Stat(filepath.Join(c.vault.Path, note.TrashDir, path)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stat of the trash path = %v, want no file", err)
	}
}
