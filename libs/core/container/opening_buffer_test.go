package container

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// repositoryAnswering is a note repository whose writes all answer err.
type repositoryAnswering struct {
	noteport.NoteRepository
	err error
}

func (r repositoryAnswering) Save(context.Context, domain.VaultID, []domain.NoteChunks) error {
	return r.err
}

func (r repositoryAnswering) Remove(context.Context, domain.VaultID, []string) error {
	return r.err
}

func notesAt(paths ...string) []domain.NoteChunks {
	var notes []domain.NoteChunks
	for _, path := range paths {
		notes = append(notes, domain.NoteChunks{Note: domain.Note{FileStat: domain.FileStat{Path: path}}})
	}
	return notes
}

// Each path written through the index while a walk runs is held once, in the
// order it was first written, whether it was saved or removed.
func TestBufferedNoteRepository_RecordsPathsOnceInOrder(t *testing.T) {
	buffered := &bufferedNoteRepository{NoteRepository: repositoryAnswering{}}
	buffered.StartRecording()

	if err := buffered.Save(t.Context(), "v", notesAt("b.md", "a.md")); err != nil {
		t.Fatal(err)
	}
	if err := buffered.Remove(t.Context(), "v", []string{"a.md", "c.md", "d.md"}); err != nil {
		t.Fatal(err)
	}
	if err := buffered.Save(t.Context(), "v", notesAt("c.md", "b.md")); err != nil {
		t.Fatal(err)
	}

	want := []string{"b.md", "a.md", "c.md", "d.md"}
	if got := buffered.StopRecording(); !slices.Equal(got, want) {
		t.Errorf("held %v, want %v", got, want)
	}
	if got := buffered.StopRecording(); len(got) != 0 {
		t.Errorf("held %v after they were taken", got)
	}
}

// A write that fails is answered with the failure.
func TestBufferedNoteRepository_ReturnsWriteError(t *testing.T) {
	broken := errors.New("the index is full")
	buffered := &bufferedNoteRepository{NoteRepository: repositoryAnswering{err: broken}}

	if err := buffered.Save(t.Context(), "v", notesAt("a.md")); !errors.Is(err, broken) {
		t.Errorf("a save was answered %v", err)
	}
	if err := buffered.Remove(t.Context(), "v", []string{"a.md"}); !errors.Is(err, broken) {
		t.Errorf("a removal was answered %v", err)
	}
}

// After the paths are taken the walk is over and later writes are not recorded;
// the next walk starts recording again.
func TestBufferedNoteRepository_StopsRecordingAfterStop(t *testing.T) {
	buffered := &bufferedNoteRepository{NoteRepository: repositoryAnswering{}}
	buffered.StartRecording()
	if err := buffered.Save(t.Context(), "v", notesAt("a.md")); err != nil {
		t.Fatal(err)
	}
	if got := buffered.StopRecording(); !slices.Equal(got, []string{"a.md"}) {
		t.Fatalf("held %v", got)
	}

	if err := buffered.Save(t.Context(), "v", notesAt("late.md")); err != nil {
		t.Fatal(err)
	}
	if got := buffered.StopRecording(); len(got) != 0 {
		t.Errorf("held %v, written after the walk was over", got)
	}

	buffered.StartRecording()
	if err := buffered.Save(t.Context(), "v", notesAt("next.md")); err != nil {
		t.Fatal(err)
	}
	if got := buffered.StopRecording(); !slices.Equal(got, []string{"next.md"}) {
		t.Errorf("the next walk held %v", got)
	}
}

// What is written under a walk and then fails to be read again is reported to
// the handler, and a walk with nothing written under it reports nothing.
func TestARefreshAfterTheWalkReportsItsFailure(t *testing.T) {
	for _, one := range []struct {
		what    string
		encoded bool
		reports int
	}{
		{"nothing written under the walk", false, 0},
		{"a note written under the walk", true, 1},
	} {
		cfg := Config{IndexPath: indexfile.Path(t)}
		db, err := cfg.OpenIndex(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		v := testsupport.NewVault(t, map[string]string{"Leaf.md": "---\ntitle: Leaf\n---\n\n# Leaf\n"})

		opening := cfg.VaultOpener(db)
		broken := errors.New("index unavailable")
		opening.Refresh = vault.NewRefresh(
			cfg.VaultReaders(), db.Vaults(), repositoryAnswering{err: broken},
			db.SourcesKnown(), db.Sources(), cfg.Clock(),
		)
		var received []error
		opening.ErrorHandler = func(err error) { received = append(received, err) }
		open := opening.Begin(t.Context(), v)

		got, err := open.Read(t.Context(), func() {
			if one.encoded {
				opening.Buffered.Record("Leaf.md")
			}
		})
		if err != nil {
			t.Fatalf("%s: %v", one.what, err)
		}
		if got != 1 {
			t.Errorf("%s: the walk read %d notes", one.what, got)
		}
		if len(received) != one.reports {
			t.Errorf("%s: %d failures reported, want %d: %v", one.what, len(received), one.reports, received)
		}
		for _, err := range received {
			if !errors.Is(err, broken) {
				t.Errorf("%s: reported %v", one.what, err)
			}
		}
	}
}

// The notes indexed while their modification time is recent are remembered in
// one ledger by every refresh of the index.
func TestEveryRefreshOfAnIndexSharesItsLedger(t *testing.T) {
	cfg := Config{IndexPath: indexfile.Path(t)}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	v := testsupport.NewVault(t, map[string]string{
		"One.md": "---\ntitle: One\n---\n\n# One\n",
		"Two.md": "---\ntitle: Two\n---\n\n# Two\n",
	})

	if err := cfg.VaultOpener(db).Level(t.Context(), v, []string{"One.md"}); err != nil {
		t.Fatal(err)
	}
	if got := db.Ledger().Len(); got != 1 {
		t.Fatalf("the opener's refresh left %d notes in the index's ledger", got)
	}
	if err := cfg.Level(db)(t.Context(), v, []string{"Two.md"}); err != nil {
		t.Fatal(err)
	}
	if got := db.Ledger().Len(); got != 2 {
		t.Errorf("the levelling left %d notes in the index's ledger", got)
	}
}
