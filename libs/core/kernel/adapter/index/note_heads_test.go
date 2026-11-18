package index

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestTheHeadOfANoteIsItsTypeAndItsFrontmatterWithWholeNumbersWhole(t *testing.T) {
	db := openDB(t)
	err := saveCut(t.Context(), db, getFirstVault().ID, []domain.Note{
		{
			FileStat:    domain.FileStat{Path: "a.md", Kind: domain.KindNote, Size: 1, ModTime: getWalkedTime()},
			Type:        "deck",
			Frontmatter: map[string]any{"rank": 3, "ratio": 2.5, "tags": []any{"x", 4}},
		},
		{FileStat: domain.FileStat{Path: "plain.md", Kind: domain.KindNote, Size: 1, ModTime: getWalkedTime()}, Type: domain.TypeNote},
	})
	if err != nil {
		t.Fatal(err)
	}

	heads, err := db.NoteQueries().GetHeads(t.Context(), getFirstVault().ID, []string{"a.md", "plain.md", "missing.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(heads) != 2 {
		t.Fatalf("heads of %v, want a.md and plain.md", heads)
	}
	deck := heads["a.md"]
	if deck.Type != "deck" || deck.Values["rank"] != 3 || deck.Values["ratio"] != 2.5 {
		t.Errorf("the deck is %+v", deck)
	}
	if tags, ok := deck.Values["tags"].([]any); !ok || len(tags) != 2 || tags[1] != 4 {
		t.Errorf("the tags are %v", deck.Values["tags"])
	}
	if plain := heads["plain.md"]; plain.Type != domain.TypeNote || plain.Values != nil {
		t.Errorf("the plain note is %+v, want no values", plain)
	}
}
