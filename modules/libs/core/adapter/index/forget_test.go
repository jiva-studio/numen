package index

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// sharedText is held word for word by a source of each vault, so one row of
// `vectors` is addressed by chunks of both.
const sharedText = "a passage that both vaults hold word for word"

// fillVault puts a row in every table the index has, for one vault: a note with
// a heading, a link and a problem, a book cut into chunks and embedded, and a
// book whose chunks open named sections.
//
// Every text carries the stem, so a row that arrives from the wrong vault is
// recognisable. The one exception is the shared passage.
func fillVault(t *testing.T, db *DB, vault domain.Vault, stem string, seed byte) {
	t.Helper()
	ctx := t.Context()

	body := stem + " heading\n" + stem + " body"
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/" + stem + ".md", Kind: domain.KindNote, Size: int64(len(body)), ModTime: walked},
		Title:       stem + " title",
		Body:        body,
		Headings:    []domain.Heading{{Level: 2, Text: stem + " heading", Line: 0, Offset: 0}},
		Links:       []domain.Link{{Target: domain.Address{Scheme: domain.SchemeName, Value: stem + " elsewhere"}, Role: domain.RoleRef}},
		Problems:    []string{stem + " problem"},
	}
	if err := db.Notes().Save(ctx, vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}

	book(t, db, vault, "library/"+stem+".epub", seed)

	path := "library/" + stem + "-sections.pdf"
	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, Hash: "hash-" + path, Recipe: "pdf",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault.ID, "book", path, []chunk.Chunk{
		{
			Start: 0, Length: 100, Location: stem + " section",
			Opens: []string{stem + " section"},
			Text:  stem + " section opens here",
			Small: []chunk.Chunk{{Start: 0, Length: 100, Text: stem + " section opens here"}},
		},
		{
			Start: 100, Length: 100, Location: "the shared passage",
			Text:  sharedText,
			Small: []chunk.Chunk{{Start: 100, Length: 100, Text: sharedText}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	vectorise(t, db, vault, seed)
}

// contents is what one vault holds, as the numbers its rows in the virtual
// tables are addressed by.
type contents struct {
	sources  []int64
	notes    []int64
	headings []int64
	chunks   []int64
}

// getVaultContents reads what a vault holds. A vault holding none of the four
// fails here.
func getVaultContents(t *testing.T, db *DB, vault domain.Vault) contents {
	t.Helper()

	c := contents{
		sources: numbers(t, db, `SELECT s.id FROM sources s
			JOIN vaults v ON v.id = s.vault_id WHERE v.identifier = ?`, vault.ID),
		notes: numbers(t, db, `SELECT n.source_id FROM notes n
			JOIN vaults v ON v.id = n.vault_id WHERE v.identifier = ?`, vault.ID),
		headings: numbers(t, db, `SELECT h.id FROM headings h
			JOIN notes n ON n.source_id = h.note_id
			JOIN vaults v ON v.id = n.vault_id WHERE v.identifier = ?`, vault.ID),
		chunks: numbers(t, db, `SELECT c.id FROM chunks c
			JOIN vaults v ON v.id = c.vault_id WHERE v.identifier = ?`, vault.ID),
	}
	if len(c.sources) == 0 || len(c.notes) == 0 || len(c.headings) == 0 || len(c.chunks) == 0 {
		t.Fatalf("%s holds %d sources, %d notes, %d headings, %d chunks",
			vault.Name, len(c.sources), len(c.notes), len(c.headings), len(c.chunks))
	}
	return c
}

// virtual is how many rows the five virtual tables hold for what a vault held,
// keyed by the table's name so a failure says which one kept them.
func virtual(t *testing.T, db *DB, c contents) map[string]int {
	t.Helper()

	return map[string]int{
		"chunks_vec":   countRows(t, db, `SELECT COUNT(*) FROM chunks_vec WHERE chunk_id IN `+list(c.chunks)),
		"chunks_fts":   countRows(t, db, `SELECT COUNT(*) FROM chunks_fts WHERE rowid IN `+list(c.chunks)),
		"sections_fts": countRows(t, db, `SELECT COUNT(*) FROM sections_fts WHERE rowid IN `+list(c.chunks)),
		"titles_fts":   countRows(t, db, `SELECT COUNT(*) FROM titles_fts WHERE rowid IN `+list(c.notes)),
		"headings_fts": countRows(t, db, `SELECT COUNT(*) FROM headings_fts WHERE rowid IN `+list(c.headings)),
	}
}

// ordinary is how many rows the cascading tables hold for what a vault held.
func ordinary(t *testing.T, db *DB, c contents) map[string]int {
	t.Helper()

	return map[string]int{
		"sources":  countRows(t, db, `SELECT COUNT(*) FROM sources WHERE id IN `+list(c.sources)),
		"notes":    countRows(t, db, `SELECT COUNT(*) FROM notes WHERE source_id IN `+list(c.notes)),
		"headings": countRows(t, db, `SELECT COUNT(*) FROM headings WHERE id IN `+list(c.headings)),
		"links":    countRows(t, db, `SELECT COUNT(*) FROM links WHERE note_id IN `+list(c.notes)),
		"problems": countRows(t, db, `SELECT COUNT(*) FROM problems WHERE note_id IN `+list(c.notes)),
		"chunks":   countRows(t, db, `SELECT COUNT(*) FROM chunks WHERE id IN `+list(c.chunks)),
	}
}

func numbers(t *testing.T, db *DB, statement string, args ...any) []int64 {
	t.Helper()

	found, err := db.read.QueryContext(t.Context(), statement, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer found.Close()

	var out []int64
	for found.Next() {
		var row int64
		if err := found.Scan(&row); err != nil {
			t.Fatal(err)
		}
		out = append(out, row)
	}
	if err := found.Err(); err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// list writes row numbers as an SQL list, for a question asked of rows that the
// answer is allowed to have taken away.
func list(rows []int64) string {
	written := make([]string, len(rows))
	for i, row := range rows {
		written[i] = strconv.FormatInt(row, 10)
	}
	return "(" + strings.Join(written, ",") + ")"
}

func TestForgettingAVaultEmptiesTheVirtualTablesOfIt(t *testing.T) {
	// Nothing cascades into a virtual table. A row left in one of them answers a
	// search with a passage of a vault the index no longer holds, and the row it
	// names is gone, so nothing else says so either.
	ctx := t.Context()
	db := openDB(t)
	fillVault(t, db, first, "first", 0x0f)
	fillVault(t, db, second, "second", 0xf0)

	gone := getVaultContents(t, db, first)
	kept := getVaultContents(t, db, second)
	goneVirtual, goneOrdinary := virtual(t, db, gone), ordinary(t, db, gone)
	keptVirtual, keptOrdinary := virtual(t, db, kept), ordinary(t, db, kept)

	vectors := countRows(t, db, `SELECT COUNT(*) FROM vectors`)
	bought := numbers(t, db, `SELECT rowid FROM vectors WHERE hash IN
		(SELECT unhex(hash) FROM chunks WHERE id IN `+list(gone.chunks)+`)`)
	if len(bought) == 0 {
		t.Fatal("the vault that is forgotten paid for no vector, so this test would pass either way")
	}
	if shared := countRows(t, db, `SELECT COUNT(*) FROM
		(SELECT hash FROM chunks GROUP BY hash HAVING COUNT(DISTINCT vault_id) > 1)`); shared == 0 {
		t.Fatal("no text is held by both vaults, so no vector here is one the other vault still holds")
	}

	for _, before := range []map[string]int{goneVirtual, goneOrdinary} {
		for table, n := range before {
			if n == 0 {
				t.Fatalf("%s held nothing of the first vault to begin with, so this test would pass either way", table)
			}
		}
	}

	if err := db.Vaults().Forget(ctx, first.ID); err != nil {
		t.Fatal(err)
	}

	for table, n := range virtual(t, db, gone) {
		if n != 0 {
			t.Errorf("%s holds %d rows of the vault that was forgotten", table, n)
		}
	}
	for table, n := range ordinary(t, db, gone) {
		if n != 0 {
			t.Errorf("%s holds %d rows of the vault that was forgotten", table, n)
		}
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM vaults WHERE identifier = ?`, first.ID); got != 0 {
		t.Errorf("the forgotten vault still has %d rows of its own", got)
	}

	for table, n := range virtual(t, db, kept) {
		if n != keptVirtual[table] {
			t.Errorf("%s holds %d rows of the other vault, and held %d", table, n, keptVirtual[table])
		}
	}
	for table, n := range ordinary(t, db, kept) {
		if n != keptOrdinary[table] {
			t.Errorf("%s holds %d rows of the other vault, and held %d", table, n, keptOrdinary[table])
		}
	}
	if now := getVaultContents(t, db, second); !slices.Equal(now.chunks, kept.chunks) ||
		!slices.Equal(now.sources, kept.sources) ||
		!slices.Equal(now.notes, kept.notes) ||
		!slices.Equal(now.headings, kept.headings) {
		t.Errorf("the second vault holds %+v, and held %+v", now, kept)
	}

	// A vector is addressed by the text it was made from, so chunks of several
	// vaults hold one, and it is bought work.
	if got := countRows(t, db, `SELECT COUNT(*) FROM vectors WHERE rowid IN `+list(bought)); got != len(bought) {
		t.Errorf("%d of the %d vectors the forgotten vault paid for are left", got, len(bought))
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM vectors`); got != vectors {
		t.Errorf("%d vectors, and %d were bought", got, vectors)
	}
}

func TestTheVaultThatIsKeptStillAnswers(t *testing.T) {
	// Every search runs over one table holding every vault. What the forgotten
	// vault was filed under is what the other vault's rows are read back by.
	ctx := t.Context()
	db := openDB(t)
	fillVault(t, db, first, "first", 0x0f)
	fillVault(t, db, second, "second", 0xf0)

	if err := db.Vaults().Forget(ctx, first.ID); err != nil {
		t.Fatal(err)
	}

	queries := db.ChunkQueries()
	lexical, err := queries.Lexical(ctx, second.ID, "second", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lexical) == 0 {
		t.Error("the vault that was kept answers no search by words")
	}
	for _, p := range lexical {
		if !strings.Contains(p.Source, "second") {
			t.Errorf("the second vault answered with %s, which belongs to the vault that was forgotten", p.Source)
		}
	}

	sections, err := queries.Named(ctx, second.ID, "second section", nil, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) == 0 {
		t.Error("the vault that was kept answers with none of its sections")
	}

	names, err := db.NoteQueries().Names(ctx, second.ID, "second", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Error("the vault that was kept answers with none of its names")
	}
}

func TestForgettingAVaultTheIndexDoesNotHold(t *testing.T) {
	db := openDB(t)

	if err := db.Vaults().Forget(t.Context(), "01NOTHING"); err != nil {
		t.Errorf("forgetting a vault the index never held: %v", err)
	}
}
