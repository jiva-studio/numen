package note

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A note is cut at the sizes the repository was told, and the chunk that
// carries a vector stays under the limit. A chunk over the model's input limit
// is embedded truncated, and then it indexes text it does not hold.
func TestASmallChunkIsCutUnderTheLimitGiven(t *testing.T) {
	body := strings.TrimSpace(strings.Repeat("chunks carry vectors ", 200))
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/cut.md", Size: int64(len(body)), ModTime: time.Unix(0, 1)},
		Title:       "Cut",
		Body:        body,
	}

	for _, limit := range []int{64, 512} {
		chunks := cut(n, domain.Artifact{}, outline(n), chunking.Sizes{Limit: limit}, chunking.Legibility{})
		if len(chunks) != 1 {
			t.Fatalf("a note is one large chunk, and it was cut into %d", len(chunks))
		}
		if chunks[0].Length != len(body) {
			t.Errorf("the large chunk holds %d bytes of a note of %d", chunks[0].Length, len(body))
		}
		if len(chunks[0].Small) == 0 {
			t.Fatalf("nothing inside the note carries a vector at a limit of %d", limit)
		}
		for _, small := range chunks[0].Small {
			if held := utf8.RuneCountInString(small.Text); held > limit {
				t.Errorf("a chunk holds %d characters, told to hold %d", held, limit)
			}
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
