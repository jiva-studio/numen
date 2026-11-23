// Package operationresult stores the results of connection operations in the index database, by operation,
// recipe and input. The table is shared by every vault.
package operationresult

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

// batchSize is the most input hashes one statement reads.
const batchSize = 500

// Repository is the stored results. Reads and writes go through the write pool, since a read marks the rows
// it finds used. A save removes the least recently used rows above maxRows.
type Repository struct {
	write   *sql.DB
	maxRows int
}

// NewRepository returns the results stored in the index database, at most maxRows of them.
func NewRepository(write *sql.DB, maxRows int) *Repository {
	return &Repository{write: write, maxRows: maxRows}
}

// FindResults returns the stored outputs of the input hashes that have one, by input hash, and marks them
// used at at.
func (r *Repository) FindResults(
	ctx context.Context, operation, recipeHash string, inputHashes []string, at time.Time,
) (map[string][]byte, error) {
	found := map[string][]byte{}
	err := sqlwrite.RunInTransaction(ctx, r.write, func(tx *sqlwrite.Transaction) error {
		for start := 0; start < len(inputHashes); start += batchSize {
			batch, err := json.Marshal(inputHashes[start:min(start+batchSize, len(inputHashes))])
			if err != nil {
				return err
			}
			if err := readOutputs(ctx, tx, found, operation, recipeHash, string(batch)); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, getStatement("touch"), operation, recipeHash, string(batch), at.UnixNano()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read the stored results of %s: %w", operation, err)
	}
	return found, nil
}

func readOutputs(ctx context.Context, tx *sqlwrite.Transaction, found map[string][]byte, operation, recipeHash, batch string) (err error) {
	rows, err := tx.QueryContext(ctx, getStatement("find"), operation, recipeHash, batch)
	if err != nil {
		return err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	for rows.Next() {
		var hash string
		var output []byte
		if err := rows.Scan(&hash, &output); err != nil {
			return err
		}
		found[hash] = output
	}
	return rows.Err()
}

// SaveResults stores the outputs by input hash, used at at, replacing equal keys, then removes the least
// recently used rows above the row limit.
func (r *Repository) SaveResults(ctx context.Context, operation, recipeHash string, outputs map[string][]byte, at time.Time) error {
	err := sqlwrite.RunInTransaction(ctx, r.write, func(tx *sqlwrite.Transaction) error {
		for hash, output := range outputs {
			if _, err := tx.ExecContext(ctx, getStatement("save"), operation, recipeHash, hash, output, at.UnixNano()); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, getStatement("evict"), r.maxRows)
		return err
	})
	if err != nil {
		return fmt.Errorf("store the results of %s: %w", operation, err)
	}
	return nil
}

// ClearResults removes every stored result and returns how many it removed.
func (r *Repository) ClearResults(ctx context.Context) (int, error) {
	result, err := sqlwrite.Exec(ctx, r.write, getStatement("clear"))
	if err != nil {
		return 0, fmt.Errorf("clear the stored results: %w", err)
	}
	removed, err := result.RowsAffected()
	return int(removed), err
}
