package vault_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// watchedLinks answers like the index, records each note whose backlinks were
// queried, and fails where configured.
type watchedLinks struct {
	noteport.LinkQueries
	requested    []string
	backlinksErr error
	linksErr     error
}

func (w *watchedLinks) GetBacklinks(ctx context.Context, id domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	w.requested = append(w.requested, to)
	if w.backlinksErr != nil {
		return nil, w.backlinksErr
	}
	return w.LinkQueries.GetBacklinks(ctx, id, to)
}

func (w *watchedLinks) GetLinks(ctx context.Context, id domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	if w.linksErr != nil {
		return nil, w.linksErr
	}
	return w.LinkQueries.GetLinks(ctx, id, from)
}

// unknowing is an index that cannot say what the vault holds under a path.
type unknowing struct {
	sourceport.SourceQueries
	err error
}

func (u unknowing) GetSourcesUnder(context.Context, domain.VaultID, string) ([]domain.FileStat, error) {
	return nil, u.err
}

// unfiling is an index that cannot file a source under another path.
type unfiling struct {
	sourceport.SourceRepository
	err error
}

func (u unfiling) MoveSources(context.Context, domain.VaultID, string, string) error { return u.err }

// wiring is everything a Move relies on that a test may replace.
type wiring struct {
	links   noteport.LinkQueries
	queries sourceport.SourceQueries
	sources sourceport.SourceRepository
	writers vaultport.VaultWriters
	index   func(context.Context, domain.Vault, []string) error
}

func (f filing) wiring() wiring {
	return wiring{
		links:   f.db.Links(),
		queries: f.db.SourcesKnown(),
		sources: f.db.Sources(),
		writers: vaultwriter.VaultWriters{},
		index:   f.index,
	}
}

// moveOver returns the move of an unconfigured installation, built from what
// the test supplies.
func (f filing) moveOver(w wiring) vaults.Move {
	notes := note.NewMove(
		f.readers, vaultwriter.VaultWriters{},
		w.links, f.db.Queries(), f.db.Sources(), w.index, testsupport.NewClock(),
	)
	notes.Sync = func() note.SyncTitleAndFilename { return true }
	notes.Drawing = func(_ context.Context, went domain.Move) error {
		*f.went = append(*f.went, went)
		return nil
	}
	return vaults.NewMove(w.writers, w.links, w.queries, w.sources, notes)
}

// recordBooks files the books the vault holds in the index.
func (f filing) recordBooks(t *testing.T) {
	t.Helper()
	extract := filetext.NewIndex(vaultreader.Readers{}, f.db.Sources(), f.db.SourcesKnown(), testsupport.Texts{Readers: vaultreader.Readers{}})
	if _, err := extract.Execute(t.Context(), f.vault); err != nil {
		t.Fatal(err)
	}
}

// exists is whether the vault holds anything at the path.
func (f filing) exists(path string) bool {
	_, err := os.Lstat(filepath.Join(f.vault.Path, filepath.FromSlash(path)))
	return err == nil
}

// A path moved onto itself is not a move.
func TestAMoveOntoItselfIsNothingCompleted(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})

	moved, err := f.move().Execute(t.Context(), f.vault, "physics/Entropy.md", "physics/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if moved.From != "physics/Entropy.md" || moved.To != "physics/Entropy.md" {
		t.Errorf("the answer names the move as %q to %q", moved.From, moved.To)
	}
	if moved.IsLanded {
		t.Error("a move onto itself says it landed")
	}
	if got := f.read(t, "physics/Entropy.md"); got != "# Entropy\n" {
		t.Errorf("the note became %q", got)
	}
}

// A move that cannot query the index for the affected notes moves nothing.
func TestAMoveThatCannotAskWhatTravelsMovesNothing(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})
	cannot := errors.New("the index cannot be read")
	w := f.wiring()
	w.queries = unknowing{SourceQueries: w.queries, err: cannot}

	moved, err := f.moveOver(w).Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Entropy.md")
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if moved.From != "physics/Entropy.md" || moved.To != "archive/Entropy.md" {
		t.Errorf("the rejection names the move as %q to %q", moved.From, moved.To)
	}
	if !f.exists("physics/Entropy.md") {
		t.Error("the note moved")
	}
}

// Backlinks are queried for every note under a folder, and for nothing else.
func TestMove_QueriesBacklinksForNotesInFolderOnly(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{
		"physics/Entropy.md":   "# Entropy\n",
		"physics/heat/Heat.md": "# Heat\n",
		"physics/Carnot.md":    "# Carnot\n",
	})
	testsupport.WriteBook(t, f.vault.Path, "physics/A Book.epub")
	f.recordBooks(t)
	watched := &watchedLinks{LinkQueries: f.db.Links()}
	w := f.wiring()
	w.links = watched

	if _, err := f.moveOver(w).Execute(t.Context(), f.vault, "physics", "science/physics"); err != nil {
		t.Fatal(err)
	}

	slices.Sort(watched.requested)
	want := []string{"physics/Carnot.md", "physics/Entropy.md", "physics/heat/Heat.md"}
	if !slices.Equal(watched.requested, want) {
		t.Errorf("what points at %v was asked, want %v", watched.requested, want)
	}
}

// Every note of a folder is reported with its new path, and a book is not a
// note.
func TestEveryNoteOfAFolderIsReportedWhereItWent(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{
		"physics/Entropy.md":   "# Entropy\n",
		"physics/heat/Heat.md": "# Heat\n",
		"physics/Carnot.md":    "# Carnot\n",
	})
	testsupport.WriteBook(t, f.vault.Path, "physics/A Book.epub")
	f.recordBooks(t)

	moved, err := f.moveOver(f.wiring()).Execute(t.Context(), f.vault, "physics", "science/physics")
	if err != nil {
		t.Fatal(err)
	}
	if !moved.IsLanded {
		t.Fatal("the folder did not land")
	}

	got := make([]string, 0, len(*f.went))
	for _, went := range *f.went {
		got = append(got, went.From+" -> "+went.To)
	}
	slices.Sort(got)
	want := []string{
		"physics/Carnot.md -> science/physics/Carnot.md",
		"physics/Entropy.md -> science/physics/Entropy.md",
		"physics/heat/Heat.md -> science/physics/heat/Heat.md",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the notes were told %v, want %v", got, want)
	}
}

// A move that cannot ask what points at a note moves nothing.
func TestAMoveThatCannotAskWhatPointsAtANoteMovesNothing(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})
	cannot := errors.New("the links cannot be read")
	w := f.wiring()
	w.links = &watchedLinks{LinkQueries: f.db.Links(), backlinksErr: cannot}

	moved, err := f.moveOver(w).Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Entropy.md")
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if moved.From != "physics/Entropy.md" || moved.To != "archive/Entropy.md" {
		t.Errorf("the rejection names the move as %q to %q", moved.From, moved.To)
	}
	if !f.exists("physics/Entropy.md") {
		t.Error("the note moved")
	}
}

// A vault that cannot be written moves nothing.
func TestAMoveInAVaultThatCannotBeSavedIsAnError(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})
	cannot := errors.New("the vault cannot be written")
	w := f.wiring()
	w.writers = unwritable{err: cannot}

	moved, err := f.moveOver(w).Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Entropy.md")
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if moved.From != "physics/Entropy.md" || moved.To != "archive/Entropy.md" {
		t.Errorf("the rejection names the move as %q to %q", moved.From, moved.To)
	}
}

// A path the vault holds nothing at is no note, and nothing landed.
func TestAMoveOfAPathThatContainsNothingIsNoNote(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})

	moved, err := f.move().Execute(t.Context(), f.vault, "physics/Missing.md", "archive/Missing.md")
	if !errors.Is(err, note.ErrNoNote) {
		t.Fatalf("got %v", err)
	}
	if moved.IsLanded {
		t.Error("the move says it landed")
	}
	if moved.From != "physics/Missing.md" || moved.To != "archive/Missing.md" {
		t.Errorf("the rejection names the move as %q to %q", moved.From, moved.To)
	}
}

// A file that landed whose filing in the index failed is reported with the
// error, and it is where it was sent.
func TestAMoveThatCannotFileTheSourcesIsAnErrorAfterTheFileLanded(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})
	cannot := errors.New("the index cannot be written")
	w := f.wiring()
	w.sources = unfiling{SourceRepository: w.sources, err: cannot}

	moved, err := f.moveOver(w).Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Entropy.md")
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if !moved.IsLanded || !f.exists("archive/Entropy.md") {
		t.Error("the file is not where it was sent")
	}
}

// A note renamed within its folder carries the new name in its title.
func TestRenamingANoteWritesTheNewNameIntoItsTitle(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{
		"physics/Entropy.md": "---\ntitle: Entropy\n---\nA measure of disorder.\n",
	})

	if _, err := f.move().Execute(t.Context(), f.vault, "physics/Entropy.md", "physics/Thermo.md"); err != nil {
		t.Fatal(err)
	}
	if got := f.read(t, "physics/Thermo.md"); !strings.Contains(got, "title: Thermo") {
		t.Errorf("the note after the rename:\n%s", got)
	}
}

// A folder holding one note, given another name, is not a note renamed.
func TestAFolderOfOneNoteRenamedIsNotANoteRenamed(t *testing.T) {
	isolateHome(t)
	body := "---\ntitle: Entropy\n---\nA measure of disorder.\n"
	f := openFiling(t, map[string]string{"physics/Entropy.md": body})

	if _, err := f.move().Execute(t.Context(), f.vault, "physics", "science"); err != nil {
		t.Fatal(err)
	}
	if got := f.read(t, "science/Entropy.md"); got != body {
		t.Errorf("the note inside the folder became\n%s", got)
	}
}

// A book given another name is a file renamed, and its bytes stay as they
// were.
func TestARenamedBookKeepsItsBytes(t *testing.T) {
	isolateHome(t)
	body := "---\ntitle: A Book\n---\nWords.\n"
	f := openFiling(t, map[string]string{"A Book.epub": body})
	f.recordBooks(t)

	if _, err := f.move().Execute(t.Context(), f.vault, "A Book.epub", "Another Book.epub"); err != nil {
		t.Fatal(err)
	}
	if got := f.read(t, "Another Book.epub"); got != body {
		t.Errorf("the book became\n%s", got)
	}
}

// A note moved to another folder under another name keeps its title.
func TestAMoveToAnotherFolderKeepsTheTitle(t *testing.T) {
	isolateHome(t)
	body := "---\ntitle: Entropy\n---\nA measure of disorder.\n"
	f := openFiling(t, map[string]string{"physics/Entropy.md": body})

	if _, err := f.move().Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Thermo.md"); err != nil {
		t.Fatal(err)
	}
	if got := f.read(t, "archive/Thermo.md"); got != body {
		t.Errorf("the note became\n%s", got)
	}
}

// A link written by the path a note had is written again by its new name, and
// the note holding it is named.
func TestALinkThatStoppedResolvingIsRepairedAndNamed(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
	})

	moved, err := f.move().Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(moved.Repaired, []string{"physics/Heat.md"}) {
		t.Errorf("the links written again are %v", moved.Repaired)
	}
	if got := f.read(t, "physics/Heat.md"); !strings.Contains(got, "to: Thermodynamics") {
		t.Errorf("the repair wrote\n%s", got)
	}
}

// A link nothing could be written into is named as one now reaching nothing.
func TestALinkThatCouldNotBeRepairedIsNamed(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Cold.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Cold\n",
	})
	broken := "---\nlinks:\n\t- to: physics/Entropy.md\n    role: parent\n---\n# Cold\n"
	if err := os.WriteFile(filepath.Join(f.vault.Path, "physics", "Cold.md"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	moved, err := f.move().Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(moved.Dangling, []string{"physics/Cold.md"}) {
		t.Errorf("the links now reaching nothing are %v", moved.Dangling)
	}
}

// A note that cannot be settled after it moved is an error, and the file is
// where it was sent.
func TestANoteThatCannotBeSettledIsAnErrorAfterTheFileLanded(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
	})
	cannot := errors.New("the links cannot be read")
	w := f.wiring()
	w.links = &watchedLinks{LinkQueries: f.db.Links(), linksErr: cannot}

	moved, err := f.moveOver(w).Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if !moved.IsLanded {
		t.Error("the file is not where it was sent")
	}
}

// A Move given another index files through that index, and reports which.
func TestAMoveGivenAnotherIndexFilesThroughIt(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})
	cannot := errors.New("the other index cannot be written")
	other := unfiling{SourceRepository: f.db.Sources(), err: cannot}

	provided := f.move().WithSources(other)
	if _, ok := provided.Sources().(unfiling); !ok {
		t.Fatalf("the move files through %T", provided.Sources())
	}
	if _, err := provided.Execute(t.Context(), f.vault, "physics/Entropy.md", "archive/Entropy.md"); !errors.Is(err, cannot) {
		t.Errorf("got %v", err)
	}
}

// A title that could not be written on a rename is reported once the file is
// where it was sent.
func TestATitleThatCouldNotBeSavedOnARenameIsAnError(t *testing.T) {
	isolateHome(t)
	f := openFiling(t, map[string]string{
		"physics/Entropy.md": "---\ntitle: Entropy\n---\nA measure of disorder.\n",
	})
	cannot := errors.New("the index cannot be levelled")
	w := f.wiring()
	w.index = func(context.Context, domain.Vault, []string) error { return cannot }

	moved, err := f.moveOver(w).Execute(t.Context(), f.vault, "physics/Entropy.md", "physics/Thermo.md")
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if !moved.IsLanded || !f.exists("physics/Thermo.md") {
		t.Error("the file is not where it was sent")
	}
}
