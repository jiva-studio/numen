package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestZZDump(t *testing.T) {
	db, err := Open(t.Context(), filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatal(err)
	}
	type e struct{ Kind, Name, Body string }
	var out []e
	for _, c := range readCatalogue(t.Context(), t, db.write) {
		out = append(out, e{c.kind, c.name, c.body})
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	os.WriteFile("/tmp/schema_dump.json", append(b, '\n'), 0o644)
	_ = getUserVersion
}
