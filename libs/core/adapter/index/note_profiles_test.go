package index

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// saveTypedNoteWithHeadings stores one note, of the type its file declares,
// divided by the given headings. The body is the headings written out, so a
// note that is cut has text to cut.
func saveTypedNoteWithHeadings(
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
		FileStat: domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(body)), ModTime: walked},
		Title:    path,
		Type:     kind,
		Body:     body,
		Headings: headings,
	}
	if err := saveCut(t.Context(), db, vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

// chunksOfNote is how many chunks one note of one vault holds, of both sizes.
func chunksOfNote(t *testing.T, db *DB, vault domain.Vault, path string) int {
	t.Helper()
	return countRows(t, db, `SELECT COUNT(*) FROM chunks c
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

// A type that its plugin declares as not cut keeps its body out of the chunks.
func TestATypeWithoutChunksContributesNoChunkAndNoVector(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "samples/mammals.md", sampleType,
		domain.Heading{Level: 1, Text: "Roots"},
		domain.Heading{Level: 2, Text: "Compost, what is it made of ^k7m2xq9fzp"},
		domain.Heading{Level: 3, Text: "Summary"},
		domain.Heading{Level: 3, Text: "Source"},
	)

	if got := chunksOfNote(t, db, first, "samples/mammals.md"); got != 0 {
		t.Errorf("a sample holds %d chunks, want none", got)
	}
	unembedded, err := db.ChunkQueries().GetUnembeddedChunks(t.Context(), first.ID, "model", 0, 0, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 0 {
		t.Errorf("a sample was offered for embedding: %v", unembedded)
	}
}

// A type that keeps no heading is not cut either.
func TestATypeKeepingNoHeadingsContributesNoChunkAndNoVector(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "forms/term.md", formType,
		domain.Heading{Level: 1, Text: "Layout"},
		domain.Heading{Level: 2, Text: "Header"},
	)

	if got := chunksOfNote(t, db, first, "forms/term.md"); got != 0 {
		t.Errorf("a form holds %d chunks, want none", got)
	}
	unembedded, err := db.ChunkQueries().GetUnembeddedChunks(t.Context(), first.ID, "model", 0, 0, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) != 0 {
		t.Errorf("a form was offered for embedding: %v", unembedded)
	}
}

// A note of the default type is cut.
func TestAnOrdinaryNoteIsStillSplit(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "notes/entropy.md", domain.TypeNote,
		domain.Heading{Level: 1, Text: "What it is"},
		domain.Heading{Level: 2, Text: "Where it came from"},
	)

	if got := chunksOfNote(t, db, first, "notes/entropy.md"); got == 0 {
		t.Error("an ordinary note holds no chunk")
	}
}

// A note saved again as a type that is not cut loses the chunks it held as a note.
func TestARetypedNoteLosesItsChunks(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "samples/mammals.md", domain.TypeNote,
		domain.Heading{Level: 1, Text: "Roots"},
	)
	if got := chunksOfNote(t, db, first, "samples/mammals.md"); got == 0 {
		t.Fatal("the note held no chunk to begin with")
	}

	saveTypedNoteWithHeadings(t, db, first, "samples/mammals.md", sampleType,
		domain.Heading{Level: 1, Text: "Roots"},
	)
	if got := chunksOfNote(t, db, first, "samples/mammals.md"); got != 0 {
		t.Errorf("a note read again as a sample holds %d chunks, want none", got)
	}
}

// A type keeps the heading levels its profile names and drops the rest. The
// third level here repeats the same names under every second-level heading.
func TestATypeKeepsOnlyItsDeclaredHeadingLevels(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "samples/mammals.md", sampleType,
		domain.Heading{Level: 1, Text: "Roots"},
		domain.Heading{Level: 2, Text: "Compost, what is it made of ^k7m2xq9fzp"},
		domain.Heading{Level: 3, Text: "Summary"},
		domain.Heading{Level: 3, Text: "Source"},
		domain.Heading{Level: 2, Text: "Leaf mould, how long ^0123456789"},
		domain.Heading{Level: 3, Text: "Summary"},
		domain.Heading{Level: 3, Text: "Source"},
	)

	got := headingTexts(t, db, first, "samples/mammals.md")
	want := []string{"Roots", "Compost, what is it made of", "Leaf mould, how long"}
	if len(got) != len(want) {
		t.Fatalf("the sample holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
	// The name search is over the same rows, and a name repeated under every
	// second-level heading is what crowds it.
	if n := countRows(t, db, `SELECT COUNT(*) FROM headings_fts WHERE headings_fts MATCH 'Source'`); n != 0 {
		t.Errorf("the name search holds %d dropped headings of a sample", n)
	}
}

// A type whose profile keeps no heading level contributes no heading.
func TestATypeKeepingNoHeadingsContributesNone(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "forms/term.md", formType,
		domain.Heading{Level: 1, Text: "Layout"},
		domain.Heading{Level: 1, Text: "Variant"},
		domain.Heading{Level: 2, Text: "Header"},
		domain.Heading{Level: 2, Text: "Footer"},
	)

	if got := headingTexts(t, db, first, "forms/term.md"); len(got) != 0 {
		t.Errorf("a form holds the headings %v, want none", got)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM headings_fts WHERE headings_fts MATCH 'Header'`); n != 0 {
		t.Errorf("the name search holds %d headings of a form", n)
	}
}

// The block id at the end of a heading is written for the file, not for a
// person reading a list.
func TestAHeadingIsStoredWithoutItsBlockID(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "samples/mammals.md", sampleType,
		// A mark, and four endings that are not one: too short, too long, an
		// alphabet the id pattern does not take, and no space before the caret.
		domain.Heading{Level: 2, Text: "Compost, what is it made of ^k7m2xq9fzp"},
		domain.Heading{Level: 2, Text: "Short ^k7m2xq9f"},
		domain.Heading{Level: 2, Text: "Long ^k7m2xq9fzpb"},
		domain.Heading{Level: 2, Text: "Letters ^k7m2xq9flu"},
		domain.Heading{Level: 2, Text: "Joined^k7m2xq9fzp"},
	)

	got := headingTexts(t, db, first, "samples/mammals.md")
	want := []string{
		"Compost, what is it made of",
		"Short ^k7m2xq9f",
		"Long ^k7m2xq9fzpb",
		"Letters ^k7m2xq9flu",
		"Joined^k7m2xq9fzp",
	}
	if len(got) != len(want) {
		t.Fatalf("the sample holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM headings_fts WHERE headings_fts MATCH 'k7m2xq9fzp'`); n != 1 {
		t.Errorf("%d headings are searched by a mark, want the one that is heading text", n)
	}
}

// A heading of only a block id has no text, and the index keeps a heading for
// its text.
func TestAHeadingOfOnlyABlockIDContributesNoHeading(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "samples/mammals.md", sampleType,
		domain.Heading{Level: 1, Text: "Roots"},
		domain.Heading{Level: 2, Text: "^k7m2xq9fzp"},
		domain.Heading{Level: 3, Text: "Summary"},
		domain.Heading{Level: 2, Text: "Leaf mould, how long ^0123456789"},
	)

	got := headingTexts(t, db, first, "samples/mammals.md")
	want := []string{"Roots", "Leaf mould, how long"}
	if len(got) != len(want) {
		t.Fatalf("the sample holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM headings h JOIN sources s ON s.id = h.note_id
		 WHERE s.path = 'samples/mammals.md' AND h.text = ''`); n != 0 {
		t.Errorf("the sample holds %d headings of no name", n)
	}
}

// Every level of an ordinary note is kept, and an ending that reads as a mark
// in a sample is heading text in a note.
func TestAnOrdinaryNoteKeepsEveryHeadingItHas(t *testing.T) {
	db := openDB(t)
	saveTypedNoteWithHeadings(t, db, first, "notes/entropy.md", domain.TypeNote,
		domain.Heading{Level: 1, Text: "What it is"},
		domain.Heading{Level: 2, Text: "Where it came from"},
		domain.Heading{Level: 3, Text: "Source"},
		domain.Heading{Level: 3, Text: "An anchor ^k7m2xq9fzp"},
	)

	got := headingTexts(t, db, first, "notes/entropy.md")
	want := []string{"What it is", "Where it came from", "Source", "An anchor ^k7m2xq9fzp"}
	if len(got) != len(want) {
		t.Fatalf("the note holds %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, got[i], want[i])
		}
	}
}

// Everything else about a typed note is what it is for any note: a row, a
// title, a type, an identifier, its links and its node in the plex.
func TestATypedNoteIsANoteInEveryOtherWay(t *testing.T) {
	db := openDB(t)
	n := domain.Note{
		FileStat: domain.FileStat{Path: "samples/mammals.md", Kind: domain.KindNote, Size: 100, ModTime: walked},
		Title:    "Mammals",
		Type:     sampleType,
		ID:       "01HQXMAMMALS",
		Body:     "## Compost ^k7m2xq9fzp\n",
		Links: []domain.Link{{
			Target: domain.Address{Scheme: "note", Value: "forms/Term"},
			Role:   domain.RoleRef,
		}},
		Headings: []domain.Heading{{Level: 2, Text: "Compost ^k7m2xq9fzp"}},
	}
	if err := saveCut(t.Context(), db, first.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}

	target, err := db.NoteQueries().GetTypes(t.Context(), first.ID, []string{"samples/mammals.md"})
	if err != nil {
		t.Fatal(err)
	}
	if target["samples/mammals.md"] != sampleType {
		t.Errorf("the index calls it %q, want %q", target["samples/mammals.md"], sampleType)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM notes n JOIN vaults v ON v.id = n.vault_id
	                          WHERE v.identifier = ? AND n.title = 'Mammals'
	                            AND n.identifier = '01HQXMAMMALS'`, first.ID); got != 1 {
		t.Error("the sample's row does not carry its title and its identifier")
	}
	links, err := db.NoteQueries().GetLinks(t.Context(), first.ID, "samples/mammals.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 {
		t.Errorf("the sample's links are %v, want the one it was saved with", links)
	}
}
