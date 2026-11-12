package index

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func getPathsUnder(t *testing.T, db *DB, vault domain.Vault, path string) []string {
	t.Helper()

	found, err := db.Sources().GetSourcesUnder(t.Context(), vault.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(found))
	for _, ref := range found {
		out = append(out, ref.Path)
	}
	return out
}

// A folder holds what its own path prefixes, and a sibling whose name begins
// with the folder's is not under it.
func TestWhatIsBelowAFolderIsEverythingItContains(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "physics/Entropy.md", "Entropy")
	saveNamedNote(t, db, getFirstVault(), "physics/heat/Heat.md", "Heat")
	saveNamedNote(t, db, getFirstVault(), "physics-old/Stray.md", "Stray")
	saveNamedNote(t, db, getFirstVault(), "Outside.md", "Outside")
	book(t, db, getFirstVault(), "physics/A Book.epub", 1)

	want := []string{"physics/A Book.epub", "physics/Entropy.md", "physics/heat/Heat.md"}
	if got := getPathsUnder(t, db, getFirstVault(), "physics"); !slices.Equal(got, want) {
		t.Errorf("the folder holds %v, want %v", got, want)
	}
}

// A file is what is at its own path, and holds nothing.
func TestWhatIsBelowAFileIsTheFile(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "physics/Entropy.md", "Entropy")
	saveNamedNote(t, db, getFirstVault(), "physics/Entropy.md.bak.md", "A copy")

	want := []string{"physics/Entropy.md"}
	if got := getPathsUnder(t, db, getFirstVault(), "physics/Entropy.md"); !slices.Equal(got, want) {
		t.Errorf("the path holds %v, want %v", got, want)
	}
}

// What a source is, as well as where it is: a caller acts on the difference
// between a note and a book.
func TestWhatIsBelowAPathReturnsWhichKindEachSourceIs(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "physics/Entropy.md", "Entropy")
	book(t, db, getFirstVault(), "physics/A Book.epub", 1)

	target := map[string]domain.SourceKind{}
	found, err := db.Sources().GetSourcesUnder(t.Context(), getFirstVault().ID, "physics")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range found {
		target[ref.Path] = ref.Kind
	}
	if kind := target["physics/Entropy.md"]; kind != domain.KindNote {
		t.Errorf("the note came back as %q", kind)
	}
	if kind := target["physics/A Book.epub"]; kind != "book" {
		t.Errorf("the book came back as %q", kind)
	}
}

// One database holds every vault, and a path means something inside one of
// them. Both vaults file something under the same folder name.
func TestWhatIsBelowAPathIsOneVaultsAlone(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "physics/Entropy.md", "Entropy")
	saveNamedNote(t, db, getSecond(), "physics/Quasar.md", "Quasar")

	if got := getPathsUnder(t, db, getFirstVault(), "physics"); !slices.Equal(got, []string{"physics/Entropy.md"}) {
		t.Errorf("the first vault holds %v", got)
	}
	if got := getPathsUnder(t, db, getSecond(), "physics"); !slices.Equal(got, []string{"physics/Quasar.md"}) {
		t.Errorf("the second vault holds %v", got)
	}
}
