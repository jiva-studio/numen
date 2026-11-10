package chunk

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
)

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
