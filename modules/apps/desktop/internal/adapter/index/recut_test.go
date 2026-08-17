package index

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// A note is cut into one large window over the whole of it, tiled into the small
// windows that carry the vectors. These tests are about what a second cut of the
// same note keeps.

// wordsOf is a body of n words, each one different from every other, so that a
// window is recognisable by what it holds and no two windows of one note hold
// the same text.
func wordsOf(n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, "word"+spelt(i))
	}
	return strings.Join(out, " ")
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
	stored := counted(t, db, `SELECT COUNT(*) FROM vectors WHERE chunk_id = ?`, chunk)
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
	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != 4 {
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
	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != len(was) {
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

	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != 0 {
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

	found, err := db.ChunkQueries().Lexical(ctx, first.ID, opening, 10)
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
	typed, err := db.ChunkQueries().Lexical(ctx, first.ID, "wordzz", 10)
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
	vectors := counted(t, db, `SELECT COUNT(*) FROM vectors`)

	save(t, db, first, noteAt(mine.Ref.Path, mine.Title, mine.Body+" wordzz"))

	for _, row := range untouched {
		if !vectored(t, db, row) {
			t.Errorf("row %d of the second vault lost its vector when the first vault's note was cut", row)
		}
	}
	if got := chunksIn(t, db, second); got != was {
		t.Errorf("the second vault holds %d chunks, and held %d before the first vault's note was cut", got, was)
	}
	if got := counted(t, db, `SELECT COUNT(*) FROM vectors`); got != vectors-1 {
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
		found, err := db.ChunkQueries().Lexical(ctx, ask.vault.ID, ask.word, 10)
		if err != nil {
			t.Fatal(err)
		}
		if (len(found) > 0) != ask.want {
			t.Errorf("%s answered %d passages for %q, want %v", ask.vault.Name, len(found), ask.word, ask.want)
		}
	}
}
