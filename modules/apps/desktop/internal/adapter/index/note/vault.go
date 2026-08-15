package note

import (
	"context"
	"database/sql"
	"errors"
)

// errNoVault is returned when the index has never been told about the vault
// being asked about. Every question about a vault it does not know has the same
// answer — nothing — so callers turn this into an empty result.
var errNoVault = errors.New("vault not in the index")

// row is the anything that can answer a single-row query, so that this works
// inside a transaction as well as outside one.
type row interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// vaultRow turns the identity a vault carries in the world into the one this
// database uses.
func vaultRow(ctx context.Context, db row, identifier string) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, stmt.Get("vault_row"), identifier).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoVault
	}
	return id, err
}
