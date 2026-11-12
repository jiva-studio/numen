package index

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func chunksOf(t *testing.T, db *DB, vault domain.Vault, path string) []int64 {
	t.Helper()
	return rowsOf(t, db,
		`SELECT c.id FROM chunks c JOIN sources s ON s.id = c.source_id
		 WHERE s.vault_id = (SELECT id FROM vaults WHERE identifier = ?) AND s.path = ?
		 ORDER BY c.id`, vault.ID, path)
}

// A folder is moved with everything under it, for any type of file,
// and what was derived from each source stays with the source.
func TestAMovedFolderConsumesEverythingBelowIt(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "physics/Entropy.md", "Entropy")
	saveNamedNote(t, db, getFirstVault(), "physics/heat/Heat.md", "Heat")
	saveNamedNote(t, db, getFirstVault(), "physics-old/Stray.md", "Stray")
	book(t, db, getFirstVault(), "physics/A Book.epub", 1)
	target := chunksOf(t, db, getFirstVault(), "physics/Entropy.md")
	if len(target) == 0 {
		t.Fatal("the note was indexed with no chunks, and this test asks what happens to them")
	}

	if err := db.Sources().MoveSources(t.Context(), getFirstVault().ID, "physics", "science/physics"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"science/physics/A Book.epub",
		"science/physics/Entropy.md",
		"science/physics/heat/Heat.md",
	}
	if got := getPathsUnder(t, db, getFirstVault(), "science/physics"); !slices.Equal(got, want) {
		t.Errorf("the folder now holds %v, want %v", got, want)
	}
	if got := getPathsUnder(t, db, getFirstVault(), "physics"); len(got) != 0 {
		t.Errorf("the folder it left still holds %v", got)
	}
	if got := getPathsUnder(t, db, getFirstVault(), "physics-old"); !slices.Equal(got, []string{"physics-old/Stray.md"}) {
		t.Errorf("a folder whose name begins with the one that moved traveled: %v", got)
	}
	if got := chunksOf(t, db, getFirstVault(), "science/physics/Entropy.md"); !slices.Equal(got, target) {
		t.Errorf("the note's chunks are %v, and were %v", got, target)
	}
}

// One database holds every vault, and a folder of one name moves in the vault
// it was named in.
func TestAMovedFolderMovesInOneVaultAlone(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "physics/Entropy.md", "Entropy")
	saveNamedNote(t, db, getSecond(), "physics/Quasar.md", "Quasar")

	if err := db.Sources().MoveSources(t.Context(), getFirstVault().ID, "physics", "science"); err != nil {
		t.Fatal(err)
	}

	if got := getPathsUnder(t, db, getSecond(), "physics"); !slices.Equal(got, []string{"physics/Quasar.md"}) {
		t.Errorf("the other vault's folder holds %v", got)
	}
	if got := getPathsUnder(t, db, getSecond(), "science"); len(got) != 0 {
		t.Errorf("the other vault was given %v", got)
	}
}

// A path is bytes on disk and characters in the index, and a folder whose name
// is not Latin holds the two apart.
func TestAMovedFolderCarriesANameThatIsNotLatin(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "физика/Энтропия.md", "Энтропия")

	if err := db.Sources().MoveSources(t.Context(), getFirstVault().ID, "физика", "physics"); err != nil {
		t.Fatal(err)
	}

	want := []string{"physics/Энтропия.md"}
	if got := getPathsUnder(t, db, getFirstVault(), "physics"); !slices.Equal(got, want) {
		t.Errorf("the folder now holds %v, want %v", got, want)
	}
}

// A pass that ran between the move on disk and the move here has already
// replaced the rows at the old path with the rows at the new one. Those rows
// are the current ones and stay.
func TestAMoveKeepsTheRowsAScanFiledWhenNothingIsLeftToMove(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "science/physics/Entropy.md", "Entropy")

	if err := db.Sources().MoveSources(t.Context(), getFirstVault().ID, "physics", "science/physics"); err != nil {
		t.Fatal(err)
	}

	want := []string{"science/physics/Entropy.md"}
	if got := getPathsUnder(t, db, getFirstVault(), "science/physics"); !slices.Equal(got, want) {
		t.Errorf("the folder holds %v, want %v", got, want)
	}
}

// A file is moved on disk and then filed here, and a scan reading the vault
// between the two puts the file in afresh under its new path. The move lands,
// and the row that carries the note's chunks is the one that keeps the path.
func TestAMoveLandsWhereAScanHasAlreadyFiledTheFile(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "Old.md", "Old", "Entropy is")
	saveNamedNote(t, db, getFirstVault(), "Entropy.md", "Old")
	target := chunksOf(t, db, getFirstVault(), "Old.md")
	if len(target) == 0 {
		t.Fatal("the note was indexed with no chunks, and this test asks what happens to them")
	}

	if err := db.Sources().MoveSources(t.Context(), getFirstVault().ID, "Old.md", "Entropy.md"); err != nil {
		t.Fatal(err)
	}

	if got := getPathsUnder(t, db, getFirstVault(), "Entropy.md"); !slices.Equal(got, []string{"Entropy.md"}) {
		t.Errorf("the vault holds %v where the move landed", got)
	}
	if got := getPathsUnder(t, db, getFirstVault(), "Old.md"); len(got) != 0 {
		t.Errorf("the path it left still holds %v", got)
	}
	if got := chunksOf(t, db, getFirstVault(), "Entropy.md"); !slices.Equal(got, target) {
		t.Errorf("the note's chunks are %v, and were %v", got, target)
	}
}

// A move that goes nowhere leaves the note where it is: the row standing at the
// path is the row about to be filed there.
func TestAMoveOntoItsOwnPathKeepsTheNote(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "Entropy.md", "Entropy", "Entropy is")
	target := chunksOf(t, db, getFirstVault(), "Entropy.md")

	if err := db.Sources().MoveSources(t.Context(), getFirstVault().ID, "Entropy.md", "Entropy.md"); err != nil {
		t.Fatal(err)
	}

	if got := getPathsUnder(t, db, getFirstVault(), "Entropy.md"); !slices.Equal(got, []string{"Entropy.md"}) {
		t.Errorf("the vault holds %v at the path", got)
	}
	if got := chunksOf(t, db, getFirstVault(), "Entropy.md"); !slices.Equal(got, target) {
		t.Errorf("the note's chunks are %v, and were %v", got, target)
	}
}

// A note called by its filename is searched by the name it now carries.
func TestARenamedNoteIsFoundByTheNameItLandsBelow(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "Torpor.md", "Torpor")

	if err := db.Sources().MoveSources(t.Context(), getFirstVault().ID, "Torpor.md", "Hibernation.md"); err != nil {
		t.Fatal(err)
	}

	found := getNames(t, db, getFirstVault(), "Hibernation")
	if len(found) != 1 {
		t.Fatalf("the name it landed under found %d notes: %+v", len(found), found)
	}
	if found[0].Path != "Hibernation.md" || found[0].Title != "Hibernation" {
		t.Errorf("found %+v, want the note under its new name", found[0])
	}
	if got := describeMatch(found[0]); got != "[Hibernation]" {
		t.Errorf("marked %q, want %q", got, "[Hibernation]")
	}
	if oldMatches := getNames(t, db, getFirstVault(), "Torpor"); len(oldMatches) != 0 {
		t.Errorf("the name it left still answers with %+v", oldMatches)
	}
}
