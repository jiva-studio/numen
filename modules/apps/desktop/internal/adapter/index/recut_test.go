package index

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

// A note is cut into one large window over the whole of it, tiled into the small
// windows that carry the vectors, once inside each section the note's headings
// name. These tests are about what a second cut of the same note keeps.

// wordsOf is a body of n words, each one different from every other, so that a
// window is recognisable by what it holds and no two windows of one note hold
// the same text.
func wordsOf(n int) string { return wordsBetween(0, n) }

// wordsBetween is the words from one place in that body to another.
func wordsBetween(from, to int) string {
	out := make([]string, 0, to-from)
	for i := from; i < to; i++ {
		out = append(out, "word"+spelt(i))
	}
	return strings.Join(out, " ")
}

// sectionsOf is a body of n words under a heading every per of them. At 48 words
// a section holds what one small window is cut at, so a section is one window
// and an edit inside it is recognisable.
func sectionsOf(n, per int) string {
	var parts []string
	for at := 0; at < n; at += per {
		parts = append(parts, "## Section"+spelt(at), wordsBetween(at, min(at+per, n)))
	}
	return strings.Join(parts, "\n\n")
}

// typedBefore is the body with a word put in front of the word at that place.
// The words after it keep their spelling and move.
func typedBefore(body string, at int) string {
	word := "word" + spelt(at)
	return strings.Replace(body, word, "wordzz "+word, 1)
}

// spelt writes a number in letters, so that a word is letters throughout and a
// window made of these reads as text.
func spelt(i int) string {
	digits := []byte(strconv.Itoa(i))
	for at, d := range digits {
		digits[at] = 'a' + (d - '0')
	}
	return string(digits)
}

// noteAt is one note as the index holds it, with the body given.
func noteAt(path, title, body string) domain.Note {
	return domain.Note{
		Ref:   domain.FileRef{Path: path, Size: int64(len(body)), MTime: 1},
		Title: title,
		Body:  body,
	}
}

// parsedAt is one note as the parser produces it, so the headings a cut is
// bounded by are the ones the file names and their offsets are the parser's.
func parsedAt(path, body string) domain.Note {
	return markdown.Parse(domain.FileRef{Path: path, Size: int64(len(body)), MTime: 1}, []byte(body))
}

// unplaced is the same note with nothing naming a section in it. A note that
// names no place is one span over its whole body, and its windows tile from the
// first word of the note.
func unplaced(n domain.Note) domain.Note {
	n.Headings = nil
	return n
}

func save(t *testing.T, db *DB, vault domain.Vault, n domain.Note) {
	t.Helper()
	if err := db.Notes().Save(t.Context(), vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

// embedder stands in for the model: it answers what a vault owes, writes a
// vector for each, and says how many it was asked for. That number is what a
// save costs.
type embedder struct{ seed byte }

func (e embedder) run(t *testing.T, db *DB, vault domain.Vault) int {
	t.Helper()
	owing, err := db.ChunkQueries().Unembedded(t.Context(), vault.ID, "model", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	vectorise(t, db, vault, e.seed)
	return len(owing)
}

// smallWindows is the windows of one note that carry a vector, in the order they
// sit in the file. The vault is named as well as the path: one database holds
// every vault, and two vaults here hold a note at the same path.
func smallWindows(t *testing.T, db *DB, vault domain.Vault, path string) []int64 {
	t.Helper()
	return rowsOf(t, db, `SELECT c.id FROM chunks c
	                      JOIN sources s ON s.id = c.source_id
	                      JOIN vaults v ON v.id = c.vault_id
	                      WHERE v.identifier = ? AND s.path = ? AND c.parent IS NOT NULL
	                      ORDER BY c.start, c.length`, vault.ID, path)
}

// largeWindow is the window enclosing a note, which is the note itself.
func largeWindow(t *testing.T, db *DB, vault domain.Vault, path string) int64 {
	t.Helper()
	rows := rowsOf(t, db, `SELECT c.id FROM chunks c
	                       JOIN sources s ON s.id = c.source_id
	                       JOIN vaults v ON v.id = c.vault_id
	                       WHERE v.identifier = ? AND s.path = ? AND c.parent IS NULL`, vault.ID, path)
	if len(rows) != 1 {
		t.Fatalf("%d large windows for %s", len(rows), path)
	}
	return rows[0]
}

// chunksIn is how many chunks one vault holds, of both sizes.
func chunksIn(t *testing.T, db *DB, vault domain.Vault) int {
	t.Helper()
	return counted(t, db, `SELECT COUNT(*) FROM chunks c JOIN vaults v ON v.id = c.vault_id
	                       WHERE v.identifier = ?`, vault.ID)
}

func rowsOf(t *testing.T, db *DB, statement string, args ...any) []int64 {
	t.Helper()
	rows, err := db.read.QueryContext(t.Context(), statement, args...)
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

// vectored says whether one chunk carries a vector, in the table and in the
// coarse index both.
func vectored(t *testing.T, db *DB, chunk int64) bool {
	t.Helper()
	stored := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors WHERE chunk_id = ?`, chunk)
	coarse := counted(t, db, `SELECT COUNT(*) FROM chunks_vec WHERE chunk_id = ?`, chunk)
	if stored != coarse {
		t.Errorf("chunk %d is in %d rows of vectors and %d of the coarse index", chunk, stored, coarse)
	}
	return stored == 1
}

func TestEditingTheEndOfANoteAsksForOneVector(t *testing.T) {
	// A person types a word at the end of a note. The windows before the edit
	// hold the text they held, so they keep their rows and their vectors, and the
	// model is asked for the one window the edit landed in.
	db := opened(t)
	model := embedder{seed: 0x11}

	body := wordsOf(200)
	n := noteAt("notes/Entropy.md", "Entropy", body)
	save(t, db, first, n)

	if asked := model.run(t, db, first); asked != 5 {
		t.Fatalf("a note of 200 words owes %d vectors, want 5", asked)
	}
	was := smallWindows(t, db, first, n.Ref.Path)
	if len(was) != 5 {
		t.Fatalf("%d windows carry a vector", len(was))
	}

	save(t, db, first, noteAt(n.Ref.Path, n.Title, body+" wordzz"))

	now := smallWindows(t, db, first, n.Ref.Path)
	if len(now) != 5 {
		t.Fatalf("%d windows carry a vector after the edit", len(now))
	}
	// Every window before the edit is the window it was, on the row it was on.
	for at, row := range was[:4] {
		if now[at] != row {
			t.Errorf("the window at %d is row %d, and was row %d", at, now[at], row)
		}
		if !vectored(t, db, row) {
			t.Errorf("row %d lost its vector, and the text it holds did not change", row)
		}
	}
	if now[4] == was[4] {
		t.Errorf("the window the edit landed in is row %d, the row of the text before it", now[4])
	}
	if vectored(t, db, now[4]) {
		t.Error("the window the edit landed in carries a vector made from other text")
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors`); got != 4 {
		t.Errorf("%d vectors survived the edit, want the four windows before it", got)
	}
	if asked := model.run(t, db, first); asked != 1 {
		t.Errorf("editing the end of a note asked for %d vectors, want 1", asked)
	}
}

func TestAWindowThatMovedInTheFileKeepsItsVector(t *testing.T) {
	// A chunk is a place in a file, and the body of a note begins after its
	// frontmatter. A line added to the frontmatter moves every window in the file
	// and changes the text of none of them, so every row is kept and every one is
	// moved to where its text now is.
	ctx := t.Context()
	db := opened(t)
	model := embedder{seed: 0x44}

	body := wordsOf(200)
	n := noteAt("notes/Entropy.md", "Entropy", body)
	save(t, db, first, n)
	model.run(t, db, first)

	was := smallWindows(t, db, first, n.Ref.Path)
	enclosing := largeWindow(t, db, first, n.Ref.Path)

	const frontmatter = 20
	moved := n
	moved.Ref.Size = int64(len(body) + frontmatter)
	save(t, db, first, moved)

	if got := smallWindows(t, db, first, n.Ref.Path); !slices.Equal(got, was) {
		t.Errorf("the windows are rows %v, and were rows %v", got, was)
	}
	if got := largeWindow(t, db, first, n.Ref.Path); got != enclosing {
		t.Errorf("the large window is row %d, and was row %d", got, enclosing)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors`); got != len(was) {
		t.Errorf("%d vectors, want the %d windows that carry one", got, len(was))
	}
	if asked := model.run(t, db, first); asked != 0 {
		t.Errorf("a note whose body did not change asked for %d vectors", asked)
	}

	// Every window is read from where its text is now.
	for _, row := range append(was, enclosing) {
		p, found, err := db.ChunkQueries().Passage(ctx, first.ID, row)
		if err != nil {
			t.Fatal(err)
		}
		if !found {
			t.Fatalf("row %d is gone", row)
		}
		if p.Start < frontmatter || p.Start+p.Length > int(moved.Ref.Size) {
			t.Errorf("row %d is read at %d for %d, and the body begins at %d",
				row, p.Start, p.Length, frontmatter)
		}
	}
}

func TestEditingTheMiddleOfANoteRecutsWhatFollowsIt(t *testing.T) {
	// A note that names no place is tiled from its first word, so a word typed
	// half way down it moves the window it landed in and every window after that.
	// The windows in front of it hold the text they held.
	db := opened(t)
	model := embedder{seed: 0x11}

	body := wordsOf(200)
	n := noteAt("notes/Entropy.md", "Entropy", body)
	save(t, db, first, n)

	if asked := model.run(t, db, first); asked != 5 {
		t.Fatalf("a note of 200 words owes %d vectors, want 5", asked)
	}
	was := smallWindows(t, db, first, n.Ref.Path)

	save(t, db, first, noteAt(n.Ref.Path, n.Title, typedBefore(body, 96)))

	now := smallWindows(t, db, first, n.Ref.Path)
	if len(now) != 5 {
		t.Fatalf("%d windows carry a vector after the edit", len(now))
	}
	// The word landed in the third window, which is tiled from word 80.
	for at, row := range was[:2] {
		if now[at] != row {
			t.Errorf("the window at %d is row %d, and was row %d", at, now[at], row)
		}
		if !vectored(t, db, row) {
			t.Errorf("row %d lost its vector, and the text it holds did not change", row)
		}
	}
	for at := 2; at < len(now); at++ {
		if slices.Contains(was, now[at]) {
			t.Errorf("the window at %d is row %d, the row of the text before the edit", at, now[at])
		}
		if vectored(t, db, now[at]) {
			t.Errorf("the window at %d carries a vector made from other text", at)
		}
	}
	if asked := model.run(t, db, first); asked != 3 {
		t.Errorf("editing the middle of a note asked for %d vectors, want 3", asked)
	}
}

// cost is what one save of a note costs the model, and how much of what the
// index held survived it.
type cost struct {
	// asked is the vectors the save left owing, kept the windows that carried one
	// and still do, and was and now how many windows carry a vector on either
	// side of the save.
	asked, kept, was, now int
}

// costOf saves a note, embeds every window of it, saves the note as edited, and
// reports what the second save cost. A window that kept its row is asserted to
// have kept its vector with it.
func costOf(t *testing.T, before, after domain.Note) cost {
	t.Helper()
	db := opened(t)
	model := embedder{seed: 0x11}

	save(t, db, first, before)
	if asked := model.run(t, db, first); asked == 0 {
		t.Fatal("the note owes no vector, so this measures nothing")
	}
	was := smallWindows(t, db, first, before.Ref.Path)

	save(t, db, first, after)
	now := smallWindows(t, db, first, after.Ref.Path)

	kept := 0
	for _, row := range now {
		if !slices.Contains(was, row) {
			continue
		}
		kept++
		if !vectored(t, db, row) {
			t.Errorf("row %d kept its number and lost its vector", row)
		}
	}
	return cost{asked: model.run(t, db, first), kept: kept, was: len(was), now: len(now)}
}

func TestAHeadingBoundsWhatAnEditRecuts(t *testing.T) {
	// A note of 200 words under four headings, edited in four places. The same
	// words are cut twice: once with the headings naming the sections they open,
	// and once with nothing naming a place, where the note is one span.
	//
	// Every figure here is in docs/performance.md.
	const path = "notes/Entropy.md"
	body := sectionsOf(192, 48)

	frontmattered := parsedAt(path, body)
	frontmattered.Ref.Size += 20

	for _, edit := range []struct {
		name            string
		note            domain.Note
		placed, unnamed cost
	}{
		{
			name:    "a line added to the frontmatter",
			note:    frontmattered,
			placed:  cost{asked: 0, kept: 4, was: 4, now: 4},
			unnamed: cost{asked: 0, kept: 5, was: 5, now: 5},
		},
		{
			name:    "at the end of the body",
			note:    parsedAt(path, body+" wordzz"),
			placed:  cost{asked: 1, kept: 4, was: 4, now: 5},
			unnamed: cost{asked: 1, kept: 4, was: 5, now: 5},
		},
		{
			name:    "in the middle of the body",
			note:    parsedAt(path, typedBefore(body, 96)),
			placed:  cost{asked: 2, kept: 3, was: 4, now: 5},
			unnamed: cost{asked: 3, kept: 2, was: 5, now: 5},
		},
		{
			name:    "at the start of the body",
			note:    parsedAt(path, typedBefore(body, 0)),
			placed:  cost{asked: 2, kept: 3, was: 4, now: 5},
			unnamed: cost{asked: 5, kept: 0, was: 5, now: 5},
		},
	} {
		t.Run(edit.name, func(t *testing.T) {
			if got := costOf(t, parsedAt(path, body), edit.note); got != edit.placed {
				t.Errorf("with the headings as places: %+v, want %+v", got, edit.placed)
			}
			if got := costOf(t, unplaced(parsedAt(path, body)), unplaced(edit.note)); got != edit.unnamed {
				t.Errorf("with no place named: %+v, want %+v", got, edit.unnamed)
			}
		})
	}
}

func TestEditingTheStartOfANoteRecutsAllOfIt(t *testing.T) {
	// Windows are tiled from the first word of the note, so a word put in front
	// of it moves every window after it, and every one owes a vector again.
	db := opened(t)
	model := embedder{seed: 0x11}

	body := wordsOf(200)
	n := noteAt("notes/Entropy.md", "Entropy", body)
	save(t, db, first, n)

	if asked := model.run(t, db, first); asked != 5 {
		t.Fatalf("a note of 200 words owes %d vectors, want 5", asked)
	}
	was := smallWindows(t, db, first, n.Ref.Path)

	save(t, db, first, noteAt(n.Ref.Path, n.Title, "wordzz "+body))

	if got := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors`); got != 0 {
		t.Errorf("%d vectors survived an edit at the start of the note", got)
	}
	now := smallWindows(t, db, first, n.Ref.Path)
	for _, row := range now {
		if slices.Contains(was, row) {
			t.Errorf("row %d holds text it did not hold before", row)
		}
	}
	if asked := model.run(t, db, first); asked != len(now) {
		t.Errorf("editing the start of a note asked for %d vectors, want its %d windows", asked, len(now))
	}
}

func TestARecutKeepsAWindowInsideALargeOneThatChanged(t *testing.T) {
	// The large window covers the whole note, so its text moves on every edit and
	// it is a new row every time. `chunks.parent … ON DELETE CASCADE` takes every
	// window inside a large one with it, so the windows that were kept are
	// pointed at the new large window before the old one comes out.
	db := opened(t)
	model := embedder{seed: 0x22}

	body := wordsOf(200)
	n := noteAt("notes/Entropy.md", "Entropy", body)
	save(t, db, first, n)
	model.run(t, db, first)

	enclosing := largeWindow(t, db, first, n.Ref.Path)
	kept := smallWindows(t, db, first, n.Ref.Path)[0]
	if !vectored(t, db, kept) {
		t.Fatal("the window this is about carries no vector, so it would pass either way")
	}

	save(t, db, first, noteAt(n.Ref.Path, n.Title, body+" wordzz"))

	now := largeWindow(t, db, first, n.Ref.Path)
	if now == enclosing {
		t.Fatalf("the large window is row %d after the note was edited, so its text did not move", now)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks WHERE id = ?`, kept); got != 1 {
		t.Fatal("the window whose text did not change went with the large window enclosing it")
	}
	if !vectored(t, db, kept) {
		t.Error("the window whose text did not change lost its vector")
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks WHERE id = ? AND parent = ?`, kept, now); got != 1 {
		t.Error("the window that was kept does not sit inside the large window that is there now")
	}
}

func TestTheFullTextRowSurvivesWithTheChunk(t *testing.T) {
	// A hit in the words half and a hit in the dense half name one row, so a
	// chunk that keeps its number keeps what was indexed under it.
	ctx := t.Context()
	db := opened(t)

	body := wordsOf(200)
	n := noteAt("notes/Entropy.md", "Entropy", body)
	save(t, db, first, n)

	kept := smallWindows(t, db, first, n.Ref.Path)[0]
	// A word of the first window, which an edit at the other end does not reach.
	opening := "word" + spelt(3)

	save(t, db, first, noteAt(n.Ref.Path, n.Title, body+" wordzz"))

	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_fts WHERE rowid = ?`, kept); got != 1 {
		t.Errorf("%d full-text rows for the window that was kept", got)
	}
	// One row per chunk, and none naming a chunk that is gone.
	chunks := counted(t, db, `SELECT COUNT(*) FROM chunks`)
	if got := counted(t, db, `SELECT COUNT(*) FROM chunks_fts`); got != chunks {
		t.Errorf("%d rows in the full-text index for %d chunks", got, chunks)
	}
	if got := counted(t, db,
		`SELECT COUNT(*) FROM chunks_fts WHERE rowid NOT IN (SELECT id FROM chunks)`); got != 0 {
		t.Errorf("%d full-text rows name a chunk that is gone", got)
	}

	found, err := db.ChunkQueries().Lexical(ctx, first.ID, opening, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatalf("the note is not findable by %s, which is in the window that was kept", opening)
	}
	for _, p := range found {
		if p.Source != n.Ref.Path {
			t.Errorf("%s answered for a word of %s", p.Source, n.Ref.Path)
		}
	}
	typed, err := db.ChunkQueries().Lexical(ctx, first.ID, "wordzz", 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(typed) == 0 {
		t.Error("the word that was typed is in no full-text row")
	}
}

func TestARecutStaysInsideItsVault(t *testing.T) {
	// One database holds every vault. The two notes here are at one path and
	// share no word, so a row that came from the wrong vault is recognisable and
	// a cut that reached across would take the other vault's chunks with it.
	ctx := t.Context()
	db := opened(t)
	model := embedder{seed: 0x33}

	mine := noteAt("notes/Entropy.md", "Entropy", wordsOf(200))
	theirs := noteAt("notes/Entropy.md", "Quasar",
		"quasar "+strings.ReplaceAll(wordsOf(200), "word", "other"))
	save(t, db, first, mine)
	save(t, db, second, theirs)
	model.run(t, db, first)
	model.run(t, db, second)

	untouched := smallWindows(t, db, second, theirs.Ref.Path)
	if len(untouched) == 0 {
		t.Fatal("the second vault holds no window that carries a vector")
	}
	was := chunksIn(t, db, second)
	vectors := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors`)

	save(t, db, first, noteAt(mine.Ref.Path, mine.Title, mine.Body+" wordzz"))

	for _, row := range untouched {
		if !vectored(t, db, row) {
			t.Errorf("row %d of the second vault lost its vector when the first vault's note was cut", row)
		}
	}
	if got := chunksIn(t, db, second); got != was {
		t.Errorf("the second vault holds %d chunks, and held %d before the first vault's note was cut", got, was)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors`); got != vectors-1 {
		t.Errorf("%d vectors, want %d: the one window the edit landed in", got, vectors-1)
	}

	// Neither vault answers for the other.
	for _, ask := range []struct {
		vault domain.Vault
		word  string
		want  bool
	}{
		{first, "wordzz", true},
		{second, "wordzz", false},
		{second, "quasar", true},
		{first, "quasar", false},
	} {
		found, err := db.ChunkQueries().Lexical(ctx, ask.vault.ID, ask.word, 10, false)
		if err != nil {
			t.Fatal(err)
		}
		if (len(found) > 0) != ask.want {
			t.Errorf("%s answered %d passages for %q, want %v", ask.vault.Name, len(found), ask.word, ask.want)
		}
	}
}

// locationsOf is what the windows of one note are called, in the order they sit
// in the file. A small window is named after the section it was cut inside.
func locationsOf(t *testing.T, db *DB, vault domain.Vault, path string) []string {
	t.Helper()
	rows, err := db.read.QueryContext(t.Context(), `SELECT COALESCE(c.location, '') FROM chunks c
	                                                JOIN sources s ON s.id = c.source_id
	                                                JOIN vaults v ON v.id = c.vault_id
	                                                WHERE v.identifier = ? AND s.path = ? AND c.parent IS NOT NULL
	                                                ORDER BY c.start, c.length`, vault.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var location string
		if err := rows.Scan(&location); err != nil {
			t.Fatal(err)
		}
		out = append(out, location)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAWindowIsNamedAfterTheSectionItWasCutInside(t *testing.T) {
	// A heading names the text under it, and that name is on every window cut
	// there, which is what a result has to show to say where the passage is.
	db := opened(t)

	n := parsedAt("notes/Entropy.md", sectionsOf(192, 48))
	save(t, db, first, n)

	want := []string{"Sectiona", "Sectionei", "Sectionjg", "Sectionbee"}
	if got := locationsOf(t, db, first, n.Ref.Path); !slices.Equal(got, want) {
		t.Errorf("the windows are located %v, want %v", got, want)
	}
}

func TestANoteWithSectionsIsStillFoundByItsTitle(t *testing.T) {
	// One large window encloses the whole note however many sections it holds, and
	// the note's title is in the text of it, so a note answers to the name it was
	// given and not only to the words in it.
	ctx := t.Context()
	db := opened(t)

	body := sectionsOf(192, 48)
	n := parsedAt("notes/Entropy.md", body)
	if strings.Contains(body, n.Title) {
		t.Fatalf("the body holds the title %q, so a hit on it proves nothing", n.Title)
	}
	save(t, db, first, n)

	found, err := db.ChunkQueries().Lexical(ctx, first.ID, n.Title, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatalf("the note is not findable by %q, which is its title", n.Title)
	}
	// The enclosing window is the note: it begins at the first word of the body and
	// ends at the last.
	p, ok, err := db.ChunkQueries().Passage(ctx, first.ID, largeWindow(t, db, first, n.Ref.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("the window enclosing the note is gone")
	}
	if p.Start != 0 || p.Length != len(body) {
		t.Errorf("the enclosing window is at %d for %d, and the body is %d bytes", p.Start, p.Length, len(body))
	}
}

func TestCuttingOneSectionStaysInsideItsVault(t *testing.T) {
	// One database holds every vault. The two notes here are at one path, are cut
	// into sections of the same shape, and share no word of prose, so a window that
	// arrived from the wrong vault is recognisable and a cut that reached across
	// would take the other vault's windows with it.
	ctx := t.Context()
	db := opened(t)
	model := embedder{seed: 0x33}

	const path = "notes/Entropy.md"
	body := sectionsOf(192, 48)
	mine := parsedAt(path, body)
	theirs := parsedAt(path, strings.Replace(
		strings.ReplaceAll(body, "word", "other"), "## Sectiona", "## Quasar", 1))
	save(t, db, first, mine)
	save(t, db, second, theirs)
	model.run(t, db, first)
	model.run(t, db, second)

	untouched := smallWindows(t, db, second, path)
	if len(untouched) == 0 {
		t.Fatal("the second vault holds no window that carries a vector")
	}
	was := chunksIn(t, db, second)
	vectors := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors`)

	// A word typed into the first section of the first vault's note.
	save(t, db, first, parsedAt(path, typedBefore(body, 0)))

	for _, row := range untouched {
		if !vectored(t, db, row) {
			t.Errorf("row %d of the second vault lost its vector when the first vault's note was cut", row)
		}
	}
	if got := chunksIn(t, db, second); got != was {
		t.Errorf("the second vault holds %d chunks, and held %d before the first vault's note was cut", got, was)
	}
	// The section the word landed in is cut into two windows, and both of them owe
	// a vector; the sections it did not reach keep theirs.
	if got := counted(t, db, `SELECT COUNT(*) FROM chunk_vectors`); got != vectors-1 {
		t.Errorf("%d vectors, want %d", got, vectors-1)
	}
	if asked := model.run(t, db, first); asked != 2 {
		t.Errorf("editing the first section asked for %d vectors, want 2", asked)
	}

	// Neither vault answers for the other.
	for _, ask := range []struct {
		vault domain.Vault
		word  string
		want  bool
	}{
		{first, "wordzz", true},
		{second, "wordzz", false},
		{second, "quasar", true},
		{first, "quasar", false},
	} {
		found, err := db.ChunkQueries().Lexical(ctx, ask.vault.ID, ask.word, 10, false)
		if err != nil {
			t.Fatal(err)
		}
		if (len(found) > 0) != ask.want {
			t.Errorf("%s answered %d passages for %q, want %v", ask.vault.Name, len(found), ask.word, ask.want)
		}
	}
}

func TestTheWorstCaseIsBoundedByTheSectionAndNotByTheNote(t *testing.T) {
	// A note of a thousand words under five headings, edited in the same four
	// places. What an edit re-cuts is every window of the span it lands in: a note
	// that names no place is one span, so a longer note has more of them, and a
	// heading opening each section makes it the windows of one section at any
	// length.
	//
	// Every figure here is in docs/performance.md.
	const path = "notes/Entropy.md"
	body := sectionsOf(1000, 200)

	frontmattered := parsedAt(path, body)
	frontmattered.Ref.Size += 20

	for _, edit := range []struct {
		name            string
		note            domain.Note
		placed, unnamed cost
	}{
		{
			name:    "a line added to the frontmatter",
			note:    frontmattered,
			placed:  cost{asked: 0, kept: 25, was: 25, now: 25},
			unnamed: cost{asked: 0, kept: 25, was: 25, now: 25},
		},
		{
			name:    "at the end of the body",
			note:    parsedAt(path, body+" wordzz"),
			placed:  cost{asked: 1, kept: 24, was: 25, now: 25},
			unnamed: cost{asked: 1, kept: 25, was: 25, now: 26},
		},
		{
			name:    "in the middle of the body",
			note:    parsedAt(path, typedBefore(body, 500)),
			placed:  cost{asked: 3, kept: 22, was: 25, now: 25},
			unnamed: cost{asked: 14, kept: 12, was: 25, now: 26},
		},
		{
			name:    "at the start of the body",
			note:    parsedAt(path, typedBefore(body, 0)),
			placed:  cost{asked: 5, kept: 20, was: 25, now: 25},
			unnamed: cost{asked: 26, kept: 0, was: 25, now: 26},
		},
	} {
		t.Run(edit.name, func(t *testing.T) {
			if got := costOf(t, parsedAt(path, body), edit.note); got != edit.placed {
				t.Errorf("with the headings as places: %+v, want %+v", got, edit.placed)
			}
			if got := costOf(t, unplaced(parsedAt(path, body)), unplaced(edit.note)); got != edit.unnamed {
				t.Errorf("with no place named: %+v, want %+v", got, edit.unnamed)
			}
		})
	}
}

func TestASectionShorterThanAWindowIsAWindowOfItsOwn(t *testing.T) {
	// What places cost. A window is cut inside one section and never across a
	// heading, so a note of many short sections is cut into many short windows, and
	// every one of them owes a vector of its own.
	//
	// The figures are in docs/performance.md.
	db := opened(t)

	body := sectionsOf(200, 5)
	named := parsedAt("notes/Named.md", body)
	unnamed := unplaced(parsedAt("notes/Unnamed.md", body))
	save(t, db, first, named)
	save(t, db, first, unnamed)

	if got := len(smallWindows(t, db, first, named.Ref.Path)); got != 40 {
		t.Errorf("a note of forty sections is cut into %d windows, want 40", got)
	}
	if got := len(smallWindows(t, db, first, unnamed.Ref.Path)); got != 7 {
		t.Errorf("the same words with no place named are cut into %d windows, want 7", got)
	}
}
