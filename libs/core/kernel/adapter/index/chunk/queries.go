package chunk

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
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
func (q *Queries) FindLexical(ctx context.Context, vaultID domain.VaultID, query source.LexicalQuery) ([]domain.Passage, error) {
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
func (q *Queries) GetNamedPassages(ctx context.Context, vaultID domain.VaultID, query source.LexicalQuery) ([]domain.Passage, error) {
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

func (q *Queries) findPassages(ctx context.Context, vaultID domain.VaultID, query source.LexicalQuery, search passageSearch) (_ []domain.Passage, err error) {
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
func (q *Queries) FindNearest(ctx context.Context, vaultID domain.VaultID, search source.NearestQuery) ([]domain.Passage, error) {
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

