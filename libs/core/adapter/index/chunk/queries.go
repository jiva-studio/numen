package chunk

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"math"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/vector"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"golang.org/x/text/unicode/norm"
)

// Queries answers questions about chunks in shapes other than chunks: where a
// passage is read from, what matched a search, and which chunks lack a vector.
type Queries struct {
	db      *sql.DB
	minimum *Minimum
}

func NewQueries(db *sql.DB) *Queries { return &Queries{db: db, minimum: &Minimum{}} }

// NewQueriesWithMinimum creates queries that leave out the small chunks shorter than minimum, which
// carry no vector.
func NewQueriesWithMinimum(db *sql.DB, minimum *Minimum) *Queries {
	return &Queries{db: db, minimum: minimum}
}

// SetMinimum sets the shortest length of a chunk that gets a vector. The queries leave out the
// chunks that are shorter, as the count of chunks that can carry a vector does.
func (q *Queries) SetMinimum(length int64) { q.minimum.Swap(length) }

// Passage is one chunk, and where its text is read from.
//
// chunk.
type Passage struct {
	Chunk       int64
	Path        string
	Start       int
	Length      int
	Location    string
	Parent      int64
	Producer    string
	ContentHash string
	ChunkHash   string
	MTime       int64
	Source      int64
}

// ID is how the core addresses the chunk on a row. This is the one place a
// chunk's identity is spelled, and the core carries it back untouched.
func ID(row int64) domain.ChunkID {
	return domain.ChunkID(strconv.FormatInt(row, 10))
}

// Row is the chunk an identifier names, and an error where it names none this
// index handed out.
func Row(id domain.ChunkID) (int64, error) {
	row, err := strconv.ParseInt(string(id), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is no chunk of this index", id)
	}
	return row, nil
}

// GetFileStats is what the index believes about each file of one kind, keyed by
// path, so a scan can decide what to read again without opening anything.
func (q *Queries) GetFileStats(ctx context.Context, vaultID domain.VaultID, kind string) (_ map[string]domain.FileStat, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return map[string]domain.FileStat{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, fileStatsSQL, vault, kind)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	out := map[string]domain.FileStat{}
	for rows.Next() {
		var ref domain.FileStat
		var mtime int64
		if err := rows.Scan(&ref.Path, &ref.Size, &mtime); err != nil {
			return nil, err
		}
		ref.ModTime = ConvertNanosToTime(mtime)
		out[ref.Path] = ref
	}
	return out, rows.Err()
}

// GetSourcesUnder is every source the vault holds at a path and beneath it, by
// path: the one file, or everything a folder holds.
func (q *Queries) GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) (_ []domain.FileStat, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	first, past := getRangeUnder(path)
	rows, err := q.db.QueryContext(ctx, fileStatsUnderSQL, vault, path, vault, first, past)
	if err != nil {
		return nil, fmt.Errorf("what the vault holds at %s and under it: %w", path, err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []domain.FileStat
	for rows.Next() {
		var ref domain.FileStat
		var mtime int64
		if err := rows.Scan(&ref.Path, &ref.Kind, &ref.Size, &mtime); err != nil {
			return nil, err
		}
		ref.ModTime = ConvertNanosToTime(mtime)
		out = append(out, ref)
	}
	return out, rows.Err()
}

// FindLexical is a word search: the chunks of one vault whose text matches the
// query, best first.
//
// The result is the large chunk enclosing each hit, which is what a result
// displays.
func (q *Queries) FindLexical(ctx context.Context, vaultID domain.VaultID, query sourceport.LexicalQuery) ([]domain.Passage, error) {
	if query.Limit <= 0 {
		// How many candidates to keep is a retrieval decision. The caller makes
		// it, and arriving here without one is a mistake in the caller.
		return nil, fmt.Errorf("a search by words needs a positive limit, got %d", query.Limit)
	}
	return q.findPassages(ctx, vaultID, query, passageSearch{all: lexicalSQL, kinds: lexicalKindsSQL})
}

// GetNamedPassages returns the sections of one vault whose names match the
// query, best first.
//
// A section is represented by the chunk it opens, so a result is the chapter
// about the query and not the paragraph that repeats the name most often.
func (q *Queries) GetNamedPassages(ctx context.Context, vaultID domain.VaultID, query sourceport.LexicalQuery) ([]domain.Passage, error) {
	if query.Limit <= 0 {
		return nil, fmt.Errorf("a search by name needs a positive limit, got %d", query.Limit)
	}
	return q.findPassages(ctx, vaultID, query, passageSearch{all: sectionsSQL, kinds: sectionsKindsSQL})
}

// passageSearch is a full-text statement over every source, and its variant
// limited to sources of some kinds.
type passageSearch struct {
	all   string
	kinds string
}

func (q *Queries) findPassages(ctx context.Context, vaultID domain.VaultID, query sourceport.LexicalQuery, search passageSearch) (_ []domain.Passage, err error) {
	expression := Expression(query.Text, query.IsGrowing)
	if expression == "" {
		return nil, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	composed, decomposed := norm.NFC.String(query.PathPrefix), norm.NFD.String(query.PathPrefix)
	if len(query.Kinds) == 0 {
		rows, err = q.db.QueryContext(ctx, search.all, expression, vault, query.Limit, composed, decomposed)
	} else {
		kindFilter := formatKinds(query.Kinds)
		rows, err = q.db.QueryContext(ctx, search.kinds, expression, vault, kindFilter, query.Limit, composed, decomposed)
	}
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []domain.Passage
	for rows.Next() {
		var p domain.Passage
		var row int64
		if err := rows.Scan(&row, &p.Source, &p.Kind, &p.Producer, &p.ContentHash, &p.Start, &p.Length, &p.Location, &p.HitAt); err != nil {
			return nil, err
		}
		p.ChunkID = ID(row)
		out = append(out, p)
	}
	return out, rows.Err()
}

// FindNearest returns the chunks of one vault nearest a query vector, nearest
// first, at most `search.Limit` of them.
// The coarse pass over the bit vectors keeps several times `search.Limit`
// candidates and the full-precision vectors order them.
// A chunk below the similarity floor is not a result.
func (q *Queries) FindNearest(ctx context.Context, vaultID domain.VaultID, search sourceport.NearestQuery) ([]domain.Passage, error) {
	if search.Limit <= 0 {
		return nil, fmt.Errorf("a search by meaning needs a positive limit, got %d", search.Limit)
	}
	if len(search.Vector) == 0 {
		return nil, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	item, err := q.findNearestWithPrefix(ctx, vault, search.Vector, search.Limit*coarseCandidates, search.PathPrefix)
	if err != nil {
		if q.hasOtherCoarseRecipe(ctx, search.Recipe) {
			return nil, nil
		}
		return nil, err
	}
	ranked, err := q.rerank(ctx, search.Recipe, search.Vector, item, search.Kinds, search.Floor)
	if err != nil {
		return nil, err
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="the mutated test is >= : cutting a list of exactly limit entries to limit entries leaves it unchanged"
	if len(ranked) > search.Limit {
		ranked = ranked[:search.Limit]
	}
	return q.getEnclosingPassages(ctx, vault, ranked)
}

// GetBestSimilarity returns the cosine similarity of the chunk of one vault
// nearest a query vector, found by the coarse pass and the full-precision
// rerank that FindNearest runs, and zero where the vault has none.
func (q *Queries) GetBestSimilarity(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32) (float64, error) {
	if len(query) == 0 {
		return 0, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	item, err := q.findNearestByBits(ctx, vault, query, coarseCandidates)
	if err != nil {
		if q.hasOtherCoarseRecipe(ctx, recipe) {
			return 0, nil
		}
		return 0, err
	}
	ranked, err := q.rankCandidates(ctx, recipe, query, item, nil, math.Inf(-1))
	if err != nil {
		return 0, err
	}
	if len(ranked) == 0 {
		return 0, nil
	}
	return ranked[0].similarity, nil
}

func (q *Queries) hasOtherCoarseRecipe(ctx context.Context, recipe string) bool {
	var currentRecipe string
	err := q.db.QueryRowContext(ctx, `SELECT recipe FROM coarse_recipe WHERE id = 1`).Scan(&currentRecipe)
	return err == nil && currentRecipe != "" && currentRecipe != recipe
}

func (q *Queries) findNearestWithPrefix(ctx context.Context, vault int64, query []float32, k int, pathPrefix string) (_ []int64, err error) {
	if pathPrefix == "" {
		return q.findNearestByBits(ctx, vault, query, k)
	}
	ctx, err = sqlwrite.Uninterrupted(ctx)
	if err != nil {
		return nil, err
	}
	composed, decomposed := norm.NFC.String(pathPrefix), norm.NFD.String(pathPrefix)
	rows, err := q.db.QueryContext(ctx, coarsePrefixSQL, vector.Bits(query), vault, composed, k, getUpperBound(composed),
		decomposed, getUpperBound(decomposed))
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var kept []int64
	for rows.Next() {
		var chunk int64
		if err := rows.Scan(&chunk); err != nil {
			return nil, err
		}
		kept = append(kept, chunk)
	}
	return kept, rows.Err()
}

func getUpperBound(prefix string) any {
	end := []byte(prefix)
	for len(end) > 0 && end[len(end)-1] == 0xFF {
		end = end[:len(end)-1]
	}
	if len(end) == 0 {
		return []byte{0xFF}
	}
	end[len(end)-1]++
	return string(end)
}

func (q *Queries) findNearestByBits(ctx context.Context, vault int64, query []float32, k int) (_ []int64, err error) {
	ctx, err = sqlwrite.Uninterrupted(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, coarseSQL, vector.Bits(query), vault, k)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var val []int64
	for rows.Next() {
		var chunk int64
		var distance float64
		if err := rows.Scan(&chunk, &distance); err != nil {
			return nil, err
		}
		val = append(val, chunk)
	}
	return val, rows.Err()
}

func (q *Queries) getEnclosingPassages(ctx context.Context, vault int64, chunks []int64) (_ []domain.Passage, err error) {
	if len(chunks) == 0 {
		return nil, nil
	}
	enclosing, err := q.db.PrepareContext(ctx, enclosingSQL)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, enclosing.Close()) }()

	out := make([]domain.Passage, 0, len(chunks))
	for _, chunk := range chunks {
		p := domain.Passage{ChunkID: ID(chunk)}
		err := enclosing.QueryRowContext(ctx, chunk, vault).
			Scan(&p.Source, &p.Kind, &p.Producer, &p.ContentHash, &p.Start, &p.Length, &p.Location, &p.HitAt)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Passage is where one chunk's text is read from. False when the vault holds no
// such chunk.
func (q *Queries) Passage(ctx context.Context, vaultID domain.VaultID, chunk int64) (Passage, bool, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return Passage{}, false, nil
	}
	if err != nil {
		return Passage{}, false, err
	}
	p, found, err := scanPassage(q.db.QueryRowContext(ctx, passageSQL, chunk, vault), chunk)
	return p, found, err
}

func scanPassage(row *sql.Row, chunk int64) (Passage, bool, error) {
	p := Passage{Chunk: chunk}
	err := row.Scan(&p.Path, &p.Producer, &p.ContentHash, &p.Start, &p.Length, &p.Location, &p.Parent)
	if errors.Is(err, sql.ErrNoRows) {
		return Passage{}, false, nil
	}
	if err != nil {
		return Passage{}, false, err
	}
	return p, true, nil
}

// GetUnchunkedSources is the sources of one kind with no small chunk: the file
// changed, or it has never been cut.
func (q *Queries) GetUnchunkedSources(ctx context.Context, vaultID domain.VaultID, kind string, limit int) ([]string, error) {
	return q.paths(ctx, vaultID, "unchunked", unchunkedSQL, limit, func(vault int64) []any {
		return []any{vault, kind, limit}
	})
}

// GetSourcesWithOtherRecipe is the sources of one kind whose text was not extracted by any
// of the recipes now in use.
func (q *Queries) GetSourcesWithOtherRecipe(ctx context.Context, vaultID domain.VaultID, kind string, recipes []string, limit int) ([]string, error) {
	named, err := json.Marshal(recipes)
	if err != nil {
		return nil, err
	}
	return q.paths(ctx, vaultID, "stale_recipe", staleRecipeSQL, limit, func(vault int64) []any {
		return []any{vault, kind, string(named), limit}
	})
}

// Cursor is the point in the ordering of unembedded chunks that a page begins after.
type Cursor struct {
	MTime, Source, Chunk int64
}

// GetUnembeddedChunks is the small chunks of a vault with no vector from the
// model in use, ordered so recently modified sources are embedded first.
func (q *Queries) GetUnembeddedChunks(ctx context.Context, vaultID domain.VaultID, recipe string, after Cursor, limit int) (_ []Passage, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a batch needs a positive limit, got %d", limit)
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if after.MTime <= 0 && after.Source <= 0 && after.Chunk <= 0 {
		after.MTime = math.MaxInt64
		after.Source = math.MaxInt64
	}
	ctx, err = sqlwrite.Uninterrupted(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, unembeddedSQL, recipe, vault, after.MTime, after.MTime, after.Source, after.Source, after.Chunk, q.minimum.Get(), limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []Passage
	for rows.Next() {
		var p Passage
		if err := rows.Scan(&p.Chunk, &p.Path, &p.Producer, &p.ContentHash, &p.Start, &p.Length, &p.Location, &p.Parent, &p.ChunkHash, &p.MTime, &p.Source); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (q *Queries) paths(ctx context.Context, vaultID domain.VaultID, name, query string, limit int, args func(vault int64) []any) (_ []string, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s needs a positive limit, got %d", name, limit)
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, query, args(vault)...)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

// GetProgress returns how many of a vault's chunks can carry a vector and how
// many of those carry one for the named model. Cutting finishes long before
// embedding, so the pair measures the remaining work.
//
// An unknown vault has no chunks and no remaining work.
func (q *Queries) GetProgress(ctx context.Context, vaultID domain.VaultID, recipe string) (target, embedded int64, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	ctx, err = sqlwrite.Uninterrupted(ctx)
	if err != nil {
		return 0, 0, err
	}
	err = q.db.QueryRowContext(ctx, progressSQL, recipe, vault, q.minimum.Get()).
		Scan(&target, &embedded)
	return target, embedded, err
}

// GetKeptVectors is the vectors already made for the texts given under the
// recipe given, by the hex of their hash.
//
// A vector that is returned was paid for once, and asking a model for it again
// is buying what is already here.
func (q *Queries) GetKeptVectors(ctx context.Context, recipe string, of [][]byte) (_ map[string][]byte, err error) {
	if len(of) == 0 {
		return nil, nil
	}
	wanted := make([]string, 0, len(of))
	for _, one := range of {
		if len(one) == 0 {
			continue
		}
		wanted = append(wanted, hex.EncodeToString(one))
	}
	if len(wanted) == 0 {
		return nil, nil
	}
	request, err := json.Marshal(wanted)
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, keptVectorsSQL, string(request), recipe)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	out := make(map[string][]byte, len(wanted))
	for rows.Next() {
		var hash string
		var v []byte
		if err := rows.Scan(&hash, &v); err != nil {
			return nil, err
		}
		out[strings.ToLower(hash)] = v
	}
	return out, rows.Err()
}

// SourceText is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that recognition are kept under.
type SourceText struct {
	Path        string
	Producer    string
	ContentHash string
	Size        int64
	MTime       int64
}

// GetSourceText is what one source's text came from, and false where the index holds
// no source at that path.
func (q *Queries) GetSourceText(ctx context.Context, vaultID domain.VaultID, path string) (SourceText, bool, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return SourceText{}, false, nil
	}
	if err != nil {
		return SourceText{}, false, err
	}
	found := SourceText{Path: path}
	err = q.db.QueryRowContext(ctx, sourceTextSQL, vault, path).Scan(&found.Producer, &found.ContentHash, &found.Size, &found.MTime)
	if errors.Is(err, sql.ErrNoRows) {
		return SourceText{}, false, nil
	}
	if err != nil {
		return SourceText{}, false, fmt.Errorf("reading %s: %w", path, err)
	}
	return found, true, nil
}

// GetRecognisedSources returns, by path, the sources of one kind whose text a
// producer created and not read from the file's own bytes.
// A scan uses it to find sources whose files are gone: the store is a folder
// the user may empty.
// search results.
func (q *Queries) GetRecognisedSources(ctx context.Context, vaultID domain.VaultID, kind string) (_ []SourceText, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, recognisedSQL, vault, kind)
	if err != nil {
		return nil, fmt.Errorf("what a model made the text of: %w", err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []SourceText
	for rows.Next() {
		var r SourceText
		if err := rows.Scan(&r.Path, &r.Producer, &r.ContentHash); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
