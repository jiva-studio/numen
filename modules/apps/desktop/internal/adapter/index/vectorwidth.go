package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// coarseWidth is the width the vector index is created at, and is what an
// installation with no model leaves it as.
const coarseWidth = 1024

// declaredWidth reads the width out of the statement the table was created by,
// which is where a virtual table keeps its shape.
var declaredWidth = regexp.MustCompile(`bit\[(\d+)]`)

// FitVectors makes the coarse index hold vectors of the width given.
//
// The coarse index is built for one width, and a model of another width cannot
// be written to it. The width is the model's, so changing model rebuilds it
// from what has been made. What a model made is kept by the text it read, and
// is not touched here.
func (db *DB) FitVectors(ctx context.Context, dims int) error {
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
	return tx.Commit()
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
