package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
)

// coarseWidth is the width the vector index is created at, and is what an
// installation with no model leaves it as.
const coarseWidth = 1024

// declaredWidth reads the width out of the statement the table was created by,
// which is where a virtual table keeps its shape.
var declaredWidth = regexp.MustCompile(`bit\[(\d+)]`)

// FitVectors makes the coarse index hold vectors of the width given, and fills
// it from what the recipe has already bought.
//
// The coarse index is built for one width, and a model of another width cannot
// be written to it. What a model made is kept by the text it read, in a table of
// its own, and the coarse form is read out of that.
func (db *DB) FitVectors(ctx context.Context, dims int, recipe string) error {
	if dims <= 0 {
		return fmt.Errorf("a vector of %d dimensions is not a vector", dims)
	}
	held, err := vectorWidth(ctx, db.write)
	if err != nil {
		return err
	}
	if held == dims {
		return nil
	}

	tx, err := db.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	for _, statement := range []string{
		`DROP TABLE IF EXISTS chunks_vec`,
		fmt.Sprintf(`CREATE VIRTUAL TABLE chunks_vec USING vec0 (
			chunk_id  integer primary key,
			vault_id  integer,
			embedding bit[%d]
		)`, dims),
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("fit the vector index to %d dimensions: %w", dims, err)
		}
	}
	if err := fillCoarse(ctx, tx, dims, recipe); err != nil {
		return err
	}
	return tx.Commit()
}

// fillCoarse writes a coarse row for every vector the recipe holds of the width
// given. A vector of another width belongs to another model and is left where
// it is.
func fillCoarse(ctx context.Context, tx *sql.Tx, dims int, recipe string) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT c.id, c.vault_id, v.v
		 FROM vectors v JOIN chunks c ON unhex(c.hash) = v.fingerprint
		 WHERE v.recipe = ? AND length(v.v) = ?`, recipe, dims)
	if err != nil {
		return fmt.Errorf("what the recipe has bought: %w", err)
	}
	defer rows.Close()

	type coarse struct {
		chunk, vault int64
		bits         []byte
	}
	var held []coarse
	for rows.Next() {
		var one coarse
		var value []byte
		if err := rows.Scan(&one.chunk, &one.vault, &value); err != nil {
			return fmt.Errorf("what the recipe has bought: %w", err)
		}
		one.bits = embedding.Coarse(quantised(value))
		held = append(held, one)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("what the recipe has bought: %w", err)
	}

	// `vec_bit` says the blob is one bit per dimension. Its length alone does
	// not distinguish that from float32.
	for _, one := range held {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chunks_vec (chunk_id, vault_id, embedding) VALUES (?, ?, vec_bit(?))`,
			one.chunk, one.vault, one.bits); err != nil {
			return fmt.Errorf("fill the vector index for chunk %d: %w", one.chunk, err)
		}
	}
	return nil
}

// quantised reads a stored vector as the signed bytes it was written from.
func quantised(stored []byte) []int8 {
	out := make([]int8, len(stored))
	for i, b := range stored {
		out[i] = int8(b)
	}
	return out
}

// vectorWidth is the width the vector index holds, or nothing when it holds no
// vector index at all.
func vectorWidth(ctx context.Context, db *sql.DB) (int, error) {
	var created string
	err := db.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'chunks_vec'`).Scan(&created)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("what the vector index holds: %w", err)
	}
	found := declaredWidth.FindStringSubmatch(created)
	if found == nil {
		return 0, fmt.Errorf("the vector index does not say how wide it is: %s", created)
	}
	return strconv.Atoi(found[1])
}
