package note

import (
	"strings"
	"testing"
)

func TestFullTextRowsAreAddressedByRowID(t *testing.T) {
	// An FTS5 table has no key but its rowid: filtering on the indexed columns
	// scans the entire index, and Save touches the full-text row once per note,
	// so the mistake turns a rebuild from linear into quadratic.
	for _, name := range []string{"save_fts", "delete_fts"} {
		statement := withoutComments(stmt.Get(name))
		if !strings.Contains(statement, "rowid") {
			t.Errorf("%s does not address the row by its rowid: %s", name, statement)
		}
		if strings.Contains(statement, "vault_id") || strings.Contains(statement, "path") {
			t.Errorf("%s filters on columns the index cannot search: %s", name, statement)
		}
	}
}

// TestNothingIsStoredAgainstAPath holds the shape of the schema: a note is
// pointed at by its identity in this database, and the path it is filed under
// lives in one place.
func TestNothingIsStoredAgainstAPath(t *testing.T) {
	for _, name := range []string{
		"insert_heading", "insert_link", "insert_problem",
		"clear_headings", "clear_links", "clear_problems",
	} {
		statement := withoutComments(stmt.Get(name))
		if strings.Contains(statement, "path") || strings.Contains(statement, "vault_id") {
			t.Errorf("%s addresses a note by where it is filed: %s", name, statement)
		}
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
