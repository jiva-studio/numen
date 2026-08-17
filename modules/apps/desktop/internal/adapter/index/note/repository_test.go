package note

import (
	"strings"
	"testing"
)

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
