package note

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// errNoVault is returned when the index has no record of the vault being
// queried. Every query about an unknown vault has the same result — nothing —
// so callers convert this into an empty result.
var errNoVault = errors.New("vault not in the index")

// querier is anything that can answer a single-row query, so this works inside
// a transaction as well as outside one.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func vaultRow(ctx context.Context, db querier, identifier domain.VaultID) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, vaultRowSQL, string(identifier)).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoVault
	}
	return row, err
}
