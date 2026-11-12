package chunk

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
	"golang.org/x/text/unicode/norm"
)

// Search finds the chunks of a vault that match words or a meaning.
type Search struct{ db *sql.DB }

// NewSearch creates the search over the read pool.
func NewSearch(db *sql.DB) *Search { return &Search{db: db} }

// FindLexical is a word search: the chunks of one vault whose text matches the
// query, best first.
//
// The result is the large chunk enclosing each hit, which is what a result
// displays.
func (q *Search) FindLexical(ctx context.Context, vaultID domain.VaultID, query source.LexicalQuery) ([]domain.Passage, error) {
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
func (q *Search) GetNamedPassages(ctx context.Context, vaultID domain.VaultID, query source.LexicalQuery) ([]domain.Passage, error) {
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

func (q *Search) findPassages(ctx context.Context, vaultID domain.VaultID, query source.LexicalQuery, search passageSearch) (_ []domain.Passage, err error) {
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

// FindCoarseCandidates returns the chunks of one vault whose one-bit vectors are nearest the query
// vector, nearest first. It fails with source.ErrOtherRecipe where the failure comes from vectors
// kept under another recipe.
func (q *Search) FindCoarseCandidates(ctx context.Context, vaultID domain.VaultID, query source.CoarseQuery) ([]domain.ChunkID, error) {
	if len(query.Vector) == 0 {
		return nil, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.findNearestWithPrefix(ctx, vault, query.Vector, query.Limit, query.PathPrefix)
	if err != nil {
		if q.hasOtherCoarseRecipe(ctx, query.Recipe) {
			return nil, fmt.Errorf("%w: %w", source.ErrOtherRecipe, err)
		}
		return nil, err
	}
	ids := make([]domain.ChunkID, len(rows))
	for i, row := range rows {
		ids[i] = ID(row)
	}
	return ids, nil
}

func (q *Search) hasOtherCoarseRecipe(ctx context.Context, recipe string) bool {
	var currentRecipe string
	err := q.db.QueryRowContext(ctx, `SELECT recipe FROM coarse_recipe WHERE id = 1`).Scan(&currentRecipe)
	return err == nil && currentRecipe != "" && currentRecipe != recipe
}

func (q *Search) findNearestWithPrefix(ctx context.Context, vault int64, query []float32, k int, pathPrefix string) (_ []int64, err error) {
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

func (q *Search) findNearestByBits(ctx context.Context, vault int64, query []float32, k int) (_ []int64, err error) {
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

// GetEnclosingPassages returns the passage that encloses each chunk of one vault, in the order of
// the chunks, leaving out a chunk the vault does not hold.
func (q *Search) GetEnclosingPassages(ctx context.Context, vaultID domain.VaultID, ids []domain.ChunkID) (_ []domain.Passage, err error) {
	if len(ids) == 0 {
		return nil, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	enclosing, err := q.db.PrepareContext(ctx, enclosingSQL)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, enclosing.Close()) }()

	out := make([]domain.Passage, 0, len(ids))
	for _, id := range ids {
		chunk, err := Row(id)
		if err != nil {
			return nil, err
		}
		p := domain.Passage{ChunkID: id}
		err = enclosing.QueryRowContext(ctx, chunk, vault).
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
