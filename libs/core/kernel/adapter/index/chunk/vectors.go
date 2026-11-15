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
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Vectors answers which chunks lack a vector, how far the embedding has come, which vectors exist already and what
// the index holds of a vault.
// The chunks shorter than the minimum carry no vector and are left out.
type Vectors struct {
	db      *sql.DB
	minimum *Minimum
}

// NewVectors creates the queries about vectors over the read pool, leaving out the chunks shorter than minimum.
func NewVectors(db *sql.DB, minimum *Minimum) *Vectors { return &Vectors{db: db, minimum: minimum} }

// SetMinimum sets the shortest length of a chunk that gets a vector. The queries leave out the
// chunks that are shorter, as the count of chunks that can carry a vector does.
func (q *Vectors) SetMinimum(length int64) { q.minimum.Swap(length) }

// Cursor is the point in the ordering of unembedded chunks that a page begins after.
type Cursor struct {
	MTime, Source, Chunk int64
}

// GetUnembeddedChunks is the small chunks of a vault with no vector from the
// model in use, ordered so recently modified sources are embedded first.
func (q *Vectors) GetUnembeddedChunks(ctx context.Context, vaultID domain.VaultID, recipe string, after Cursor, limit int) (_ []Record, err error) {
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

	var out []Record
	for rows.Next() {
		var p Record
		if err := rows.Scan(&p.Chunk, &p.Path, &p.Producer, &p.ContentHash, &p.Start, &p.Length, &p.Location, &p.Parent, &p.ChunkHash, &p.MTime, &p.Source); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProgress returns how many of a vault's chunks can carry a vector and how
// many of those carry one for the named model. Cutting finishes long before
// embedding, so the pair measures the remaining work.
//
// An unknown vault has no chunks and no remaining work.
func (q *Vectors) GetProgress(ctx context.Context, vaultID domain.VaultID, recipe string) (target, embedded int64, err error) {
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
func (q *Vectors) GetKeptVectors(ctx context.Context, recipe string, of [][]byte) (_ map[string][]byte, err error) {
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
