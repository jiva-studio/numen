package index

import "testing"

func TestScratchProbeOneBitWidth(t *testing.T) {
	for _, dims := range []int{1, 0} {
		db := openDB(t)
		_, err := db.write.ExecContext(t.Context(), "DROP TABLE chunks_vec")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.write.ExecContext(t.Context(), "CREATE VIRTUAL TABLE chunks_vec USING vec0 (chunk_id integer primary key, vault_id integer, coarse bit["+string(rune('0'+dims))+"])")
		t.Logf("PROBE bit[%d]: %v", dims, err)
	}
}
