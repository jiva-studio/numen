package note

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// reads runs the queries of one read through the pool. Each statement is
// prepared on its first use and reused until close.
type reads struct {
	db    *sql.DB
	ready map[string]*sql.Stmt
}

func newReads(db *sql.DB) *reads { return &reads{db: db, ready: map[string]*sql.Stmt{}} }

func (r *reads) prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	if prepared, ok := r.ready[query]; ok {
		return prepared, nil
	}
	prepared, err := prepareStatement(ctx, r.db, query)
	if err != nil {
		return nil, err
	}
	r.ready[query] = prepared
	return prepared, nil
}

func (r *reads) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	prepared, err := r.prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	return prepared.QueryContext(ctx, args...)
}

// QueryRowContext passes a statement that could not be prepared to the pool
// itself, so the row carries the error to the caller.
func (r *reads) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	prepared, err := r.prepare(ctx, query)
	if err != nil {
		return r.db.QueryRowContext(ctx, query, args...)
	}
	return prepared.QueryRowContext(ctx, args...)
}

func prepareStatement(ctx context.Context, db *sql.DB, query string) (*sql.Stmt, error) {
	// The driver keeps a statement compiled between queries only when nothing,
	// whitespace included, follows its last semicolon. Otherwise it compiles the
	// statement again for every query.
	return db.PrepareContext(ctx, strings.TrimSpace(query))
}

func (r *reads) close() error {
	var err error
	for _, prepared := range r.ready {
		err = sqlwrite.JoinClose(err, prepared.Close())
	}
	return err
}

// resolution resolves the links of one read in one vault. The candidates of a
// name are asked once per read, because the backlinks of one note mostly write
// the same name.
type resolution struct {
	reads   *reads
	vault   int64
	vaultID domain.VaultID
	named   map[string][]string
}

func runResolution[T any](
	ctx context.Context, db *sql.DB, vaultID domain.VaultID, none T, read func(r *resolution) (T, error),
) (_ T, err error) {
	statements := newReads(db)
	defer func() { err = sqlwrite.JoinClose(err, statements.close()) }()

	vault, err := vaultRow(ctx, statements, vaultID)
	if errors.Is(err, errNoVault) {
		return none, nil
	}
	if err != nil {
		var zero T
		return zero, err
	}
	return read(&resolution{reads: statements, vault: vault, vaultID: vaultID, named: map[string][]string{}})
}
