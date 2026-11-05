package index

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

// schemaFile is the one file the whole schema of the index is in.
const schemaFile = "schema.sql"

type catalogueEntry struct{ kind, name, body string }

func readCatalogue(ctx context.Context, t *testing.T, db *sql.DB) []catalogueEntry {
	t.Helper()
	rows, err := db.QueryContext(ctx,
		`SELECT type, name, coalesce(sql, '') FROM sqlite_master
		 WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	}()
	var out []catalogueEntry
	for rows.Next() {
		var e catalogueEntry
		if err := rows.Scan(&e.kind, &e.name, &e.body); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func getUserVersion(ctx context.Context, t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func openRaw(t *testing.T, path string) *sql.DB {
	t.Helper()
	raw, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestAFreshIndexHasTheWholeSchemaOfOneFile(t *testing.T) {
	ctx := t.Context()
	body, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatalf("the schema is one file, %s: %v", schemaFile, err)
	}

	reference, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "reference.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, reference)
	if _, err := reference.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("the schema file does not run on an empty database: %v", err)
	}

	db, err := Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)

	got, want := readCatalogue(ctx, t, db.write), readCatalogue(ctx, t, reference)
	if len(want) == 0 {
		t.Fatal("the schema file creates nothing")
	}
	if !slices.Equal(got, want) {
		t.Errorf("a fresh index holds\n%v\nthe schema file makes\n%v", got, want)
	}
	if v := getUserVersion(ctx, t, db.write); v <= 0 {
		t.Errorf("a fresh index is at version %d, want the current one, above zero", v)
	}
}

// expectedVersion is the number the last of the old steps leaves, so an index of the current build matches it.
const expectedVersion = 10

func TestAnIndexOfTheCurrentBuildIsUsedAsItIsAndTheSchemaFileAppliesTwice(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Vaults().Register(ctx, "01KEPT"); err != nil {
		t.Fatal(err)
	}
	if v := getUserVersion(ctx, t, db.write); v != expectedVersion {
		t.Errorf("a new index is at version %d, want %d", v, expectedVersion)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("an index of the current build was refused: %v", err)
	}
	defer testsupport.Close(t, again)
	var kept int
	if err := again.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults WHERE identifier = '01KEPT'`).Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if kept != 1 {
		t.Errorf("%d rows of the vault are left, want 1: the index was not used as it is", kept)
	}

	body, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatalf("the schema is one file, %s: %v", schemaFile, err)
	}
	for range 2 {
		if _, err := again.write.ExecContext(ctx, string(body)); err != nil {
			t.Fatalf("the schema file does not apply over an index that has it: %v", err)
		}
	}
	if err := again.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults WHERE identifier = '01KEPT'`).Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if kept != 1 {
		t.Errorf("%d rows of the vault are left after the schema file applied, want 1", kept)
	}
}

func TestAnIndexOfAnotherVersionStopsTheStartAndStaysByteForByteTheSame(t *testing.T) {
	ctx := t.Context()

	for _, version := range []int{0, 1, expectedVersion - 1, expectedVersion + 1, expectedVersion + 100} {
		t.Run(fmt.Sprintf("version %d", version), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "index.db")
			raw := openRaw(t, path)
			for _, statement := range []string{
				`CREATE TABLE vaults (id INTEGER PRIMARY KEY, marker TEXT)`,
				`INSERT INTO vaults (marker) VALUES ('01OLDMARKER')`,
				`CREATE TABLE old_marker_rows (marker TEXT)`,
				`INSERT INTO old_marker_rows VALUES ('01OLDMARKER')`,
				fmt.Sprintf("PRAGMA user_version = %d", version),
			} {
				if _, err := raw.ExecContext(ctx, statement); err != nil {
					t.Fatal(err)
				}
			}
			if err := raw.Close(); err != nil {
				t.Fatal(err)
			}
			before := hashFile(t, path)

			db, err := Open(ctx, path)
			if err == nil {
				testsupport.Close(t, db)
				t.Fatal("an index of another version was opened")
			}
			for _, want := range []string{path, fmt.Sprint(version), fmt.Sprint(expectedVersion)} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the error %q does not name %q", err, want)
				}
			}
			if after := hashFile(t, path); after != before {
				t.Error("the file of the index changed")
			}
		})
	}
}

func hashFile(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(body)
}
