package chunk

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/embedding"
)

// Queries answers questions about chunks in shapes that are not chunks: where a
// passage is read from, what matched a search, what still owes work.
type Queries struct{ db *sql.DB }

func NewQueries(db *sql.DB) *Queries { return &Queries{db: db} }

// Passage is one chunk, and where its text is read from.
//
// `Parent` is the large chunk this one sits inside, and is zero for a large
// chunk.
type Passage struct {
	Chunk    int64
	Path     string
	Start    int
	Length   int
	Location string
	Parent   int64
	// TextFrom names the producer of the text this chunk is a place in. Empty
	// where the source's own bytes are the text, which is the ordinary case.
	TextFrom string
	// Hash addresses the content of the source, and is what the files of a
	// reading of it are kept under.
	Hash string
	// Fingerprint is the text this chunk holds, as the index recorded it.
	Fingerprint string
}

// Fingerprints is what the index believes about each file of one kind, keyed by
// path, so a scan can decide what to read again without opening anything.
func (q *Queries) Fingerprints(ctx context.Context, vaultID, kind string) (map[string]domain.Fingerprint, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return map[string]domain.Fingerprint{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("fingerprints"), vault, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]domain.Fingerprint{}
	for rows.Next() {
		var ref domain.Fingerprint
		if err := rows.Scan(&ref.Path, &ref.Size, &ref.MTime); err != nil {
			return nil, err
		}
		out[ref.Path] = ref
	}
	return out, rows.Err()
}

// Under is every source the vault holds at a path and beneath it, by path: the
// one file, or everything a folder holds.
func (q *Queries) Under(ctx context.Context, vaultID, path string) ([]domain.Fingerprint, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	first, past := under(path)
	rows, err := q.db.QueryContext(ctx, stmt.Get("sources_under"), vault, path, vault, first, past)
	if err != nil {
		return nil, fmt.Errorf("sources_under: %w", err)
	}
	defer rows.Close()

	var out []domain.Fingerprint
	for rows.Next() {
		var ref domain.Fingerprint
		if err := rows.Scan(&ref.Path, &ref.Kind, &ref.Size, &ref.MTime); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

// kinds is what a caller's chosen kinds are on the wire: a JSON array, empty
// for a question that says nothing about what sort of file it wants.
func kinds(chosen []domain.SourceKind) (string, error) {
	out := make([]string, 0, len(chosen))
	for _, k := range chosen {
		out = append(out, string(k))
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Lexical is a search asked by words: the chunks of one vault whose text
// matches what was typed, best first.
//
// What comes back is the large chunk enclosing each hit, which is what a
// result shows.
func (q *Queries) Lexical(ctx context.Context, vaultID, query string, of []domain.SourceKind, limit int, growing bool) ([]domain.Passage, error) {
	if limit <= 0 {
		// How many candidates to keep is a retrieval decision. The caller makes
		// it, and arriving here without one is a mistake in the caller.
		return nil, fmt.Errorf("a search by words needs a positive limit, got %d", limit)
	}
	expression := Expression(query, growing)
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

	wanted, err := kinds(of)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, stmt.Get("lexical"), expression, vault, wanted, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Passage
	for rows.Next() {
		var p domain.Passage
		if err := rows.Scan(&p.Chunk, &p.Source, &p.Kind, &p.TextFrom, &p.Hash, &p.Start, &p.Length, &p.Location, &p.HitAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Named is the sections of one vault whose names match what was typed, best
// first.
//
// A section answers with the chunk it opens, so what comes back stands where the
// section begins. Asked where a book speaks about a thing, this is the half that
// answers with the chapter about it and not with the paragraph that says its
// name most often.
func (q *Queries) Named(ctx context.Context, vaultID, query string, of []domain.SourceKind, limit int, growing bool) ([]domain.Passage, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a search by name needs a positive limit, got %d", limit)
	}
	expression := Expression(query, growing)
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

	wanted, err := kinds(of)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, stmt.Get("sections"), expression, vault, wanted, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Passage
	for rows.Next() {
		var p domain.Passage
		if err := rows.Scan(&p.Chunk, &p.Source, &p.Kind, &p.TextFrom, &p.Hash, &p.Start, &p.Length, &p.Location, &p.HitAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Nearest is a search asked by meaning: the chunks of one vault nearest a
// query vector, nearest first, at most `limit` of them.
//
// The coarse pass over the bit vectors keeps several times that many, and the
// full-precision vectors order what it kept. A chunk that does not reach the
// similarity floor is not an answer, so a vault with nothing to say answers
// with nothing.
func (q *Queries) Nearest(ctx context.Context, vaultID, recipe string, query []float32, of []domain.SourceKind, limit int, floor float64) ([]domain.Passage, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a search by meaning needs a positive limit, got %d", limit)
	}
	if len(query) == 0 {
		return nil, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	near, err := q.coarse(ctx, vault, query, limit*coarseCandidates)
	if err != nil {
		return nil, err
	}
	ranked, err := q.rerank(ctx, recipe, query, near, of, floor)
	if err != nil {
		return nil, err
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return q.enclosing(ctx, vault, ranked)
}

// coarse is the pass over the bit vectors: the chunks of one vault whose signs
// stand nearest the query's, by Hamming distance, k of them.
func (q *Queries) coarse(ctx context.Context, vault int64, query []float32, k int) ([]int64, error) {
	rows, err := q.db.QueryContext(ctx, stmt.Get("search"), embedding.Bits(query), vault, k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var near []int64
	for rows.Next() {
		var chunk int64
		var distance float64
		if err := rows.Scan(&chunk, &distance); err != nil {
			return nil, err
		}
		near = append(near, chunk)
	}
	return near, rows.Err()
}

// enclosing is the large chunk each chunk sits inside, in the order given.
//
// The nearest-neighbour question is asked of the vector index alone: it takes
// its own ordering and does not join. Where each answer is read from is a
// second question, asked once per answer.
func (q *Queries) enclosing(ctx context.Context, vault int64, chunks []int64) ([]domain.Passage, error) {
	if len(chunks) == 0 {
		return nil, nil
	}
	enclosing, err := q.db.PrepareContext(ctx, stmt.Get("enclosing"))
	if err != nil {
		return nil, err
	}
	defer enclosing.Close()

	out := make([]domain.Passage, 0, len(chunks))
	for _, chunk := range chunks {
		p := domain.Passage{Chunk: chunk}
		err := enclosing.QueryRowContext(ctx, chunk, vault).
			Scan(&p.Source, &p.Kind, &p.TextFrom, &p.Hash, &p.Start, &p.Length, &p.Location, &p.HitAt)
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
func (q *Queries) Passage(ctx context.Context, vaultID string, chunk int64) (Passage, bool, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return Passage{}, false, nil
	}
	if err != nil {
		return Passage{}, false, err
	}
	p, found, err := scanPassage(q.db.QueryRowContext(ctx, stmt.Get("passage"), chunk, vault), chunk)
	return p, found, err
}

func scanPassage(row *sql.Row, chunk int64) (Passage, bool, error) {
	p := Passage{Chunk: chunk}
	err := row.Scan(&p.Path, &p.TextFrom, &p.Hash, &p.Start, &p.Length, &p.Location, &p.Parent)
	if errors.Is(err, sql.ErrNoRows) {
		return Passage{}, false, nil
	}
	if err != nil {
		return Passage{}, false, err
	}
	return p, true, nil
}

// Unchunked is the sources of one kind with no small chunk: the file changed,
// or it has never been cut.
func (q *Queries) Unchunked(ctx context.Context, vaultID, kind string, limit int) ([]string, error) {
	return q.paths(ctx, vaultID, "unchunked", limit, func(vault int64) []any {
		return []any{vault, kind, limit}
	})
}

// ByOtherRecipe is the sources of one kind whose text was not extracted by any
// of the recipes now in use.
func (q *Queries) ByOtherRecipe(ctx context.Context, vaultID, kind string, recipes []string, limit int) ([]string, error) {
	named, err := json.Marshal(recipes)
	if err != nil {
		return nil, err
	}
	return q.paths(ctx, vaultID, "stale_recipe", limit, func(vault int64) []any {
		return []any{vault, kind, string(named), limit}
	})
}

// Unembedded is the small chunks of a vault with no vector from the model in
// use, from `after` onwards. Asked with the last id of the previous answer, it
// resumes.
func (q *Queries) Unembedded(ctx context.Context, vaultID, recipe string, after int64, limit int) ([]Passage, error) {
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

	rows, err := q.db.QueryContext(ctx, stmt.Get("unembedded"), recipe, vault, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Passage
	for rows.Next() {
		var p Passage
		if err := rows.Scan(&p.Chunk, &p.Path, &p.TextFrom, &p.Hash, &p.Start, &p.Length, &p.Location, &p.Parent, &p.Fingerprint); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// paths answers the questions that come back as a list of paths in one vault.
func (q *Queries) paths(ctx context.Context, vaultID, statement string, limit int, args func(vault int64) []any) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s needs a positive limit, got %d", statement, limit)
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get(statement), args(vault)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// Progress is how many of a vault's chunks can carry a vector and how many of
// those carry one for the model named. Cutting finishes long before embedding
// does, so the pair is what says how far there is to go.
//
// A vault the index has never heard of has nothing and owes nothing.
func (q *Queries) Progress(ctx context.Context, vaultID, recipe string) (held, embedded int64, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	err = q.db.QueryRowContext(ctx, stmt.Get("progress"), recipe, vault).
		Scan(&held, &embedded)
	return held, embedded, err
}

// Kept is the vectors already made for the texts given under the recipe given,
// by the hex of their fingerprint.
//
// A vector that comes back was paid for once, and asking a model for it again
// is buying what is already here.
func (q *Queries) Kept(ctx context.Context, recipe string, of [][]byte) (map[string][]byte, error) {
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
	asked, err := json.Marshal(wanted)
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("kept_vectors"), string(asked), recipe)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]byte, len(wanted))
	for rows.Next() {
		var fingerprint string
		var v []byte
		if err := rows.Scan(&fingerprint, &v); err != nil {
			return nil, err
		}
		out[strings.ToLower(fingerprint)] = v
	}
	return out, rows.Err()
}

// SourceText is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that reading are kept under.
type SourceText struct {
	Path     string
	Producer string
	Hash     string
	Size     int64
	MTime    int64
}

// Reading is what one source's text came from, and false where the index holds
// no source at that path.
func (q *Queries) Reading(ctx context.Context, vaultID, path string) (SourceText, bool, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return SourceText{}, false, nil
	}
	if err != nil {
		return SourceText{}, false, err
	}
	found := SourceText{Path: path}
	err = q.db.QueryRowContext(ctx, stmt.Get("reading"), vault, path).Scan(&found.Producer, &found.Hash, &found.Size, &found.MTime)
	if errors.Is(err, sql.ErrNoRows) {
		return SourceText{}, false, nil
	}
	if err != nil {
		return SourceText{}, false, fmt.Errorf("reading %s: %w", path, err)
	}
	return found, true, nil
}

// Recognised is the sources of one kind whose text a producer made rather than
// their own bytes, by path.
//
// A scan asks it in order to find the ones whose files are gone: the store is a
// folder on the person's disk and they may empty it, and a source standing on
// files that are not there answers a search with nothing.
func (q *Queries) Recognised(ctx context.Context, vaultID, kind string) ([]SourceText, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, stmt.Get("recognised"), vault, kind)
	if err != nil {
		return nil, fmt.Errorf("recognised: %w", err)
	}
	defer rows.Close()

	var out []SourceText
	for rows.Next() {
		var r SourceText
		if err := rows.Scan(&r.Path, &r.Producer, &r.Hash); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
