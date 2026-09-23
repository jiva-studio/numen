package note

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// errNoVault is returned when the index has never been told about the vault
// being asked about. Every question about a vault it does not know has the same
// answer — nothing — so callers turn this into an empty result.
var errNoVault = errors.New("vault not in the index")

// querier is anything that can answer a single-row question, so that this works
// inside a transaction as well as outside one.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// vaultRow turns the identifier a vault carries in the world into the row
// number this database files it under.
func vaultRow(ctx context.Context, db querier, identifier domain.VaultID) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, stmt.Get("vault_row"), string(identifier)).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoVault
	}
	return row, err
}
