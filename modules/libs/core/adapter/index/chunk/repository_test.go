package chunk

import (
	"strings"
	"testing"
)

func TestVectorRowsAreAddressedByTheChunkNumber(t *testing.T) {
	// The vector index has no key but its rowid, which is the chunk's own
	// number. A delete that filters on a metadata column reads the whole index,
	// The vector index has no key but its rowid, which is the chunk's own
	// number, and a delete addresses a row by it.
	for _, name := range []string{"insert_vec", "delete_vec", "clear_vec"} {
		statement := withoutComments(stmt.Get(name))
		if !strings.Contains(statement, "chunk_id") {
			t.Errorf("%s does not address the row by the chunk's number: %s", name, statement)
		}
	}
	for _, name := range []string{"delete_vec", "clear_vec"} {
		statement := withoutComments(stmt.Get(name))
		if strings.Contains(statement, "vault_id") {
			t.Errorf("%s deletes by a column the index cannot search: %s", name, statement)
		}
	}
}

func TestFullTextRowsAreAddressedByTheChunkNumber(t *testing.T) {
	// An FTS5 table has no key but its rowid, which is the chunk's own number.
	// Filtering on the indexed column scans the entire index, and every chunk
	// The full-text index has no key but its rowid, which is the chunk's own
	// number, and a delete addresses a row by it.
	for _, name := range []string{"insert_fts", "clear_fts"} {
		statement := withoutComments(stmt.Get(name))
		if !strings.Contains(statement, "rowid") {
			t.Errorf("%s does not address the row by its rowid: %s", name, statement)
		}
		if strings.Contains(statement, "vault_id") {
			t.Errorf("%s filters on a column the index cannot search: %s", name, statement)
		}
	}
}

func TestTheCoarsePassIsConstrainedInsideTheQuery(t *testing.T) {
	// A nearest-neighbour search answers with the whole table's best k. A vault
	// filter applied to the answer instead returns fewer than k passages, or
	// none, and a query plan shows neither case: it says only that the virtual
	// table was read.
	statement := withoutComments(stmt.Get("search"))
	for _, want := range []string{"MATCH", "vault_id = ?", "k = ?", "ORDER BY distance"} {
		if !strings.Contains(statement, want) {
			t.Errorf("the coarse pass does not constrain %q: %s", want, statement)
		}
	}
}

func TestABitVectorSaysThatItIsOne(t *testing.T) {
	// The 128 bytes of a 1024-bit vector are also 32 float32 dimensions, and the
	// extension reads them as the latter unless the type is named.
	for _, name := range []string{"insert_vec", "search"} {
		if !strings.Contains(withoutComments(stmt.Get(name)), "vec_bit(?)") {
			t.Errorf("%s does not say the blob is one bit per dimension", name)
		}
	}
}

func TestTheVaultOfAVectorRowComesFromItsChunk(t *testing.T) {
	// The vault a chunk carries comes from its source, so the two cannot
	// disagree.
	statement := withoutComments(stmt.Get("insert_vec"))
	if !strings.Contains(statement, "SELECT id, vault_id, vec_bit(?) FROM chunks") {
		t.Errorf("insert_vec takes the vault from somewhere other than the chunk: %s", statement)
	}
}

func TestNothingIsStoredAgainstAPath(t *testing.T) {
	// A chunk is pointed at by its source's identity in this database, and the
	// path that source is filed under lives in one place.
	for _, name := range []string{
		"insert_chunk", "clear_chunks", "clear_vec", "delete_vec", "insert_fts", "clear_fts",
	} {
		statement := withoutComments(stmt.Get(name))
		if strings.Contains(statement, "path") {
			t.Errorf("%s addresses a source by where it is filed: %s", name, statement)
		}
	}
}

// withoutComments leaves only what the database will execute, for a test
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

func TestTheCoarseRowReadsTheChunkItBelongsTo(t *testing.T) {
	// A chunk that went while its vector was being made is given no coarse row,
	// so the coarse index never names a chunk that is not there. The vector
	// itself is kept whatever became of the chunk: it was paid for, and the
	// text it was made from is what addresses it.
	statement := withoutComments(stmt.Get("insert_vec"))
	if !strings.Contains(statement, "FROM chunks WHERE id = ?") {
		t.Errorf("insert_vec writes without reading the chunk it belongs to: %s", statement)
	}
	if kept := withoutComments(stmt.Get("keep_vector")); strings.Contains(kept, "FROM chunks") {
		t.Errorf("keep_vector reads a chunk, and a vector outlives every chunk: %s", kept)
	}
}
