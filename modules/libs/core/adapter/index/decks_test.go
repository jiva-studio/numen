package index

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// laid puts one note in, of the type its file says it is, divided by the
// headings given. The body is the headings written out, so a note that is cut
// has words to cut.
func laid(
	t *testing.T,
	db *DB,
	vault domain.Vault,
	path string,
	kind domain.NoteType,
	headings ...domain.Heading,
) {
	t.Helper()

	body := ""
	for i := range headings {
		headings[i].Line = i
		headings[i].Offset = len(body)
		body += headings[i].Text + "\n"
	}
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: int64(len(body)), ModTime: walked},
		Title:       path,
		Type:        kind,
		Body:        body,
		Headings:    headings,
	}
	if err := db.Notes().Save(t.Context(), vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

// chunksOfNote is how many chunks one note of one vault holds, of both sizes.
func chunksOfNote(t *testing.T, db *DB, vault domain.Vault, path string) int {
	t.Helper()
	return counted(t, db, `SELECT COUNT(*) FROM chunks c
	                       JOIN sources s ON s.id = c.source_id
	                       JOIN vaults v ON v.id = c.vault_id
	                       WHERE v.identifier = ? AND s.path = ?`, vault.ID, path)
}

// headingTexts is what the index holds as the outline of one note.
func headingTexts(t *testing.T, db *DB, vault domain.Vault, path string) []string {
	t.Helper()

	found := divisions(t, db, vault, path)[path]
	out := make([]string, 0, len(found))
	for _, h := range found {
		out = append(out, h.Text)
	}
	return out
}

// A deck's body is a few hundred fragments a person wrote to be recalled one at
// a time, and nothing searches inside a card.
func TestADeckContributesNoChunkAndNoVector(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "decks/mammals.md", domain.TypeDeck,
		domain.Heading{Level: 1, Text: "Roots"},
		domain.Heading{Level: 2, Text: "Compost, what is it made of ^k7m2xq9fzp"},
		domain.Heading{Level: 3, Text: "Question"},
		domain.Heading{Level: 3, Text: "Answer"},
	)

	if got := chunksOfNote(t, db, first, "decks/mammals.md"); got != 0 {
		t.Errorf("a deck holds %d chunks, want none", got)
	}
	owing, err := db.ChunkQueries().Unembedded(t.Context(), first.ID, "model", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 0 {
		t.Errorf("a deck was offered for embedding: %v", owing)
	}
}

// A stencil holds templates full of {{Field}}, and is found by its title.
func TestAStencilContributesNoChunkAndNoVector(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "stencils/term.md", domain.TypeStencil,
		domain.Heading{Level: 1, Text: "Question"},
		domain.Heading{Level: 2, Text: "Front"},
	)

	if got := chunksOfNote(t, db, first, "stencils/term.md"); got != 0 {
		t.Errorf("a stencil holds %d chunks, want none", got)
	}
	owing, err := db.ChunkQueries().Unembedded(t.Context(), first.ID, "model", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) != 0 {
		t.Errorf("a stencil was offered for embedding: %v", owing)
	}
}

// A note beside them is cut as it always was.
func TestAnOrdinaryNoteIsStillCut(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "notes/entropy.md", domain.TypeNote,
		domain.Heading{Level: 1, Text: "What it is"},
		domain.Heading{Level: 2, Text: "Where it came from"},
	)

	if got := chunksOfNote(t, db, first, "notes/entropy.md"); got == 0 {
		t.Error("an ordinary note holds no chunk")
	}
}

// A note that is saved again as a deck loses the chunks it held as a note.
func TestANoteThatBecomesADeckLosesItsChunks(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "decks/mammals.md", domain.TypeNote,
		domain.Heading{Level: 1, Text: "Roots"},
	)
	if got := chunksOfNote(t, db, first, "decks/mammals.md"); got == 0 {
		t.Fatal("the note held no chunk to begin with")
	}

	laid(t, db, first, "decks/mammals.md", domain.TypeDeck,
		domain.Heading{Level: 1, Text: "Roots"},
	)
	if got := chunksOfNote(t, db, first, "decks/mammals.md"); got != 0 {
		t.Errorf("a note read again as a deck holds %d chunks, want none", got)
	}
}

// A deck keeps its sections and its cards. Its third level is the stencil's
// field names written out under every card.
func TestADeckKeepsTheHeadingsAPersonWroteAndNoOthers(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "decks/mammals.md", domain.TypeDeck,
		domain.Heading{Level: 1, Text: "Roots"},
		domain.Heading{Level: 2, Text: "Compost, what is it made of ^k7m2xq9fzp"},
		domain.Heading{Level: 3, Text: "Question"},
		domain.Heading{Level: 3, Text: "Answer"},
		domain.Heading{Level: 2, Text: "Leaf mould, how long ^0123456789"},
		domain.Heading{Level: 3, Text: "Question"},
		domain.Heading{Level: 3, Text: "Answer"},
	)

	got := headingTexts(t, db, first, "decks/mammals.md")
	want := []string{"Roots", "Compost, what is it made of", "Leaf mould, how long"}
	if len(got) != len(want) {
		t.Fatalf("the deck holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
	// The name search is over the same rows, and a field name standing in it
	// once per card is what crowds it.
	if n := counted(t, db, `SELECT COUNT(*) FROM headings_fts WHERE headings_fts MATCH 'Answer'`); n != 0 {
		t.Errorf("the name search holds %d field names of a deck", n)
	}
}

// A stencil's headings are its faces and their two sides.
func TestAStencilContributesNoHeading(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "stencils/term.md", domain.TypeStencil,
		domain.Heading{Level: 1, Text: "Question"},
		domain.Heading{Level: 1, Text: "Recognise"},
		domain.Heading{Level: 2, Text: "Front"},
		domain.Heading{Level: 2, Text: "Back"},
	)

	if got := headingTexts(t, db, first, "stencils/term.md"); len(got) != 0 {
		t.Errorf("a stencil holds the headings %v, want none", got)
	}
	if n := counted(t, db, `SELECT COUNT(*) FROM headings_fts WHERE headings_fts MATCH 'Front'`); n != 0 {
		t.Errorf("the name search holds %d headings of a stencil", n)
	}
}

// The mark is written for the file, not for a person reading a list.
func TestACardsHeadingIsStoredWithoutItsMark(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "decks/mammals.md", domain.TypeDeck,
		// A mark, and four endings that are not one: too short, too long, an
		// alphabet the mark does not use, and no space before the caret.
		domain.Heading{Level: 2, Text: "Compost, what is it made of ^k7m2xq9fzp"},
		domain.Heading{Level: 2, Text: "Short ^k7m2xq9f"},
		domain.Heading{Level: 2, Text: "Long ^k7m2xq9fzpb"},
		domain.Heading{Level: 2, Text: "Letters ^k7m2xq9flu"},
		domain.Heading{Level: 2, Text: "Joined^k7m2xq9fzp"},
	)

	got := headingTexts(t, db, first, "decks/mammals.md")
	want := []string{
		"Compost, what is it made of",
		"Short ^k7m2xq9f",
		"Long ^k7m2xq9fzpb",
		"Letters ^k7m2xq9flu",
		"Joined^k7m2xq9fzp",
	}
	if len(got) != len(want) {
		t.Fatalf("the deck holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
	if n := counted(t, db,
		`SELECT COUNT(*) FROM headings_fts WHERE headings_fts MATCH 'k7m2xq9fzp'`); n != 1 {
		t.Errorf("%d headings are searched by a mark, want the one that is heading text", n)
	}
}

// A card whose first field is empty is named by nothing at all, and a name is
// what a heading is kept for.
func TestACardOfAnEmptyFirstFieldContributesNoHeading(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "decks/mammals.md", domain.TypeDeck,
		domain.Heading{Level: 1, Text: "Roots"},
		domain.Heading{Level: 2, Text: "^k7m2xq9fzp"},
		domain.Heading{Level: 3, Text: "Question"},
		domain.Heading{Level: 2, Text: "Leaf mould, how long ^0123456789"},
	)

	got := headingTexts(t, db, first, "decks/mammals.md")
	want := []string{"Roots", "Leaf mould, how long"}
	if len(got) != len(want) {
		t.Fatalf("the deck holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
	if n := counted(t, db,
		`SELECT COUNT(*) FROM headings h JOIN sources s ON s.id = h.note_id
		 WHERE s.path = 'decks/mammals.md' AND h.text = ''`); n != 0 {
		t.Errorf("the deck holds %d headings of no name", n)
	}
}

// Every level of an ordinary note is kept, and an ending that reads as a mark
// in a deck is heading text in a note.
func TestAnOrdinaryNoteKeepsEveryHeadingItHas(t *testing.T) {
	db := opened(t)
	laid(t, db, first, "notes/entropy.md", domain.TypeNote,
		domain.Heading{Level: 1, Text: "What it is"},
		domain.Heading{Level: 2, Text: "Where it came from"},
		domain.Heading{Level: 3, Text: "Answer"},
		domain.Heading{Level: 3, Text: "An anchor ^k7m2xq9fzp"},
	)

	got := headingTexts(t, db, first, "notes/entropy.md")
	want := []string{"What it is", "Where it came from", "Answer", "An anchor ^k7m2xq9fzp"}
	if len(got) != len(want) {
		t.Fatalf("the note holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
}

// Everything else about a deck is what it was: a row, a title, a type, an
// identifier, its links and its node in the plex.
func TestADeckIsANoteInEveryOtherWay(t *testing.T) {
	db := opened(t)
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "decks/mammals.md", Kind: domain.KindNote, Size: 100, ModTime: walked},
		Title:       "Mammals",
		Type:        domain.TypeDeck,
		ID:          "01HQXMAMMALS",
		Body:        "## Compost ^k7m2xq9fzp\n",
		Links: []domain.Link{{
			Target: domain.Address{Scheme: "note", Value: "stencils/Term"},
			Role:   domain.RoleRef,
		}},
		Headings: []domain.Heading{{Level: 2, Text: "Compost ^k7m2xq9fzp"}},
	}
	if err := db.Notes().Save(t.Context(), first.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}

	held, err := db.NoteQueries().Types(t.Context(), first.ID, []string{"decks/mammals.md"})
	if err != nil {
		t.Fatal(err)
	}
	if held["decks/mammals.md"] != domain.TypeDeck {
		t.Errorf("the index calls it %q, want %q", held["decks/mammals.md"], domain.TypeDeck)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM notes n JOIN vaults v ON v.id = n.vault_id
	                          WHERE v.identifier = ? AND n.title = 'Mammals'
	                            AND n.identifier = '01HQXMAMMALS'`, first.ID); got != 1 {
		t.Error("the deck's row does not carry its title and its identifier")
	}
	links, err := db.NoteQueries().Links(t.Context(), first.ID, "decks/mammals.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 {
		t.Errorf("the deck's links are %v, want the one it was saved with", links)
	}
}
