package index

import (
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestAnIndexAtVersionOneGainsTheTableOfOperationResultsAndKeepsItsData(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")
	raw := openRawDB(t, path)
	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if err := apply(ctx, raw, available[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO vaults (identifier) VALUES ('01KEPT')`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)
	if v := getUserVersion(ctx, t, db.write); v != expectedVersion {
		t.Errorf("the index is at version %d, want %d", v, expectedVersion)
	}
	if !hasTable(t, db.write, "operation_results") {
		t.Error("the index has no table operation_results")
	}
	var kept int
	if err := db.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults WHERE identifier = '01KEPT'`).Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if kept != 1 {
		t.Errorf("%d rows of the vault are left, want 1", kept)
	}
}
