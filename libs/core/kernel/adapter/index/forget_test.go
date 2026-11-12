package index

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// sharedText is held word for word by a source of each vault, so one row of
// `vectors` is addressed by chunks of both.
const sharedText = "a passage that both vaults hold word for word"

func fillVault(t *testing.T, db *DB, vault domain.Vault, stem string, seed byte) {
	t.Helper()
	ctx := t.Context()

	body := stem + " heading\n" + stem + " body"
	n := domain.Note{
		FileStat: domain.FileStat{Path: "notes/" + stem + ".md", Kind: domain.KindNote, Size: int64(len(body)), ModTime: getWalkedTime()},
		Title:    stem + " title",
		Body:     body,
		Headings: []domain.Heading{{Level: 2, Text: stem + " heading", Line: 0, Offset: 0}},
		Links:    []domain.Link{{Target: domain.Address{Scheme: domain.SchemeName, Value: stem + " elsewhere"}, Role: domain.RoleRef}},
		Problems: []string{stem + " problem"},
	}
	if err := saveCut(ctx, db, vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}

	book(t, db, vault, "library/"+stem+".epub", seed)

	path := "library/" + stem + "-sections.pdf"
	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 1000, MTime: 1, ContentHash: "hash-" + path, Recipe: "pdf",
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
	saveSeededVectors(t, db, vault, seed)
}

// contents is what one vault holds, as the numbers its rows in the virtual
// tables are addressed by.
type contents struct {
	sources  []int64
	notes    []int64
	headings []int64
	chunks   []int64
}

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
	defer func() {
		if err := found.Close(); err != nil {
			t.Error(err)
		}
	}()

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

func list(rows []int64) string {
	content := make([]string, len(rows))
	for i, row := range rows {
		content[i] = strconv.FormatInt(row, 10)
	}
	return "(" + strings.Join(content, ",") + ")"
}

func TestForgettingAVaultEmptiesTheVirtualTablesOfIt(t *testing.T) {
	// Nothing cascades into a virtual table. A row remaining in one of them answers a
	// search with a passage of a vault the index does not contain, and the row it
	// references is gone, so nothing else reports it either.
	ctx := t.Context()
	db := openDB(t)
	fillVault(t, db, getFirstVault(), "first", 0x0f)
	fillVault(t, db, getSecond(), "second", 0xf0)

	removed := getVaultContents(t, db, getFirstVault())
	remainingContent := getVaultContents(t, db, getSecond())
	goneVirtual, goneOrdinary := virtual(t, db, removed), ordinary(t, db, removed)
	keptVirtual, keptOrdinary := virtual(t, db, remainingContent), ordinary(t, db, remainingContent)

	vectors := countRows(t, db, `SELECT COUNT(*) FROM vectors`)
	bought := numbers(t, db, `SELECT rowid FROM vectors WHERE hash IN
		(SELECT unhex(hash) FROM chunks WHERE id IN `+list(removed.chunks)+`)`)
	if len(bought) == 0 {
		t.Fatal("the vault that is forgotten paid for no vector, so this test would pass either way")
	}
	if shared := countRows(t, db, `SELECT COUNT(*) FROM
		(SELECT hash FROM chunks GROUP BY hash HAVING COUNT(DISTINCT vault_id) > 1)`); shared == 0 {
		t.Fatal("vaults do not share vectors for distinct text")
	}

	for _, before := range []map[string]int{goneVirtual, goneOrdinary} {
		for table, n := range before {
			if n == 0 {
				t.Fatalf("%s held nothing of the first vault to begin with, so this test would pass either way", table)
			}
		}
	}

	if err := db.Vaults().Unregister(ctx, getFirstVault().ID); err != nil {
		t.Fatal(err)
	}

	for table, n := range virtual(t, db, removed) {
		if n != 0 {
			t.Errorf("%s holds %d rows of the vault that was forgotten", table, n)
		}
	}
	for table, n := range ordinary(t, db, removed) {
		if n != 0 {
			t.Errorf("%s holds %d rows of the vault that was forgotten", table, n)
		}
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM vaults WHERE identifier = ?`, getFirstVault().ID); got != 0 {
		t.Errorf("the forgotten vault still has %d rows of its own", got)
	}

	for table, n := range virtual(t, db, remainingContent) {
		if n != keptVirtual[table] {
			t.Errorf("%s holds %d rows of the other vault, and held %d", table, n, keptVirtual[table])
		}
	}
	for table, n := range ordinary(t, db, remainingContent) {
		if n != keptOrdinary[table] {
			t.Errorf("%s holds %d rows of the other vault, and held %d", table, n, keptOrdinary[table])
		}
	}
	if now := getVaultContents(t, db, getSecond()); !slices.Equal(now.chunks, remainingContent.chunks) ||
		!slices.Equal(now.sources, remainingContent.sources) ||
		!slices.Equal(now.notes, remainingContent.notes) ||
		!slices.Equal(now.headings, remainingContent.headings) {
		t.Errorf("the second vault holds %+v, and held %+v", now, remainingContent)
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

func TestTheVaultThatIsCachedStillAnswers(t *testing.T) {
	// Every search runs over one table holding every vault. What the forgotten
	// vault was filed under is what the other vault's rows are read back by.
	ctx := t.Context()
	db := openDB(t)
	fillVault(t, db, getFirstVault(), "first", 0x0f)
	fillVault(t, db, getSecond(), "second", 0xf0)

	if err := db.Vaults().Unregister(ctx, getFirstVault().ID); err != nil {
		t.Fatal(err)
	}

	queries := queriesOf(db)
	lexical, err := queries.FindLexical(ctx, getSecond().ID, sourcedomain.LexicalQuery{Text: "second", Kinds: nil, Limit: 10})
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

	sections, err := queries.GetNamedPassages(ctx, getSecond().ID, sourcedomain.LexicalQuery{Text: "second section", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) == 0 {
		t.Error("the vault that was kept answers with none of its sections")
	}

	names, err := db.NoteQueries().GetNameMatches(ctx, getSecond().ID, "second", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Error("the vault that was kept answers with none of its names")
	}
}

func TestForgettingAVaultTheIndexDoesNotHold(t *testing.T) {
	db := openDB(t)

	if err := db.Vaults().Unregister(t.Context(), "01NOTHING"); err != nil {
		t.Errorf("forgetting a vault the index never held: %v", err)
	}
}
