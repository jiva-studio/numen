package chunk_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	notecase "github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// saveNote indexes a note whose title is its own, or the filename's where the
// frontmatter names none.
func (s *store) saveNote(vault domain.VaultID, path, title string, named bool) {
	s.t.Helper()
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: 10, ModTime: time.Unix(100, 0)},
		Title:       title,
		Body:        "words of " + title,
	}
	if named {
		n.Frontmatter = map[string]any{"title": title}
	}
	if err := s.db.Notes().Save(s.t.Context(), vault, notecase.CutNotes{}.Execute([]domain.Note{n})); err != nil {
		s.t.Fatal(err)
	}
}

func (s *store) sourceRow(vault domain.VaultID, path string) int64 {
	s.t.Helper()
	got := s.ints(`SELECT s.id FROM sources s JOIN vaults v ON v.id = s.vault_id WHERE v.identifier = ? AND s.path = ?`, string(vault), path)
	if len(got) != 1 {
		s.t.Fatalf("%s is held on %v rows", path, got)
	}
	return got[0]
}

func (s *store) noteName(row int64) (title, folded, indexed string) {
	s.t.Helper()
	if err := s.side.QueryRowContext(s.t.Context(),
		`SELECT title, folded_name, COALESCE((SELECT text FROM titles_fts WHERE rowid = source_id), '') FROM notes WHERE source_id = ?`, row).
		Scan(&title, &folded, &indexed); err != nil {
		s.t.Fatal(err)
	}
	return title, folded, indexed
}

func TestAMovedFolderTakesEverythingUnderItAndNothingBesides(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "physics/Entropy.md", "Entropy", false)
	s.saveNote(repoVaultA, "physics/heat/Heat.md", "Heat", false)
	s.saveNote(repoVaultA, "physics-old/Stray.md", "Stray", false)
	s.saveNote(repoVaultA, "physics0/Next.md", "Next", false)
	s.saveNote(repoVaultB, "physics/Other.md", "Other", false)
	s.cut(repoVaultA, "book", "physics/A Book.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "book words"}})
	target := s.rowsOf(repoVaultA, "physics/A Book.epub")

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "physics", "science/physics"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"physics-old/Stray.md", "physics0/Next.md",
		"science/physics/A Book.epub", "science/physics/Entropy.md", "science/physics/heat/Heat.md",
	}
	if got := s.paths(repoVaultA); !slices.Equal(got, want) {
		t.Errorf("the vault holds %v, want %v", got, want)
	}
	if got := s.paths(repoVaultB); !slices.Equal(got, []string{"physics/Other.md"}) {
		t.Errorf("the other vault holds %v", got)
	}
	if got := s.rowsOf(repoVaultA, "science/physics/A Book.epub"); !slices.Equal(got, target) {
		t.Errorf("the book's chunks are %v, were %v", got, target)
	}
}

func TestAFileMovedToAnotherNameKeepsItsOwnFolderUnrenamed(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "Old.md", "Old", false)
	s.saveNote(repoVaultA, "Old.md.bak", "Backup", false)

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "Old.md", "New.md"); err != nil {
		t.Fatal(err)
	}

	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"New.md", "Old.md.bak"}) {
		t.Errorf("the vault holds %v", got)
	}
}

func TestAFolderWithANameThatIsNotLatinMovesByCharacters(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "физика/Энтропия.md", "Энтропия", false)

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "физика", "physics"); err != nil {
		t.Fatal(err)
	}

	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"physics/Энтропия.md"}) {
		t.Errorf("the vault holds %v", got)
	}
}

func TestAMoveOntoWhereAScanAlreadyFiledTheFileKeepsTheRowThatTravels(t *testing.T) {
	s := newStore(t)
	s.cut(repoVaultA, "book", "Old.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "old words"}})
	s.cut(repoVaultA, "book", "New.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "fresh words"}})
	s.cut(repoVaultA, "book", "New.epub.d/inner.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "inner words"}})
	target := s.rowsOf(repoVaultA, "Old.epub")
	standing := s.rowsOf(repoVaultA, "New.epub")
	if err := s.db.Chunks().SaveVectors(t.Context(), []chunk.Vector{s.vectorFor(standing[0], "m", 1)}); err != nil {
		t.Fatal(err)
	}

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "Old.epub", "New.epub"); err != nil {
		t.Fatal(err)
	}

	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"New.epub", "New.epub.d/inner.epub"}) {
		t.Errorf("the vault holds %v", got)
	}
	if got := s.rowsOf(repoVaultA, "New.epub"); !slices.Equal(got, target) {
		t.Errorf("the file at the landing holds %v, want the travelling %v", got, target)
	}
	if n := s.count(`SELECT count(*) FROM chunks_fts WHERE rowid = ?`, standing[0]); n != 0 {
		t.Error("the words of the displaced file are still indexed")
	}
	if n := s.count(`SELECT count(*) FROM chunks_vec WHERE chunk_id = ?`, standing[0]); n != 0 {
		t.Error("the coarse row of the displaced file is still indexed")
	}
}

func TestAMoveOntoAFolderDisplacesEverythingStandingUnderIt(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "draft/Idea.md", "Idea", false)
	s.saveNote(repoVaultA, "final/Idea.md", "Idea", false)
	s.saveNote(repoVaultA, "final/deep/Plan.md", "Plan", false)
	s.saveNote(repoVaultA, "final-old/Keep.md", "Keep", false)
	s.saveNote(repoVaultB, "final/Keep.md", "Keep", false)

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "draft", "final"); err != nil {
		t.Fatal(err)
	}

	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"final-old/Keep.md", "final/Idea.md"}) {
		t.Errorf("the vault holds %v", got)
	}
	if got := s.paths(repoVaultB); !slices.Equal(got, []string{"final/Keep.md"}) {
		t.Errorf("the other vault holds %v", got)
	}
}

func TestAFolderMovedInsideItselfKeepsWhatIsMoving(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "a/One.md", "One", false)
	s.saveNote(repoVaultA, "a/b/Two.md", "Two", false)
	s.saveNote(repoVaultA, "a/b/c/Three.md", "Three", false)
	s.saveNote(repoVaultA, "a/z.md", "Zed", false)

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "a/b", "a"); err != nil {
		t.Fatal(err)
	}

	// The sibling standing where the folder lands is displaced; what moves stays.
	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"a/Two.md", "a/c/Three.md"}) {
		t.Errorf("the vault holds %v", got)
	}
}

func TestAMoveOntoItsOwnPathChangesNothing(t *testing.T) {
	s := newStore(t)
	s.cut(repoVaultA, "book", "Same.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "words"}})
	target := s.rowsOf(repoVaultA, "Same.epub")

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "Same.epub", "Same.epub"); err != nil {
		t.Fatal(err)
	}

	if got := s.rowsOf(repoVaultA, "Same.epub"); !slices.Equal(got, target) {
		t.Errorf("the chunks are %v, were %v", got, target)
	}
}

func TestANoteCalledByItsFilenameIsCalledByTheFilenameItLandsUnder(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "Torpor.md", "Torpor", false)
	row := s.sourceRow(repoVaultA, "Torpor.md")

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "Torpor.md", "sleep/Hibernation.md"); err != nil {
		t.Fatal(err)
	}

	title, folded, indexed := s.noteName(row)
	if title != "Hibernation" || folded != domain.FoldName("Hibernation") || indexed != "Hibernation" {
		t.Errorf("the note is called %q, folded %q, indexed %q", title, folded, indexed)
	}
}

func TestANoteWithATitleOfItsOwnTakesItsTitleWhenTheFileLandsUnderTheNameOfIt(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "Draft.md", "Heat: a study", true)
	row := s.sourceRow(repoVaultA, "Draft.md")

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "Draft.md", "Heat- a study.md"); err != nil {
		t.Fatal(err)
	}

	title, folded, indexed := s.noteName(row)
	if title != "Heat: a study" || indexed != "Heat: a study" {
		t.Errorf("the note is called %q, indexed %q", title, indexed)
	}
	if folded != domain.FoldName("Heat- a study") {
		t.Errorf("the note is reached by %q", folded)
	}
}

func TestANoteWithATitleOfItsOwnKeepsItsNameWhenTheFileLandsUnderAnotherOne(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "Draft.md", "Heat", true)
	row := s.sourceRow(repoVaultA, "Draft.md")
	_, foldedBefore, indexedBefore := s.noteName(row)

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "Draft.md", "Unrelated.md"); err != nil {
		t.Fatal(err)
	}

	title, folded, indexed := s.noteName(row)
	if title != "Heat" || folded != foldedBefore || indexed != indexedBefore {
		t.Errorf("the note is called %q, folded %q, indexed %q", title, folded, indexed)
	}
}

func TestAFileMovedWithoutChangingNameKeepsTheNameOfTheNote(t *testing.T) {
	s := newStore(t)
	s.saveNote(repoVaultA, "a/Torpor.md", "Custom", false)
	row := s.sourceRow(repoVaultA, "a/Torpor.md")
	s.exec(`UPDATE notes SET title = 'Marked'`)

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "a/Torpor.md", "b/Torpor.md"); err != nil {
		t.Fatal(err)
	}

	if title, _, _ := s.noteName(row); title != "Marked" {
		t.Errorf("the note is called %q", title)
	}
}

func TestASourceThatIsNoNoteMovesWithoutBeingNamed(t *testing.T) {
	s := newStore(t)
	s.cut(repoVaultA, "book", "Old.epub", []chunk.Chunk{{Start: 0, Length: 4, Text: "words"}})

	if err := s.db.Chunks().MoveSources(t.Context(), repoVaultA, "Old.epub", "New.epub"); err != nil {
		t.Fatal(err)
	}

	if got := s.paths(repoVaultA); !slices.Equal(got, []string{"New.epub"}) {
		t.Errorf("the vault holds %v", got)
	}
}
