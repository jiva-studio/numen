package index

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// under is the paths one vault holds at a path and beneath it.
func under(t *testing.T, db *DB, vault domain.Vault, path string) []string {
	t.Helper()

	found, err := db.Sources().Under(t.Context(), vault.ID, path)
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
func TestWhatIsUnderAFolderIsEverythingItHolds(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "physics/Entropy.md", "Entropy")
	noted(t, db, first, "physics/heat/Heat.md", "Heat")
	noted(t, db, first, "physics-old/Stray.md", "Stray")
	noted(t, db, first, "Outside.md", "Outside")
	book(t, db, first, "physics/A Book.epub", 1)

	want := []string{"physics/A Book.epub", "physics/Entropy.md", "physics/heat/Heat.md"}
	if got := under(t, db, first, "physics"); !slices.Equal(got, want) {
		t.Errorf("the folder holds %v, want %v", got, want)
	}
}

// A file is what is at its own path, and holds nothing.
func TestWhatIsUnderAFileIsTheFile(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "physics/Entropy.md", "Entropy")
	noted(t, db, first, "physics/Entropy.md.bak.md", "A copy")

	want := []string{"physics/Entropy.md"}
	if got := under(t, db, first, "physics/Entropy.md"); !slices.Equal(got, want) {
		t.Errorf("the path holds %v, want %v", got, want)
	}
}

// What a source is, as well as where it is: a caller acts on the difference
// between a note and a book.
func TestWhatIsUnderAPathSaysWhichKindEachSourceIs(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "physics/Entropy.md", "Entropy")
	book(t, db, first, "physics/A Book.epub", 1)

	held := map[string]domain.SourceKind{}
	found, err := db.Sources().Under(t.Context(), first.ID, "physics")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range found {
		held[ref.Path] = ref.Kind
	}
	if kind := held["physics/Entropy.md"]; kind != domain.KindNote {
		t.Errorf("the note came back as %q", kind)
	}
	if kind := held["physics/A Book.epub"]; kind != domain.KindBook {
		t.Errorf("the book came back as %q", kind)
	}
}

// One database holds every vault, and a path means something inside one of
// them. Both vaults file something under the same folder name.
func TestWhatIsUnderAPathIsOneVaultsAlone(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "physics/Entropy.md", "Entropy")
	noted(t, db, second, "physics/Quasar.md", "Quasar")

	if got := under(t, db, first, "physics"); !slices.Equal(got, []string{"physics/Entropy.md"}) {
		t.Errorf("the first vault holds %v", got)
	}
	if got := under(t, db, second, "physics"); !slices.Equal(got, []string{"physics/Quasar.md"}) {
		t.Errorf("the second vault holds %v", got)
	}
}
