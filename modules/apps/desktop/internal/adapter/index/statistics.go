package index

import (
	"context"
	"database/sql"
)

// Statistics keeps the database's picture of its own contents current.
//
// Without it the database answers "who points at this note" by narrowing to the
// vault and then reading every link in it, because with nothing measured it has
// no reason to believe the narrower index is worth using. The indexes are there
// either way; what is missing is the knowledge that they help.
type Statistics struct{ db *sql.DB }

// Update re-measures.
//
// The mask asks for every table rather than only the ones this connection has
// read from, which is what a scan needs: it writes and does not query, so a
// connection-scoped decision would find nothing worth measuring and leave the
// index exactly as slow as before. The database limits how much of a large
// table it samples, so this stays a moment's work rather than a second pass
// over the vault.
func (s Statistics) Update(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "PRAGMA optimize = 0x10002")
	return err
}
