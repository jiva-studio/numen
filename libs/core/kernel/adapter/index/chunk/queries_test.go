package chunk_test

import (
	"context"
	"database/sql"
	"encoding/hex"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const (
	vaultA  domain.VaultID = "01A"
	vaultB  domain.VaultID = "01B"
	nobody  domain.VaultID = "01NOBODY"
	recipe                 = "model"
	century                = int64(1_000_000_000_000_000_000)
)

func newMigratedTemplate() *vaultfixture.Template {
	return vaultfixture.NewTemplate(func(ctx context.Context, path string) error {
		db, err := index.Open(ctx, path)
		if err != nil {
			return err
		}
		return db.Close()
	})
}

func seedBits(seed byte) []byte {
	b := make([]byte, 128)
	for i := range b {
		b[i] = seed
	}
	return b
}

func seedVector(seed byte) []float32 {
	v := make([]float32, 1024)
	for i := range v {
		v[i] = -1
		if seed&(1<<(7-uint(i)%8)) != 0 {
			v[i] = 1
		}
	}
	return v
}

func seedBytes(seed byte) []byte {
	q := vector.Bytes(vector.Normalize(seedVector(seed)))
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

func openIndex(t *testing.T) *index.DB {
	t.Helper()
	db, _ := openIndexAt(t)
	return db
}

func openIndexAt(t *testing.T) (*index.DB, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	path := filepath.Join(t.TempDir(), "index.db")
	newMigratedTemplate().CopyTo(t, path)
	db, err := index.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	for _, v := range []domain.VaultID{vaultA, vaultB} {
		if err := db.Vaults().Register(t.Context(), v); err != nil {
			t.Fatal(err)
		}
	}
	return db, path
}

// source is one file the index holds, cut into one large chunk that holds a
// small chunk for each text in `smalls`.
type source struct {
	vault    domain.VaultID
	path     string
	kind     string
	size     int64
	mtime    int64
	producer string
	hash     string
	recipe   string
	large    string
	smalls   []string
	opens    []string
	location string
}

func (s source) save(t *testing.T, db *index.DB) {
	t.Helper()
	if s.vault == "" {
		s.vault = vaultA
	}
	if s.kind == "" {
		s.kind = "book"
	}
	if s.hash == "" {
		s.hash = "hash-" + s.path
	}
	if s.recipe == "" {
		s.recipe = "epub"
	}
	if s.location == "" {
		s.location = "chapter 1"
	}
	small := make([]chunk.Chunk, 0, len(s.smalls))
	for i, text := range s.smalls {
		small = append(small, chunk.Chunk{Start: 100 + 50*i, Length: 50, Text: text})
	}
	var chunks []chunk.Chunk
	if s.large != "" {
		chunks = []chunk.Chunk{{
			Start: 100, Length: 100, Location: s.location, Text: s.large, Opens: s.opens, Small: small,
		}}
	}
	err := db.Chunks().SaveExtraction(t.Context(), s.vault, chunk.Source{
		Path: s.path, Kind: s.kind, Size: s.size, MTime: s.mtime,
		ContentHash: s.hash, Recipe: s.recipe, Producer: s.producer,
	}, chunks)
	if err != nil {
		t.Fatal(err)
	}
}

func getUnembeddedChunks(t *testing.T, db *index.DB, vault domain.VaultID) []chunk.Record {
	t.Helper()
	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), vault, recipe, chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	return unembedded
}

func embed(t *testing.T, db *index.DB, vault domain.VaultID, path string, seed byte) {
	t.Helper()
	var vectors []chunk.Vector
	for _, p := range getUnembeddedChunks(t, db, vault) {
		if p.Path != path {
			continue
		}
		hash, err := hex.DecodeString(p.ChunkHash)
		if err != nil {
			t.Fatal(err)
		}
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hash, Recipe: recipe, Value: seedBytes(seed), Coarse: seedBits(seed),
		})
	}
	if err := db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
		t.Fatal(err)
	}
}

func getPaths(found []domain.FileStat) []string {
	out := make([]string, 0, len(found))
	for _, f := range found {
		out = append(out, f.Path)
	}
	return out
}

func getSources(found []domain.SearchResult) []string {
	out := make([]string, 0, len(found))
	for _, p := range found {
		out = append(out, p.Path)
	}
	return out
}

func broken(t *testing.T, seed func(db *index.DB), statements ...string) chunkQueries {
	t.Helper()
	db, path := openIndexAt(t)
	seed(db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, raw)
	for _, statement := range statements {
		if _, err := raw.ExecContext(t.Context(), statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	return queriesOfRaw(raw)
}

func closed(t *testing.T) chunkQueries {
	t.Helper()
	db := openIndex(t)
	queries := queriesOf(db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return queries
}

func TestAChunkIdentifierIsTheDecimalOfItsRow(t *testing.T) {
	for _, row := range []int64{0, 7, 10, 19, 4611686018427387905, math.MaxInt64} {
		id := chunk.ID(row)
		if string(id) != strconv.FormatInt(row, 10) {
			t.Errorf("row %d is spelled %q", row, id)
		}
		back, err := chunk.Row(id)
		if err != nil || back != row {
			t.Errorf("%q names row %d, %v; want %d", id, back, err, row)
		}
	}
}

func TestAnIdentifierThatIsNotDecimalNamesNoChunk(t *testing.T) {
	for _, id := range []domain.ChunkID{"", "seven", "1a", "0x10", "9223372036854775808", "1 2"} {
		row, err := chunk.Row(id)
		if err == nil {
			t.Errorf("%q names row %d", id, row)
			continue
		}
		if row != 0 {
			t.Errorf("%q fails with row %d", id, row)
		}
		if !strings.Contains(err.Error(), strconv.Quote(string(id))) {
			t.Errorf("the rejection does not say which identifier: %v", err)
		}
	}
}

func TestFingerprintsAreTheFilesOfOneKindInOneVault(t *testing.T) {
	db := openIndex(t)
	pos := time.Date(2024, 3, 1, 12, 0, 0, 123, time.UTC)
	source{path: "n/one.md", kind: "note", size: 11, mtime: pos.UnixNano()}.save(t, db)
	source{path: "n/two.md", kind: "note", size: 22}.save(t, db)
	source{path: "b/x.epub", kind: "book", size: 5, mtime: 9}.save(t, db)
	source{vault: vaultB, path: "n/other.md", kind: "note", size: 3, mtime: 9}.save(t, db)

	got, err := db.ChunkFiles().GetFileStats(t.Context(), vaultA, "note")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d fingerprints, want the 2 notes of the vault: %v", len(got), got)
	}
	one := got["n/one.md"]
	if one.Path != "n/one.md" || one.Size != 11 || !one.ModTime.Equal(pos) {
		t.Errorf("the first note is %+v, want size 11 at %v", one, pos)
	}
	two := got["n/two.md"]
	if two.Path != "n/two.md" || two.Size != 22 || !two.ModTime.IsZero() {
		t.Errorf("the second note is %+v, want size 22 and no time", two)
	}
	if one.Kind != "" {
		t.Errorf("a fingerprint says its kind %q, and the kind is what was asked for", one.Kind)
	}
}

func TestFingerprintsOfAVaultTheIndexHasNeverReceivedAreAnEmptyMap(t *testing.T) {
	got, err := openIndex(t).ChunkFiles().GetFileStats(t.Context(), nobody, "note")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("got %v, want an empty map that can be written to", got)
	}
}

func TestAFingerprintReadFromAClosedOrBrokenIndexFails(t *testing.T) {
	if _, err := closed(t).GetFileStats(t.Context(), vaultA, "note"); err == nil {
		t.Error("a closed index answered")
	}
	removed := broken(t, func(*index.DB) {}, `ALTER TABLE sources RENAME TO gone`)
	if _, err := removed.GetFileStats(t.Context(), vaultA, "note"); err == nil {
		t.Error("an index with no sources table answered")
	}
	garbled := broken(t, func(db *index.DB) {
		source{path: "n/one.md", kind: "note", size: 1}.save(t, db)
	}, `UPDATE sources SET size = 'abc'`)
	if _, err := garbled.GetFileStats(t.Context(), vaultA, "note"); err == nil {
		t.Error("a size that is no number was read")
	}
}

func TestWhatIsBelowAPathIsTheFileOrEverythingTheFolderContains(t *testing.T) {
	db := openIndex(t)
	pos := time.Date(2022, 5, 6, 7, 8, 9, 10, time.UTC)
	source{path: "physics/Entropy.md", kind: "note", size: 40, mtime: pos.UnixNano()}.save(t, db)
	source{path: "physics/heat/Heat.md", kind: "note", size: 41}.save(t, db)
	source{path: "physics/A Book.epub", kind: "file", size: 42}.save(t, db)
	source{path: "physics-old/Stray.md", kind: "note", size: 43}.save(t, db)
	source{vault: vaultB, path: "physics/Other.md", kind: "note", size: 44}.save(t, db)
	queries := queriesOf(db)

	folder, err := queries.GetSourcesUnder(t.Context(), vaultA, "physics")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"physics/A Book.epub", "physics/Entropy.md", "physics/heat/Heat.md"}
	if !slices.Equal(getPaths(folder), want) {
		t.Fatalf("the folder holds %v, want %v", getPaths(folder), want)
	}
	if folder[0].Kind != domain.KindFile || folder[0].Size != 42 || !folder[0].ModTime.IsZero() {
		t.Errorf("the book is %+v", folder[0])
	}
	if folder[1].Kind != domain.KindNote || folder[1].Size != 40 || !folder[1].ModTime.Equal(pos) {
		t.Errorf("the note is %+v, want size 40 at %v", folder[1], pos)
	}

	file, err := queries.GetSourcesUnder(t.Context(), vaultA, "physics/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(getPaths(file), []string{"physics/Entropy.md"}) {
		t.Errorf("the file holds %v", getPaths(file))
	}

	none, err := queries.GetSourcesUnder(t.Context(), nobody, "physics")
	if err != nil || len(none) != 0 {
		t.Errorf("an unknown vault holds %v, %v", none, err)
	}
}

func TestWhatIsBelowAPathFailsOnAClosedOrBrokenIndex(t *testing.T) {
	if _, err := closed(t).GetSourcesUnder(t.Context(), vaultA, "p"); err == nil {
		t.Error("a closed index answered")
	}
	removed := broken(t, func(*index.DB) {}, `ALTER TABLE sources RENAME TO gone`)
	_, err := removed.GetSourcesUnder(t.Context(), vaultA, "p")
	if err == nil || !strings.Contains(err.Error(), "p") {
		t.Errorf("an index with no sources table answered, or did not say the path: %v", err)
	}
	garbled := broken(t, func(db *index.DB) {
		source{path: "p/a.md", kind: "note", size: 1}.save(t, db)
	}, `UPDATE sources SET size = 'abc'`)
	if _, err := garbled.GetSourcesUnder(t.Context(), vaultA, "p"); err == nil {
		t.Error("a size that is no number was read")
	}
}

func searchFixture(t *testing.T) *index.DB {
	t.Helper()
	db := openIndex(t)
	source{
		path: "lib/zebra.epub", kind: "file", producer: "ocr", hash: "h-zebra", location: "chapter 4",
		large: "umbrella zebra stripes at dawn", smalls: []string{"zebra", "zebra stripes at dawn and padding words around them"},
	}.save(t, db)
	source{
		path: "notes/zebra.md", kind: "note",
		large: "zebra note", smalls: []string{"zebra note"},
	}.save(t, db)
	source{
		vault: vaultB, path: "lib/other.epub", kind: "file",
		large: "zebra elsewhere", smalls: []string{"zebra elsewhere"},
	}.save(t, db)
	return db
}

func TestASearchByWordsAnswersWithWhereEachHitExists(t *testing.T) {
	db := searchFixture(t)

	got, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "stripes", Kinds: []domain.SourceKind{domain.KindFile}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d hits, want the one small chunk", len(got))
	}
	for _, p := range got {
		if p.Path != "lib/zebra.epub" || p.Kind != domain.KindFile || p.VersionID != "ocr" ||
			p.FileHash != "h-zebra" || p.Start != 100 || p.Length != 100 || p.PlaceLabel != "chapter 4" {
			t.Errorf("a hit is %+v, want the large chunk of the book at 100 for 100 in chapter 4", p)
		}
		row, err := chunk.Row(p.ChunkID)
		if err != nil || row <= 0 {
			t.Errorf("a hit is named %q: %d, %v", p.ChunkID, row, err)
		}
	}
	if got[0].MatchOffset != 50 {
		t.Errorf("the hit begins at %d inside the passage, want the small chunk's 50", got[0].MatchOffset)
	}
}

func TestASearchByWordsStaysInItsVaultAndItsKinds(t *testing.T) {
	db := searchFixture(t)
	queries := queriesOf(db)

	all, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	sources := getSources(all)
	if len(all) != 3 || slices.Contains(sources, "lib/other.epub") {
		t.Errorf("a search of every kind answered %v, want the 2 chunks of the book and the note, none of the other vault", sources)
	}

	notes, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: []domain.SourceKind{domain.KindNote}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Kind != domain.KindNote {
		t.Errorf("a search of notes answered %v", getSources(notes))
	}

	both, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: []domain.SourceKind{domain.KindNote, domain.KindFile}, Limit: 10})
	if err != nil || len(both) != 3 {
		t.Errorf("a search of two kinds answered %d, %v", len(both), err)
	}
}

func TestFindLexical_HonorsLimit(t *testing.T) {
	db := searchFixture(t)
	queries := queriesOf(db)
	for _, kinds := range [][]domain.SourceKind{nil, {domain.KindFile}} {
		got, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: kinds, Limit: 2})
		if err != nil || len(got) != 2 {
			t.Errorf("kinds %v: asked for 2, got %d, %v", kinds, len(got), err)
		}
	}
}

func TestASearchByWordsAnswersBestFirst(t *testing.T) {
	db := searchFixture(t)
	got, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Path != "lib/zebra.epub" || got[0].MatchOffset != 0 || got[2].MatchOffset != 50 {
		t.Errorf("the shortest text with the word comes first and the longest last: %+v", got)
	}
}

func TestAWordStillBeingTypedMatchesByItsPrefix(t *testing.T) {
	db := searchFixture(t)
	queries := queriesOf(db)

	growing, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "strip", Kinds: nil, Limit: 10, IsGrowing: true})
	if err != nil || len(growing) == 0 {
		t.Errorf("a prefix still being typed found %d, %v", len(growing), err)
	}
	finished, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "strip", Kinds: nil, Limit: 10})
	if err != nil || len(finished) != 0 {
		t.Errorf("a finished prefix found %d, %v", len(finished), err)
	}
}

func TestASearchByWordsHasNothingToSayWithoutWordsOrVault(t *testing.T) {
	db := searchFixture(t)
	queries := queriesOf(db)

	for _, input := range []string{"", "   "} {
		got, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: input, Kinds: nil, Limit: 10})
		if err != nil || got != nil {
			t.Errorf("%q answered %v, %v", input, got, err)
		}
	}
	got, err := queries.FindLexical(t.Context(), nobody, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10})
	if err != nil || got != nil {
		t.Errorf("an unknown vault answered %v, %v", got, err)
	}
}

func TestASearchByWordsNeedsAPositiveLimit(t *testing.T) {
	db := searchFixture(t)
	for _, limit := range []int{0, -1} {
		got, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: limit})
		if err == nil || got != nil || !strings.Contains(err.Error(), strconv.Itoa(limit)) {
			t.Errorf("limit %d answered %v, %v", limit, got, err)
		}
	}
	got, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 1})
	if err != nil || len(got) != 1 {
		t.Errorf("a limit of 1 answered %d, %v", len(got), err)
	}
}

func TestASearchByWordsFailsOnAClosedOrBrokenIndex(t *testing.T) {
	seed := func(db *index.DB) {
		source{path: "lib/a.epub", kind: "file", large: "zebra", smalls: []string{"zebra"}}.save(t, db)
	}
	for name, queries := range map[string]chunkQueries{
		"closed":      closed(t),
		"no search":   broken(t, seed, `DROP TABLE chunks_fts`),
		"garbled row": broken(t, seed, `UPDATE chunks SET start = 'abc'`),
	} {
		if _, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 5}); err == nil {
			t.Errorf("%s: a search of every kind answered", name)
		}
		if _, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: []domain.SourceKind{domain.KindFile}, Limit: 5}); err == nil {
			t.Errorf("%s: a search of one kind answered", name)
		}
	}
}

func TestASectionIsFoundByItsNameAndAnswersWithTheChunkItOpens(t *testing.T) {
	db := openIndex(t)
	source{
		path: "lib/guide.epub", kind: "file", producer: "ocr", hash: "h-guide", location: "chapter 9",
		large: "text of the chapter", opens: []string{"Gardening"},
		smalls: []string{"first words of it"},
	}.save(t, db)
	source{
		path: "notes/Gardening.md", kind: "note", large: "text", opens: []string{"Gardening notes"}, smalls: []string{"x"},
	}.save(t, db)
	source{vault: vaultB, path: "lib/other.epub", kind: "file", large: "t", opens: []string{"Gardening"}, smalls: []string{"x"}}.save(t, db)
	queries := queriesOf(db)

	got, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "gardening", Kinds: nil, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d sections, want 2: %v", len(got), getSources(got))
	}
	var book domain.SearchResult
	for _, p := range got {
		if p.Path == "lib/guide.epub" {
			book = p
		}
	}
	if book.Kind != domain.KindFile || book.VersionID != "ocr" || book.FileHash != "h-guide" ||
		book.Start != 100 || book.Length != 100 || book.PlaceLabel != "chapter 9" || book.MatchOffset != 0 || book.ChunkID == "" {
		t.Errorf("the section is %+v, want the chunk it opens in chapter 9", book)
	}

	notes, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "gardening", Kinds: []domain.SourceKind{domain.KindNote}, Limit: 10})
	if err != nil || !slices.Equal(getSources(notes), []string{"notes/Gardening.md"}) {
		t.Errorf("sections of notes: %v, %v", getSources(notes), err)
	}

	one, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "gardening", Kinds: nil, Limit: 1})
	if err != nil || len(one) != 1 {
		t.Errorf("asked for 1, got %d, %v", len(one), err)
	}
	oneOfKind, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "gardening", Kinds: []domain.SourceKind{domain.KindFile, domain.KindNote}, Limit: 1})
	if err != nil || len(oneOfKind) != 1 {
		t.Errorf("asked for 1 of two kinds, got %d, %v", len(oneOfKind), err)
	}

	typing, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "garden", Kinds: nil, Limit: 10, IsGrowing: true})
	if err != nil || len(typing) != 2 {
		t.Errorf("a name still being typed found %d, %v", len(typing), err)
	}
	typed, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "garden", Kinds: nil, Limit: 10})
	if err != nil || len(typed) != 0 {
		t.Errorf("a finished prefix found %d, %v", len(typed), err)
	}
}

func TestASearchByNameHasNothingToSayWithoutWordsOrVault(t *testing.T) {
	queries := queriesOf(openIndex(t))
	for _, input := range []string{"", "  "} {
		got, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: input, Kinds: nil, Limit: 10})
		if err != nil || got != nil {
			t.Errorf("%q answered %v, %v", input, got, err)
		}
	}
	got, err := queries.GetNamedSections(t.Context(), nobody, sourcedomain.LexicalQuery{Text: "word", Kinds: nil, Limit: 10})
	if err != nil || got != nil {
		t.Errorf("an unknown vault answered %v, %v", got, err)
	}
	for _, limit := range []int{0, -3} {
		got, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "word", Kinds: nil, Limit: limit})
		if err == nil || got != nil || !strings.Contains(err.Error(), strconv.Itoa(limit)) {
			t.Errorf("limit %d answered %v, %v", limit, got, err)
		}
	}
}

func TestASearchByNameFailsOnAClosedOrBrokenIndex(t *testing.T) {
	seed := func(db *index.DB) {
		source{path: "lib/a.epub", kind: "file", large: "t", opens: []string{"Name"}, smalls: []string{"t"}}.save(t, db)
	}
	for name, queries := range map[string]chunkQueries{
		"closed":      closed(t),
		"no sections": broken(t, seed, `DROP TABLE sections_fts`),
		"garbled row": broken(t, seed, `UPDATE chunks SET start = 'abc'`),
	} {
		if _, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "name", Kinds: nil, Limit: 5}); err == nil {
			t.Errorf("%s: a search of every kind answered", name)
		}
		if _, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "name", Kinds: []domain.SourceKind{domain.KindFile}, Limit: 5}); err == nil {
			t.Errorf("%s: a search of one kind answered", name)
		}
	}
}

func meaningFixture(t *testing.T) *index.DB {
	t.Helper()
	db := openIndex(t)
	for _, b := range []struct {
		vault domain.VaultID
		path  string
		kind  string
		seed  byte
	}{
		{vaultA, "lib/near.epub", "file", 0xF0},
		{vaultA, "lib/middle.epub", "file", 0xF8},
		{vaultA, "lib/far.epub", "file", 0x0F},
		{vaultA, "notes/far.md", "note", 0x0F},
		{vaultB, "lib/elsewhere.epub", "file", 0xF0},
	} {
		source{
			vault: b.vault, path: b.path, kind: b.kind, producer: "ocr", hash: "h-" + b.path,
			large: "large " + b.path, smalls: []string{"first " + b.path, "second " + b.path},
		}.save(t, db)
		embed(t, db, b.vault, b.path, b.seed)
	}
	return db
}

func TestASearchByMeaningAnswersNearestFirstWithWhereEachExists(t *testing.T) {
	db := meaningFixture(t)

	got, err := queriesOf(db).FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 10, Floor: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 {
		t.Fatalf("got %d passages, want the 8 small chunks of the vault: %v", len(got), getSources(got))
	}
	want := []string{"lib/near.epub", "lib/near.epub", "lib/middle.epub", "lib/middle.epub"}
	if !slices.Equal(getSources(got)[:4], want) {
		t.Errorf("the nearest are %v, want %v", getSources(got)[:4], want)
	}
	first := got[0]
	if first.Kind != domain.KindFile || first.VersionID != "ocr" || first.FileHash != "h-lib/near.epub" ||
		first.Start != 100 || first.Length != 100 || first.PlaceLabel != "chapter 1" {
		t.Errorf("the nearest is %+v, want the large chunk it is inside", first)
	}
	hits := []int{got[0].MatchOffset, got[1].MatchOffset}
	slices.Sort(hits)
	if !slices.Equal(hits, []int{0, 50}) {
		t.Errorf("the two small chunks of the nearest book begin at %v inside it, want 0 and 50", hits)
	}
	if got[0].ChunkID == got[1].ChunkID {
		t.Errorf("two answers name %q", got[0].ChunkID)
	}
}

func TestFindNearest_HonorsLimit(t *testing.T) {
	db := meaningFixture(t)
	queries := queriesOf(db)

	for _, limit := range []int{1, 3} {
		got, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: limit, Floor: -1})
		if err != nil || len(got) != limit {
			t.Errorf("asked for %d, got %d, %v", limit, len(got), err)
		}
	}
	both, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 8, Floor: -1})
	if err != nil || len(both) != 8 {
		t.Errorf("asked for exactly what there is, got %d, %v", len(both), err)
	}
}

func TestASearchByMeaningAnswersNothingBelowTheFloor(t *testing.T) {
	db := meaningFixture(t)
	got, err := queriesOf(db).FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 10, Floor: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range got {
		if strings.Contains(p.Path, "far") {
			t.Errorf("%s is opposite the query and answers", p.Path)
		}
	}
	if len(got) != 4 {
		t.Errorf("got %d passages, want the 4 near enough", len(got))
	}
}

func TestASearchByMeaningStaysInItsKindsRecipeAndVault(t *testing.T) {
	db := meaningFixture(t)
	queries := queriesOf(db)

	notes, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0x0F), Kinds: []domain.SourceKind{domain.KindNote}, Limit: 10, Floor: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 || slices.ContainsFunc(notes, func(p domain.SearchResult) bool { return p.Kind != domain.KindNote }) {
		t.Errorf("a search of notes answered %v", getSources(notes))
	}
	other, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: "another model", Vector: seedVector(0xF0), Limit: 10, Floor: -1})
	if err != nil || len(other) != 0 {
		t.Errorf("a recipe nothing was made under answered %d, %v", len(other), err)
	}
	b, err := queries.FindNearest(t.Context(), vaultB, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 10, Floor: -1})
	if err != nil || !slices.Equal(getSources(b), []string{"lib/elsewhere.epub", "lib/elsewhere.epub"}) {
		t.Errorf("the other vault answered %v, %v", getSources(b), err)
	}
}

func TestASearchByMeaningHasNothingToSayWithoutAQueryOrAVault(t *testing.T) {
	queries := queriesOf(meaningFixture(t))

	for _, empty := range [][]float32{nil, {}} {
		got, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: empty, Limit: 10, Floor: -1})
		if err != nil || got != nil {
			t.Errorf("an empty query answered %v, %v", got, err)
		}
	}
	got, err := queries.FindNearest(t.Context(), nobody, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 10, Floor: -1})
	if err != nil || got != nil {
		t.Errorf("an unknown vault answered %v, %v", got, err)
	}
	for _, limit := range []int{0, -2} {
		got, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: limit, Floor: -1})
		if err == nil || got != nil || !strings.Contains(err.Error(), strconv.Itoa(limit)) {
			t.Errorf("limit %d answered %v, %v", limit, got, err)
		}
	}
}

func TestASearchByMeaningFailsOnAClosedOrBrokenIndex(t *testing.T) {
	seed := func(db *index.DB) {
		source{path: "lib/a.epub", large: "t", smalls: []string{"t"}}.save(t, db)
		embed(t, db, vaultA, "lib/a.epub", 0xF0)
	}
	for name, queries := range map[string]chunkQueries{
		"closed":    closed(t),
		"no coarse": broken(t, seed, `DROP TABLE chunks_vec`),
		"no vector": broken(t, seed, `ALTER TABLE vectors RENAME TO gone`),
	} {
		if _, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 5, Floor: -1}); err == nil {
			t.Errorf("%s: a search answered", name)
		}
	}
}

func TestAChunkWhoseVectorIsOfAnotherWidthIsACorruptRow(t *testing.T) {
	queries := broken(t, func(db *index.DB) {
		source{path: "lib/a.epub", large: "t", smalls: []string{"t"}}.save(t, db)
		embed(t, db, vaultA, "lib/a.epub", 0xF0)
	})
	if _, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0)[:512], Limit: 5, Floor: -1}); err == nil {
		t.Error("a query of 512 dimensions was compared to vectors of 1024")
	}
}

func TestARecordIsWhereOneChunkIsReadFrom(t *testing.T) {
	db := openIndex(t)
	source{
		path: "lib/a.epub", producer: "ocr", hash: "h-a", location: "chapter 7",
		large: "large", smalls: []string{"first", "second"},
	}.save(t, db)
	source{vault: vaultB, path: "lib/b.epub", large: "other", smalls: []string{"x"}}.save(t, db)
	queries := queriesOf(db)

	unembedded := getUnembeddedChunks(t, db, vaultA)
	if len(unembedded) != 2 {
		t.Fatalf("got %d small chunks, want 2", len(unembedded))
	}
	small := unembedded[1]
	got, found, err := queries.GetRecord(t.Context(), vaultA, small.Chunk)
	if err != nil || !found {
		t.Fatalf("the small chunk is not found: %v", err)
	}
	if got.Chunk != small.Chunk || got.Path != "lib/a.epub" || got.Producer != "ocr" || got.ContentHash != "h-a" ||
		got.Start != 150 || got.Length != 50 || got.Location != "" || got.Parent != small.Parent || got.Parent == 0 {
		t.Errorf("the small chunk is %+v, want it at 150 for 50, inside %d", got, small.Parent)
	}

	large, found, err := queries.GetRecord(t.Context(), vaultA, small.Parent)
	if err != nil || !found {
		t.Fatalf("the large chunk is not found: %v", err)
	}
	if large.Chunk != small.Parent || large.Start != 100 || large.Length != 100 || large.Location != "chapter 7" || large.Parent != 0 {
		t.Errorf("the large chunk is %+v, and is inside nothing", large)
	}

	for name, ask := range map[string]struct {
		vault domain.VaultID
		chunk int64
	}{
		"another vault": {vaultB, small.Chunk},
		"no such chunk": {vaultA, 987654},
		"unknown vault": {nobody, small.Chunk},
	} {
		got, found, err := queries.GetRecord(t.Context(), ask.vault, ask.chunk)
		if err != nil || found || got != (chunk.Record{}) {
			t.Errorf("%s: %+v, %v, %v", name, got, found, err)
		}
	}
}

func TestARecordFailsOnAClosedOrBrokenIndex(t *testing.T) {
	if _, _, err := closed(t).GetRecord(t.Context(), vaultA, 1); err == nil {
		t.Error("a closed index answered")
	}
	seed := func(db *index.DB) {
		source{path: "lib/a.epub", large: "t", smalls: []string{"t"}}.save(t, db)
	}
	for name, queries := range map[string]chunkQueries{
		"no chunks":   broken(t, seed, `ALTER TABLE chunks RENAME TO gone`),
		"garbled row": broken(t, seed, `UPDATE chunks SET start = 'abc'`),
	} {
		if _, found, err := queries.GetRecord(t.Context(), vaultA, 1); err == nil || found {
			t.Errorf("%s: answered %v, %v", name, found, err)
		}
	}
}

func TestSourcesWithNoSmallChunkOweTheSplit(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/c-cut.epub", large: "l", smalls: []string{"s"}}.save(t, db)
	source{path: "lib/b-large-only.epub", large: "l"}.save(t, db)
	source{path: "lib/a-nothing.epub"}.save(t, db)
	source{path: "lib/d-nothing.epub"}.save(t, db)
	source{path: "notes/e.md", kind: "note"}.save(t, db)
	source{vault: vaultB, path: "lib/f.epub"}.save(t, db)
	queries := queriesOf(db)

	got, err := queries.GetUnchunkedSources(t.Context(), vaultA, "book", 10)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"lib/a-nothing.epub", "lib/b-large-only.epub", "lib/d-nothing.epub"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	page, err := queries.GetUnchunkedSources(t.Context(), vaultA, "book", 2)
	if err != nil || len(page) != 2 {
		t.Errorf("a page of 2 holds %v, %v", page, err)
	}
	none, err := queries.GetUnchunkedSources(t.Context(), nobody, "book", 10)
	if err != nil || none != nil {
		t.Errorf("an unknown vault needs %v, %v", none, err)
	}
	for _, limit := range []int{0, -1} {
		if _, err := queries.GetUnchunkedSources(t.Context(), vaultA, "book", limit); err == nil || !strings.Contains(err.Error(), "unchunked") {
			t.Errorf("limit %d: %v", limit, err)
		}
	}
}

func TestSourcesOfAnotherRecipeOweTheExtractionAgain(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/a-old.epub", recipe: "epub-1"}.save(t, db)
	source{path: "lib/b-current.epub", recipe: "epub-2"}.save(t, db)
	source{path: "lib/c-other.epub", recipe: "pdf-1"}.save(t, db)
	source{path: "lib/d-none.epub"}.save(t, db)
	source{path: "notes/e.md", kind: "note", recipe: "old"}.save(t, db)
	source{vault: vaultB, path: "lib/f.epub", recipe: "old"}.save(t, db)
	queries := queriesOf(db)

	got, err := queries.GetSourcesWithOtherRecipe(t.Context(), vaultA, "book", []string{"epub-2", "pdf-1"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"lib/a-old.epub", "lib/d-none.epub"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	page, err := queries.GetSourcesWithOtherRecipe(t.Context(), vaultA, "book", []string{"x"}, 1)
	if err != nil || len(page) != 1 {
		t.Errorf("a page of 1 holds %v, %v", page, err)
	}
	for _, limit := range []int{0, -1} {
		if _, err := queries.GetSourcesWithOtherRecipe(t.Context(), vaultA, "book", nil, limit); err == nil || !strings.Contains(err.Error(), "stale_recipe") {
			t.Errorf("limit %d: %v", limit, err)
		}
	}
}

func TestListsOfPathsFailOnAClosedOrBrokenIndex(t *testing.T) {
	if _, err := closed(t).GetUnchunkedSources(t.Context(), vaultA, "book", 5); err == nil {
		t.Error("a closed index answered")
	}
	seed := func(db *index.DB) { source{path: "lib/a.epub"}.save(t, db) }
	removed := broken(t, seed, `ALTER TABLE sources RENAME TO gone`)
	if _, err := removed.GetUnchunkedSources(t.Context(), vaultA, "book", 5); err == nil {
		t.Error("an index with no sources answered")
	}
}

func TestUnembeddedChunksAreSmallOnesNewestSourceFirst(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/old.epub", mtime: century + 1, producer: "ocr", hash: "h-old", large: "l", smalls: []string{"o1", "o2"}}.save(t, db)
	source{path: "lib/new.epub", mtime: century + 5, large: "l", smalls: []string{"n1", "n2"}}.save(t, db)
	source{path: "lib/done.epub", mtime: century + 3, large: "l", smalls: []string{"d1"}}.save(t, db)
	source{vault: vaultB, path: "lib/b.epub", mtime: century + 9, large: "l", smalls: []string{"b1"}}.save(t, db)
	embed(t, db, vaultA, "lib/done.epub", 0xF0)

	got := getUnembeddedChunks(t, db, vaultA)
	var paths []string
	for _, p := range got {
		paths = append(paths, p.Path)
	}
	if want := []string{"lib/new.epub", "lib/new.epub", "lib/old.epub", "lib/old.epub"}; !slices.Equal(paths, want) {
		t.Fatalf("unembedded %v, want %v", paths, want)
	}
	first, last := got[0], got[3]
	if first.Start != 100 || first.Length != 50 || first.Location != "" || first.Parent == 0 || first.MTime != century+5 ||
		first.Source == 0 || first.ChunkHash == "" || first.ContentHash == "" || first.Chunk == 0 {
		t.Errorf("the first unembedded chunk is %+v", first)
	}
	if got[1].Start != 150 || got[1].Chunk <= first.Chunk {
		t.Errorf("the second chunk of a source follows the first: %+v after %+v", got[1], first)
	}
	if last.Producer != "ocr" || last.ContentHash != "h-old" || last.MTime != century+1 || last.Source >= first.Source {
		t.Errorf("the last unembedded chunk is %+v", last)
	}
}

func TestGetUnembeddedChunks_OtherRecipeReturnsAllChunks(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/a.epub", large: "l", smalls: []string{"s1", "s2"}}.save(t, db)
	embed(t, db, vaultA, "lib/a.epub", 0xF0)

	same, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{}, 10)
	if err != nil || len(same) != 0 {
		t.Errorf("the recipe the vectors were made under still needs %d, %v", len(same), err)
	}
	other, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), vaultA, "another model", chunk.Cursor{}, 10)
	if err != nil || len(other) != 2 {
		t.Errorf("another recipe needs %d, %v", len(other), err)
	}
}

func TestAWalkOfUnembeddedChunksInPagesVisitsEachOnce(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/a.epub", mtime: century + 1, large: "l", smalls: []string{"a1", "a2", "a3"}}.save(t, db)
	source{path: "lib/b.epub", mtime: century + 2, large: "l", smalls: []string{"b1", "b2"}}.save(t, db)
	source{path: "lib/c.epub", mtime: century + 2, large: "l", smalls: []string{"c1", "c2"}}.save(t, db)
	queries := queriesOf(db)
	all := getUnembeddedChunks(t, db, vaultA)

	for _, limit := range []int{1, 2, 3} {
		var visited []int64
		var after chunk.Cursor
		for range 20 {
			page, err := queries.GetUnembeddedChunks(t.Context(), vaultA, recipe, after, limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(page) == 0 {
				break
			}
			if len(page) > limit {
				t.Fatalf("a page of %d holds %d", limit, len(page))
			}
			for _, p := range page {
				visited = append(visited, p.Chunk)
			}
			last := page[len(page)-1]
			after = chunk.Cursor{MTime: last.MTime, Source: last.Source, Chunk: last.Chunk}
		}
		var want []int64
		for _, p := range all {
			want = append(want, p.Chunk)
		}
		if !slices.Equal(visited, want) {
			t.Errorf("pages of %d visited %v, want %v", limit, visited, want)
		}
	}
}

func TestGetUnembeddedChunks_CursorPastEndReturnsNone(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/a.epub", mtime: century, large: "l", smalls: []string{"a1"}}.save(t, db)
	queries := queriesOf(db)

	for _, cursor := range [][3]int64{{0, 0, 5}, {0, 5, 0}, {0, 5, 5}, {5, 0, 0}, {5, 0, 5}, {5, 5, 0}} {
		got, err := queries.GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{MTime: cursor[0], Source: cursor[1], Chunk: cursor[2]}, 10)
		if err != nil || len(got) != 0 {
			t.Errorf("cursor %v needs %d, %v", cursor, len(got), err)
		}
	}
	start, err := queries.GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{}, 10)
	if err != nil || len(start) != 1 {
		t.Errorf("a walk from the start needs %d, %v", len(start), err)
	}
}

func TestUnembeddedChunksNeedAPositiveLimitAndAKnownVault(t *testing.T) {
	queries := queriesOf(openIndex(t))
	for _, limit := range []int{0, -1} {
		got, err := queries.GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{}, limit)
		if err == nil || got != nil || !strings.Contains(err.Error(), strconv.Itoa(limit)) {
			t.Errorf("limit %d answered %v, %v", limit, got, err)
		}
	}
	got, err := queries.GetUnembeddedChunks(t.Context(), nobody, recipe, chunk.Cursor{}, 10)
	if err != nil || got != nil {
		t.Errorf("an unknown vault needs %v, %v", got, err)
	}
}

func TestUnembeddedChunksFailOnAClosedOrBrokenIndex(t *testing.T) {
	if _, err := closed(t).GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{}, 5); err == nil {
		t.Error("a closed index answered")
	}
	seed := func(db *index.DB) { source{path: "lib/a.epub", large: "l", smalls: []string{"s"}}.save(t, db) }
	for name, queries := range map[string]chunkQueries{
		"no vectors":  broken(t, seed, `ALTER TABLE vectors RENAME TO gone`),
		"garbled row": broken(t, seed, `UPDATE chunks SET start = 'abc'`),
	} {
		if _, err := queries.GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{}, 5); err == nil {
			t.Errorf("%s: answered", name)
		}
	}
}

func TestGetProgress_CountsSmallChunksAndEmbeddedByRecipe(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/a.epub", large: "l", smalls: []string{"a1", "a2", "a3"}}.save(t, db)
	source{path: "lib/b.epub", large: "l", smalls: []string{"b1", "b2"}}.save(t, db)
	source{vault: vaultB, path: "lib/c.epub", large: "l", smalls: []string{"c1"}}.save(t, db)
	embed(t, db, vaultA, "lib/a.epub", 0xF0)
	queries := queriesOf(db)

	target, embedded, err := queries.GetProgress(t.Context(), vaultA, recipe)
	if err != nil || target != 5 || embedded != 3 {
		t.Errorf("held %d, embedded %d, %v; want 5 and 3", target, embedded, err)
	}
	target, embedded, err = queries.GetProgress(t.Context(), vaultA, "another model")
	if err != nil || target != 5 || embedded != 0 {
		t.Errorf("under another model held %d, embedded %d, %v; want 5 and 0", target, embedded, err)
	}
	target, embedded, err = queries.GetProgress(t.Context(), nobody, recipe)
	if err != nil || target != 0 || embedded != 0 {
		t.Errorf("an unknown vault held %d, embedded %d, %v", target, embedded, err)
	}
}

func TestProgressFailsOnAClosedOrBrokenIndex(t *testing.T) {
	target, embedded, err := closed(t).GetProgress(t.Context(), vaultA, recipe)
	if err == nil || target != 0 || embedded != 0 {
		t.Errorf("a closed index answered %d, %d, %v", target, embedded, err)
	}
	removed := broken(t, func(*index.DB) {}, `ALTER TABLE chunks RENAME TO gone`)
	if _, _, err := removed.GetProgress(t.Context(), vaultA, recipe); err == nil {
		t.Error("an index with no chunks answered")
	}
}

func TestCachedVectorsAreFoundByTheHashOfTheirTextBelowTheRecipe(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/a.epub", large: "l", smalls: []string{"a1", "a2"}}.save(t, db)
	chunks := getUnembeddedChunks(t, db, vaultA)
	embed(t, db, vaultA, "lib/a.epub", 0xF0)
	var hashes [][]byte
	for _, p := range chunks {
		raw, err := hex.DecodeString(p.ChunkHash)
		if err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, raw)
	}

	got, err := db.ChunkVectors().GetKeptVectors(t.Context(), recipe, append([][]byte{nil, {}}, hashes...))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d vectors, want 2: %v", len(got), got)
	}
	for _, p := range chunks {
		if !slices.Equal(got[p.ChunkHash], seedBytes(0xF0)) {
			t.Errorf("the vector kept under %s is not the one saved", p.ChunkHash)
		}
	}
	elsewhere, err := db.ChunkVectors().GetKeptVectors(t.Context(), "another model", hashes)
	if err != nil || len(elsewhere) != 0 {
		t.Errorf("another recipe found %d, %v", len(elsewhere), err)
	}
	one, err := db.ChunkVectors().GetKeptVectors(t.Context(), recipe, hashes[:1])
	if err != nil || len(one) != 1 {
		t.Errorf("one hash found %d, %v", len(one), err)
	}
}

func TestCachedVectorsOfNoHashAreNone(t *testing.T) {
	queries := queriesOf(openIndex(t))
	for _, of := range [][][]byte{nil, {}, {nil}, {{}, nil}} {
		got, err := queries.GetKeptVectors(t.Context(), recipe, of)
		if err != nil || got != nil {
			t.Errorf("%v answered %v, %v", of, got, err)
		}
	}
	got, err := queries.GetKeptVectors(t.Context(), recipe, [][]byte{{1, 2, 3}})
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("a hash nothing was made for answered %v, %v", got, err)
	}
}

func TestCachedVectorsFailOnAClosedOrBrokenIndex(t *testing.T) {
	hashes := [][]byte{{1, 2, 3}}
	if _, err := closed(t).GetKeptVectors(t.Context(), recipe, hashes); err == nil {
		t.Error("a closed index answered")
	}
	removed := broken(t, func(*index.DB) {}, `ALTER TABLE vectors RENAME TO gone`)
	if _, err := removed.GetKeptVectors(t.Context(), recipe, hashes); err == nil {
		t.Error("an index with no vectors answered")
	}
}

func TestASourceTextReturnsWhatMadeItAndTheFileItWasMadeFrom(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/scan.pdf", kind: "book", size: 321, mtime: century + 7, producer: "ocr", hash: "h-scan"}.save(t, db)
	source{path: "lib/plain.epub", size: 12}.save(t, db)
	source{vault: vaultB, path: "lib/theirs.pdf", producer: "ocr"}.save(t, db)
	queries := queriesOf(db)

	got, found, err := queries.GetSourceText(t.Context(), vaultA, "lib/scan.pdf")
	want := chunk.SourceText{Path: "lib/scan.pdf", Producer: "ocr", ContentHash: "h-scan", Size: 321, MTime: century + 7}
	if err != nil || !found || got != want {
		t.Errorf("got %+v, %v, %v; want %+v", got, found, err, want)
	}
	plain, found, err := queries.GetSourceText(t.Context(), vaultA, "lib/plain.epub")
	if err != nil || !found || plain.Producer != "" || plain.Size != 12 || plain.Path != "lib/plain.epub" {
		t.Errorf("a source made of its own bytes is %+v, %v, %v", plain, found, err)
	}
	for name, ask := range map[string]struct {
		vault domain.VaultID
		path  string
	}{
		"no such path":  {vaultA, "lib/none.pdf"},
		"other vault":   {vaultA, "lib/theirs.pdf"},
		"unknown vault": {nobody, "lib/scan.pdf"},
	} {
		got, found, err := queries.GetSourceText(t.Context(), ask.vault, ask.path)
		if err != nil || found || got != (chunk.SourceText{}) {
			t.Errorf("%s: %+v, %v, %v", name, got, found, err)
		}
	}
}

func TestASourceTextFailsOnAClosedOrBrokenIndex(t *testing.T) {
	if _, found, err := closed(t).GetSourceText(t.Context(), vaultA, "lib/a.pdf"); err == nil || found {
		t.Errorf("a closed index answered %v, %v", found, err)
	}
	for name, statement := range map[string]string{
		"no sources":  `ALTER TABLE sources RENAME TO gone`,
		"garbled row": `UPDATE sources SET size = 'abc'`,
	} {
		queries := broken(t, func(db *index.DB) { source{path: "lib/a.pdf"}.save(t, db) }, statement)
		_, found, err := queries.GetSourceText(t.Context(), vaultA, "lib/a.pdf")
		if err == nil || found || !strings.Contains(err.Error(), "lib/a.pdf") {
			t.Errorf("%s: answered %v, %v", name, found, err)
		}
	}
}

func TestRecognizedSourcesAreTheOnesAProducerMadeTheTextOf(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/b.pdf", producer: "ocr", hash: "h-b"}.save(t, db)
	source{path: "lib/a.pdf", producer: "asr", hash: "h-a"}.save(t, db)
	source{path: "lib/c.pdf", producer: "ocr"}.save(t, db)
	source{path: "lib/plain.epub"}.save(t, db)
	source{path: "notes/n.md", kind: "note", producer: "ocr"}.save(t, db)
	source{vault: vaultB, path: "lib/theirs.pdf", producer: "ocr"}.save(t, db)
	queries := queriesOf(db)

	got, err := queries.GetRecognizedSources(t.Context(), vaultA, "book")
	if err != nil {
		t.Fatal(err)
	}
	want := []chunk.SourceText{
		{Path: "lib/a.pdf", Producer: "asr", ContentHash: "h-a"},
		{Path: "lib/b.pdf", Producer: "ocr", ContentHash: "h-b"},
		{Path: "lib/c.pdf", Producer: "ocr", ContentHash: "hash-lib/c.pdf"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	none, err := queries.GetRecognizedSources(t.Context(), nobody, "book")
	if err != nil || none != nil {
		t.Errorf("an unknown vault answered %v, %v", none, err)
	}
}

func TestRecognizedSourcesFailOnAClosedOrBrokenIndex(t *testing.T) {
	if _, err := closed(t).GetRecognizedSources(t.Context(), vaultA, "book"); err == nil {
		t.Error("a closed index answered")
	}
	removed := broken(t, func(*index.DB) {}, `ALTER TABLE sources RENAME TO gone`)
	if _, err := removed.GetRecognizedSources(t.Context(), vaultA, "book"); err == nil {
		t.Error("an index with no sources answered")
	}
}

func TestEveryQueryFailsWhenTheVaultCannotBeLooked(t *testing.T) {
	ctx := t.Context()
	q := broken(t, func(*index.DB) {}, `ALTER TABLE vaults RENAME TO gone`)
	kinds := []domain.SourceKind{domain.KindFile}

	checks := map[string]func() error{
		"GetFingerprints": func() error { _, err := q.GetFileStats(ctx, vaultA, "book"); return err },
		"GetSourcesUnder": func() error { _, err := q.GetSourcesUnder(ctx, vaultA, "p"); return err },
		"FindLexical": func() error {
			_, err := q.FindLexical(ctx, vaultA, sourcedomain.LexicalQuery{Text: "w", Kinds: kinds, Limit: 5})
			return err
		},
		"GetNamedSections": func() error {
			_, err := q.GetNamedSections(ctx, vaultA, sourcedomain.LexicalQuery{Text: "w", Kinds: kinds, Limit: 5})
			return err
		},
		"FindNearest": func() error {
			_, err := q.FindNearest(ctx, vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 5, Floor: -1})
			return err
		},
		"GetUnchunkedSources": func() error { _, err := q.GetUnchunkedSources(ctx, vaultA, "book", 5); return err },
		"GetSourcesWithOtherRecipe": func() error {
			_, err := q.GetSourcesWithOtherRecipe(ctx, vaultA, "book", []string{"r"}, 5)
			return err
		},
		"GetUnembeddedChunks": func() error { _, err := q.GetUnembeddedChunks(ctx, vaultA, recipe, chunk.Cursor{}, 5); return err },
		"GetProgress":         func() error { _, _, err := q.GetProgress(ctx, vaultA, recipe); return err },
		"GetRecognizedSources": func() error {
			_, err := q.GetRecognizedSources(ctx, vaultA, "book")
			return err
		},
	}
	for name, check := range checks {
		if err := check(); err == nil {
			t.Errorf("%s answered although the vault could not be looked up", name)
		}
	}

	p, found, err := q.GetRecord(ctx, vaultA, 1)
	if err == nil || found || p != (chunk.Record{}) {
		t.Errorf("GetRecord answered %+v, %v, %v", p, found, err)
	}
	text, found, err := q.GetSourceText(ctx, vaultA, "p")
	if err == nil || found || text != (chunk.SourceText{}) {
		t.Errorf("GetSourceText answered %+v, %v, %v", text, found, err)
	}
	target, embedded, err := q.GetProgress(ctx, vaultA, recipe)
	if err == nil || target != 0 || embedded != 0 {
		t.Errorf("GetProgress answered %d, %d, %v", target, embedded, err)
	}
}

func TestACursorOfOneIsAPositionAndNotTheStart(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/a.epub", mtime: century, large: "l", smalls: []string{"a1"}}.save(t, db)
	for _, cursor := range [][3]int64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} {
		got, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{MTime: cursor[0], Source: cursor[1], Chunk: cursor[2]}, 10)
		if err != nil || len(got) != 0 {
			t.Errorf("cursor %v needs %d, %v", cursor, len(got), err)
		}
	}
}

func TestASourceModifiedAtTheLatestInstantIsStillWalked(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/last.epub", mtime: math.MaxInt64, large: "l", smalls: []string{"x"}}.save(t, db)
	got, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), vaultA, recipe, chunk.Cursor{}, 10)
	if err != nil || len(got) != 1 {
		t.Errorf("a walk from the start needs %d, %v", len(got), err)
	}
}

func TestASearchByMeaningThatNothingReachesAnswersWithNoChunks(t *testing.T) {
	got, err := queriesOf(meaningFixture(t)).FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 5, Floor: 2})
	if err != nil || got != nil {
		t.Errorf("a floor nothing reaches answered %v, %v", got, err)
	}
}

func TestAChunkWhoseOffsetIsNoNumberIsACorruptRow(t *testing.T) {
	queries := broken(t, func(db *index.DB) {
		source{path: "lib/a.epub", large: "t", smalls: []string{"t"}}.save(t, db)
		embed(t, db, vaultA, "lib/a.epub", 0xF0)
	}, `UPDATE chunks SET start = 'abc'`)
	got, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 5, Floor: -1})
	if err == nil || got != nil {
		t.Errorf("answered %v, %v", got, err)
	}
}
