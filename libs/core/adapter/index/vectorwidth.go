package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/internal/vector"
)

// declaredWidth reads the width out of the statement the table was created by,
// which is where a virtual table keeps its shape.
var declaredWidth = regexp.MustCompile(`bit\[(\d+)]`)

// FitVectors makes the coarse index hold vectors of the width given and of the
// recipe given, and fills it from what that recipe has already bought.
//
// The coarse index is built for one width, and a model of another width cannot
// be written to it. Two models of one width write bits that mean different
// things, so the index also records the recipe that filled it. What a model
// made is kept by the text it read, in a table of its own, and the coarse form
// is read out of that. A chunk the recipe holds no vector for gets no row until
// it is embedded.
func (d *DB) FitVectors(ctx context.Context, dims int, recipe string) error {
	if dims <= 0 {
		return fmt.Errorf("a vector of %d dimensions is not a vector", dims)
	}
	target, err := getVectorWidth(ctx, d.write)
	if err != nil {
		return err
	}
	filledBy, recorded, err := coarseRecipe(ctx, d.write)
	if err != nil {
		return err
	}
	if target == dims && recorded && filledBy == recipe {
		return nil
	}

	return sqlwrite.InTransaction(ctx, d.write, func(tx *sqlwrite.Transaction) error {
		for _, statement := range []string{
			`DROP TABLE IF EXISTS chunks_vec`,
			fmt.Sprintf(`CREATE VIRTUAL TABLE chunks_vec USING vec0 (
			chunk_id integer primary key,
			vault_id integer,
			coarse   bit[%d]
		)`, dims),
		} {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("fit the vector index to %d dimensions: %w", dims, err)
			}
		}
		if err := fillCoarse(ctx, tx, dims, recipe); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO coarse_recipe (id, recipe) VALUES (1, ?)
		 ON CONFLICT (id) DO UPDATE SET recipe = excluded.recipe`, recipe); err != nil {
			return fmt.Errorf("record the recipe that filled the vector index: %w", err)
		}
		return nil
	})
}

// coarseRecipe is the recipe that filled the coarse index, and whether any is
// recorded.
func coarseRecipe(ctx context.Context, db *sql.DB) (string, bool, error) {
	var recipe string
	err := db.QueryRowContext(ctx, `SELECT recipe FROM coarse_recipe WHERE id = 1`).Scan(&recipe)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("which recipe filled the vector index: %w", err)
	}
	return recipe, true, nil
}

// fillCoarse writes a coarse row for every vector the recipe holds of the width
// given. A vector of another width belongs to another model and is left where
// it is.
func fillCoarse(ctx context.Context, tx *sqlwrite.Transaction, dims int, recipe string) (err error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT c.id, c.vault_id, v.embedding
		 FROM vectors v JOIN chunks c ON unhex(c.hash) = v.hash
		 WHERE v.recipe = ? AND length(v.embedding) = ?`, recipe, dims)
	if err != nil {
		return fmt.Errorf("the vectors the recipe holds: %w", err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	type coarse struct {
		chunk, vault int64
		bits         []byte
	}
	var target []coarse
	for rows.Next() {
		var one coarse
		var value []byte
		if err := rows.Scan(&one.chunk, &one.vault, &value); err != nil {
			return fmt.Errorf("the vectors the recipe holds: %w", err)
		}
		one.bits = vector.Coarse(vector.Dimensions(value))
		target = append(target, one)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("the vectors the recipe holds: %w", err)
	}

	// `vec_bit` marks the blob as one bit per dimension; its length alone does not
	// distinguish it from float32.
	for _, one := range target {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chunks_vec (chunk_id, vault_id, coarse) VALUES (?, ?, vec_bit(?))`,
			one.chunk, one.vault, one.bits); err != nil {
			return fmt.Errorf("fill the vector index for chunk %d: %w", one.chunk, err)
		}
	}
	return nil
}

// getVectorWidth is the width the vector index holds, or nothing when it holds no
// vector index at all.
func getVectorWidth(ctx context.Context, db *sql.DB) (int, error) {
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
