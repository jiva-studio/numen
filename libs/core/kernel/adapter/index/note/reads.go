package note

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// statementCache runs the queries of one read through the pool. Each statement is
// prepared on its first use and reused until close.
type statementCache struct {
	db    *sql.DB
	ready map[string]*sql.Stmt
}

func newStatementCache(db *sql.DB) *statementCache {
	return &statementCache{db: db, ready: map[string]*sql.Stmt{}}
}

func (r *statementCache) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return useStatement(ctx, r, query, func(prepared *sql.Stmt) (*sql.Rows, error) {
		return prepared.QueryContext(ctx, args...)
	})
}

// QueryRowContext passes a statement that could not be prepared to the pool
// itself, so the row carries the error to the caller.
func (r *statementCache) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	row, err := useStatement(ctx, r, query, func(prepared *sql.Stmt) (*sql.Row, error) {
		return prepared.QueryRowContext(ctx, args...), nil
	})
	if err != nil {
		return r.db.QueryRowContext(ctx, query, args...)
	}
	return row
}

func useStatement[T any](ctx context.Context, r *statementCache, query string, run func(*sql.Stmt) (T, error)) (T, error) {
	prepared, ok := r.ready[query]
	if !ok {
		var err error
		prepared, err = prepareStatement(ctx, r.db, query)
		if err != nil {
			var zero T
			return zero, err
		}
		r.ready[query] = prepared
	}
	return run(prepared)
}

func prepareStatement(ctx context.Context, db *sql.DB, query string) (*sql.Stmt, error) {
	// The driver keeps a statement compiled between queries only when nothing,
	// whitespace included, follows its last semicolon. Otherwise it compiles the
	// statement again for every query.
	return db.PrepareContext(ctx, strings.TrimSpace(query))
}

func (r *statementCache) close() error {
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
	statements *statementCache
	vault      int64
	vaultID    domain.VaultID
	named      map[string][]string
}

func runResolution[T any](
	ctx context.Context, db *sql.DB, vaultID domain.VaultID, none T, read func(r *resolution) (T, error),
) (_ T, err error) {
	statements := newStatementCache(db)
	defer func() { err = sqlwrite.JoinClose(err, statements.close()) }()

	vault, err := vaultRow(ctx, statements, vaultID)
	if errors.Is(err, errNoVault) {
		return none, nil
	}
	if err != nil {
		var zero T
		return zero, err
	}
	return read(&resolution{statements: statements, vault: vault, vaultID: vaultID, named: map[string][]string{}})
}
