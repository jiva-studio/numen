package index

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// chunksOf is the rows the chunks of one source sit on, in order. A source
// filed somewhere else keeps them, and the vectors addressed by their text with
// them.
func chunksOf(t *testing.T, db *DB, vault domain.Vault, path string) []int64 {
	t.Helper()

	rows, err := db.read.QueryContext(t.Context(),
		`SELECT c.id FROM chunks c JOIN sources s ON s.id = c.source_id
		 WHERE s.vault_id = (SELECT id FROM vaults WHERE identifier = ?) AND s.path = ?
		 ORDER BY c.id`, vault.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// A folder is filed somewhere else with everything under it, whatever kind of
// file that is, and what was made from each source stays on the source.
func TestAMovedFolderTakesEverythingUnderIt(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "physics/Entropy.md", "Entropy")
	noted(t, db, first, "physics/heat/Heat.md", "Heat")
	noted(t, db, first, "physics-old/Stray.md", "Stray")
	book(t, db, first, "physics/A Book.epub", 1)
	held := chunksOf(t, db, first, "physics/Entropy.md")
	if len(held) == 0 {
		t.Fatal("the note was indexed with no chunks, and this test asks what happens to them")
	}

	if err := db.Sources().MoveSources(t.Context(), first.ID, "physics", "science/physics"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"science/physics/A Book.epub",
		"science/physics/Entropy.md",
		"science/physics/heat/Heat.md",
	}
	if got := under(t, db, first, "science/physics"); !slices.Equal(got, want) {
		t.Errorf("the folder now holds %v, want %v", got, want)
	}
	if got := under(t, db, first, "physics"); len(got) != 0 {
		t.Errorf("the folder it left still holds %v", got)
	}
	if got := under(t, db, first, "physics-old"); !slices.Equal(got, []string{"physics-old/Stray.md"}) {
		t.Errorf("a folder whose name begins with the one that moved travelled: %v", got)
	}
	if got := chunksOf(t, db, first, "science/physics/Entropy.md"); !slices.Equal(got, held) {
		t.Errorf("the note's chunks are %v, and were %v", got, held)
	}
}

// One database holds every vault, and a folder of one name moves in the vault
// it was named in.
func TestAMovedFolderMovesInOneVaultAlone(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "physics/Entropy.md", "Entropy")
	noted(t, db, second, "physics/Quasar.md", "Quasar")

	if err := db.Sources().MoveSources(t.Context(), first.ID, "physics", "science"); err != nil {
		t.Fatal(err)
	}

	if got := under(t, db, second, "physics"); !slices.Equal(got, []string{"physics/Quasar.md"}) {
		t.Errorf("the other vault's folder holds %v", got)
	}
	if got := under(t, db, second, "science"); len(got) != 0 {
		t.Errorf("the other vault was given %v", got)
	}
}

// A path is bytes on disk and characters in the index, and a folder whose name
// is not Latin holds the two apart.
func TestAMovedFolderCarriesANameThatIsNotLatin(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "физика/Энтропия.md", "Энтропия")

	if err := db.Sources().MoveSources(t.Context(), first.ID, "физика", "physics"); err != nil {
		t.Fatal(err)
	}

	want := []string{"physics/Энтропия.md"}
	if got := under(t, db, first, "physics"); !slices.Equal(got, want) {
		t.Errorf("the folder now holds %v, want %v", got, want)
	}
}
