package note_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// trashSources is the index saying which sources sit under a path.
type trashSources struct {
	sourceport.SourceQueries
	underCtx []domain.FileStat
	err      error
}

func (s trashSources) GetSourcesUnder(context.Context, domain.VaultID, string) ([]domain.FileStat, error) {
	return s.underCtx, s.err
}

// trashLinks is the index saying which notes point at a path, and recording
// which paths it was asked about.
type trashLinks struct {
	noteport.LinkQueries
	pointing  map[string][]domain.ResolvedLink
	err       error
	requested *[]string
}

func (l trashLinks) GetBacklinks(_ context.Context, _ domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	*l.requested = append(*l.requested, to)
	return l.pointing[to], l.err
}

// trashWriter is a vault whose move and remove answer as they were told to.
type trashWriter struct {
	vaultport.VaultWriter
	move   func(from, to string) error
	remove func(path string) error
}

func (w trashWriter) Move(_ context.Context, from, to string) error { return w.move(from, to) }

func (w trashWriter) Remove(_ context.Context, path string) error { return w.remove(path) }

type trashWriters struct {
	vaultport.VaultWriters
	writer  trashWriter
	openErr error
}

func (w trashWriters) Open(domain.Vault) (vaultport.VaultWriter, error) {
	return w.writer, w.openErr
}

// trashing is one removal with everything it asks of the vault faked.
type trashing struct {
	sources   trashSources
	links     trashLinks
	writers   trashWriters
	moves     *[]string
	requested *[]string
	levelled  *[][]string
	levelErr  error
}

func newTrashing() *trashing {
	moves, request, levelled := &[]string{}, &[]string{}, &[][]string{}
	tr := &trashing{moves: moves, requested: request, levelled: levelled}
	tr.sources = trashSources{}
	tr.links = trashLinks{requested: request}
	tr.writers = trashWriters{writer: trashWriter{
		move: func(_, to string) error {
			*moves = append(*moves, to)
			return nil
		},
		remove: func(string) error { return nil },
	}}
	return tr
}

func (tr *trashing) remove() note.Remove {
	return note.NewRemove(tr.writers, tr.links, tr.sources, func(_ context.Context, _ domain.Vault, paths []string) error {
		*tr.levelled = append(*tr.levelled, slices.Clone(paths))
		return tr.levelErr
	})
}

func noteAt(path string) domain.FileStat {
	return domain.FileStat{Path: path, Kind: domain.KindNote}
}

func TestRemovingStopsWhenTheIndexCannotSayWhatIsBelowThePath(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	broken := errors.New("the index is gone")
	tr.sources.err = broken

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, broken) {
		t.Fatalf("want the index's own error, got %v", err)
	}
	if got.Path != "Entropy.md" {
		t.Errorf("the answer lost the path: %+v", got)
	}
	if len(*tr.moves) != 0 {
		t.Errorf("a note was moved with nothing known about it: %v", *tr.moves)
	}
}

func TestRemove_StopsWhenLinksQueryFails(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.sources.underCtx = []domain.FileStat{noteAt("Entropy.md")}
	broken := errors.New("the index is gone")
	tr.links.err = broken

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, broken) {
		t.Fatalf("want the index's own error, got %v", err)
	}
	if got.Path != "Entropy.md" {
		t.Errorf("the answer lost the path: %+v", got)
	}
	if len(*tr.moves) != 0 {
		t.Errorf("a note was moved before its links were known: %v", *tr.moves)
	}
}

func TestRemovingStopsWhenTheVaultCannotBeOpenedForWriting(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	broken := errors.New("the vault is read-only")
	tr.writers.openErr = broken

	_, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, broken) {
		t.Fatalf("want the vault's own error, got %v", err)
	}
	if len(*tr.moves) != 0 {
		t.Errorf("a note was moved in a vault that could not be written: %v", *tr.moves)
	}
}

// Only a note has links, so nothing else under a folder is asked about.
func TestRemove_QueriesBacklinksForNotesOnly(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.sources.underCtx = []domain.FileStat{
		{Path: "physics/figure.epub", Kind: domain.KindFile},
		noteAt("physics/Entropy.md"),
	}
	tr.links.pointing = map[string][]domain.ResolvedLink{
		"physics/figure.epub": {{From: "Outside.md"}},
	}

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "physics")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"physics/Entropy.md"}; !slices.Equal(*tr.requested, want) {
		t.Errorf("asked about %v, want %v", *tr.requested, want)
	}
	if len(got.Dangling) != 0 {
		t.Errorf("a note was left pointing at a file that is not a note: %v", got.Dangling)
	}
}

// The trash takes the note under the first name that is free, counting up from
// two.
func TestTheTrashNumbersACopyFromTwoUpward(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.writers.writer.move = func(_, to string) error {
		*tr.moves = append(*tr.moves, to)
		switch {
		case len(*tr.moves) < 4:
			return vaultport.ErrAlreadyExists
		case len(*tr.moves) < 150:
			return nil
		}
		return errors.New("the trash was queried too many times")
	}

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".trash/Entropy.md", ".trash/Entropy-2.md", ".trash/Entropy-3.md", ".trash/Entropy-4.md"}
	if !slices.Equal(*tr.moves, want) {
		t.Errorf("the note was offered to the trash as %v, want %v", *tr.moves, want)
	}
	if got.Trashed != ".trash/Entropy-4.md" {
		t.Errorf("want the last name offered, got %q", got.Trashed)
	}
}

// A trash that already holds a hundred copies is full: the hundredth is the
// last name tried.
func TestRemove_FullTrashFailsAfter100Names(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.writers.writer.move = func(_, to string) error {
		*tr.moves = append(*tr.moves, to)
		return vaultport.ErrAlreadyExists
	}

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if err == nil || !strings.Contains(err.Error(), "the trash already holds it") {
		t.Fatalf("want the trash refusing, got %v", err)
	}
	if len(*tr.moves) != 100 {
		t.Errorf("the trash was offered the note %d times, want 100", len(*tr.moves))
	}
	if got.Path != "Entropy.md" || got.Trashed != "" {
		t.Errorf("the answer to a rejected removal is %+v", got)
	}
	if len(*tr.levelled) != 0 {
		t.Errorf("the index was levelled for a note that is still there: %v", *tr.levelled)
	}
}

func TestRemovingLevelsTheIndexAtEachSourceThatWent(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		path     string
		underCtx []domain.FileStat
		want     []string
	}{
		"a note": {"Entropy.md", []domain.FileStat{noteAt("Entropy.md")}, []string{"Entropy.md"}},
		"a folder with one note": {
			"physics", []domain.FileStat{noteAt("physics/Entropy.md")}, []string{"physics/Entropy.md"},
		},
		"a folder with two": {
			"physics",
			[]domain.FileStat{noteAt("physics/Entropy.md"), noteAt("physics/Heat.md")},
			[]string{"physics/Entropy.md", "physics/Heat.md"},
		},
		"a path the index knows nothing under": {"physics", nil, []string{"physics"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tr := newTrashing()
			tr.sources.underCtx = tc.underCtx

			if _, err := tr.remove().Execute(t.Context(), domain.Vault{}, tc.path); err != nil {
				t.Fatal(err)
			}
			if len(*tr.levelled) != 1 || !slices.Equal((*tr.levelled)[0], tc.want) {
				t.Errorf("the index was levelled at %v, want one call at %v", *tr.levelled, tc.want)
			}
		})
	}
}

// A note that went with its target has nothing left to reach from, and the
// notes after it are still named.
func TestRemovingNamesTheNotesAfterOneThatWentWithTheFolder(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.sources.underCtx = []domain.FileStat{noteAt("physics/A.md"), noteAt("physics/B.md")}
	tr.links.pointing = map[string][]domain.ResolvedLink{
		"physics/A.md": {{From: "physics/B.md"}},
		"physics/B.md": {{From: "Outside.md"}},
	}

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "physics")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Outside.md"}; !slices.Equal(got.Dangling, want) {
		t.Errorf("left pointing at nothing: %v, want %v", got.Dangling, want)
	}
}

func TestAMoveThatFailsOtherwiseThanByAnOccupiedNameIsReported(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.writers.writer.move = func(string, string) error { return fs.ErrNotExist }

	_, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, note.ErrNoNote) {
		t.Fatalf("want ErrNoNote, got %v", err)
	}
}

// Without an index to level, a removal is the move alone.
func TestRemovingWithNoIndexIsTheMoveAlone(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.sources.underCtx = []domain.FileStat{noteAt("Entropy.md")}
	removing := tr.remove().WithIndex(nil)

	got, err := removing.Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if got.Trashed != ".trash/Entropy.md" {
		t.Errorf("want the note in the trash, got %q", got.Trashed)
	}
	if _, err := removing.Destroy(t.Context(), domain.Vault{}, "Entropy.md"); err != nil {
		t.Fatalf("a destroy with no index failed: %v", err)
	}
	if len(*tr.levelled) != 0 {
		t.Errorf("an index that was not given was levelled: %v", *tr.levelled)
	}
}

// The file is gone and the index is not level: the answer says both.
func TestRemove_ReportsIndexFailureWithRemoval(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.sources.underCtx = []domain.FileStat{noteAt("Entropy.md")}
	tr.links.pointing = map[string][]domain.ResolvedLink{"Entropy.md": {{From: "Heat.md"}}}
	broken := errors.New("the index is gone")
	tr.levelErr = broken

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, note.ErrUnlevelled) || !errors.Is(err, broken) {
		t.Fatalf("want the index's error said as unlevelled, got %v", err)
	}
	if got.Trashed != ".trash/Entropy.md" || !slices.Equal(got.Dangling, []string{"Heat.md"}) {
		t.Errorf("the removal that landed is not reported: %+v", got)
	}

	destroyed, err := tr.remove().Destroy(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, note.ErrUnlevelled) || !errors.Is(err, broken) {
		t.Fatalf("want the index's error said as unlevelled, got %v", err)
	}
	if !slices.Equal(destroyed.Dangling, []string{"Heat.md"}) {
		t.Errorf("the destroy that landed is not reported: %+v", destroyed)
	}
}

func TestAFailureToOpenTheVaultForWritingKeepsThePathInTheAnswer(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.writers.openErr = errors.New("the vault is read-only")

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if err == nil || got.Path != "Entropy.md" {
		t.Errorf("want an error beside the path, got %+v and %v", got, err)
	}
	got, err = tr.remove().Destroy(t.Context(), domain.Vault{}, "Entropy.md")
	if err == nil || got.Path != "Entropy.md" {
		t.Errorf("want an error beside the path, got %+v and %v", got, err)
	}
}

func TestAMoveThatFailsKeepsThePathInTheAnswer(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.writers.writer.move = func(string, string) error { return errors.New("the disk is gone") }

	got, err := tr.remove().Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if err == nil || got.Path != "Entropy.md" {
		t.Errorf("want an error beside the path, got %+v and %v", got, err)
	}
}

func TestDestroy_StopsWhenLinksQueryFails(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	broken := errors.New("the index is gone")
	tr.links.err = broken
	removed := false
	tr.writers.writer.remove = func(string) error {
		removed = true
		return nil
	}

	got, err := tr.remove().Destroy(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, broken) || got.Path != "Entropy.md" {
		t.Errorf("want the index's error beside the path, got %+v and %v", got, err)
	}
	if removed {
		t.Error("the file was destroyed before its links were known")
	}
}

func TestDestroyingReportsAFileThatCannotBeRemoved(t *testing.T) {
	t.Parallel()
	tr := newTrashing()
	tr.writers.writer.remove = func(string) error { return fs.ErrNotExist }

	got, err := tr.remove().Destroy(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, note.ErrNoNote) || got.Path != "Entropy.md" {
		t.Errorf("want ErrNoNote beside the path, got %+v and %v", got, err)
	}
	if len(*tr.levelled) != 0 {
		t.Errorf("the index was levelled for a file that is still there: %v", *tr.levelled)
	}
}
