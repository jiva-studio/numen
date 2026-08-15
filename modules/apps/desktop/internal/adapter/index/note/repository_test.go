package note

import (
	"strings"
	"testing"
)

func TestFullTextRowsAreAddressedByRowID(t *testing.T) {
	// An FTS5 table has no key but its rowid: filtering a delete on the stored
	// columns scans the entire index, and Save does one delete per note, so the
	// mistake turns a rebuild from linear into quadratic. Measured on this
	// driver, one column-matched delete costs 11 ms at 2k rows and 112 ms at
	// 50k; at the sizes this is built for that is hours instead of minutes.
	clear := withoutComments(stmt.Get("clear_fts"))
	if !strings.Contains(clear, "rowid = ?") {
		t.Errorf("clear_fts does not delete by rowid: %s", clear)
	}
	if strings.Contains(clear, "vault_id") || strings.Contains(clear, "path") {
		t.Errorf("clear_fts filters on unindexed columns, which scans the index: %s", clear)
	}
	if !strings.Contains(withoutComments(stmt.Get("insert_fts")), "rowid") {
		t.Error("insert_fts does not set the rowid, so nothing can find the row again")
	}
}

// withoutComments leaves only what the database will execute, so that a test
// about a statement is not answered by the prose above it.
func withoutComments(sql string) string {
	var b strings.Builder
	for line := range strings.SplitSeq(sql, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			b.WriteString(line)
			b.WriteString(" ")
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
