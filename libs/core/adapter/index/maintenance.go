package index

import (
	"context"
	"database/sql"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
)

// DatabaseMaintenance keeps the database's picture of its own contents current,
// which is what it chooses between indexes by.
type DatabaseMaintenance struct{ db *sql.DB }

// Every bit is named because naming any turns off the ones left out.
//
//	0x00010  sample a large table, stopping short of reading all of it
//	0x00002  measure the tables that stand to gain by it
//	0x10000  include tables this connection has not read from
//
// The last one is what a scan needs: it writes and asks nothing, on whichever
// pooled connection was free.
//
// A pragma configures the connection, so it belongs here beside the ones in
// db.go.
const measure = "PRAGMA optimize = 0x10012"

// ReportChanges says what the database knows about itself is out of date.
func (m DatabaseMaintenance) ReportChanges(ctx context.Context) error {
	if _, err := sqlwrite.Exec(ctx, m.db, measure); err != nil {
		return err
	}
	for _, fts := range []string{"chunks_fts", "sections_fts", "titles_fts", "headings_fts"} {
		_, _ = sqlwrite.Exec(ctx, m.db, "INSERT INTO "+fts+"("+fts+") VALUES('optimize')")
	}
	return nil
}
