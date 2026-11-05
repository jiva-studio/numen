package index

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func TestAnIndexOfAnotherOrNoVersionIsEmptiedAndTheVaultIsIndexedAgain(t *testing.T) {
	ctx := t.Context()

	current := func() int {
		db, err := Open(ctx, filepath.Join(t.TempDir(), "current.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer testsupport.Close(t, db)
		return getUserVersion(ctx, t, db.write)
	}()

	versions := []int{0, current + 1, current + 100}
	if current > 1 {
		versions = append(versions, current-1)
	}

	for _, version := range versions {
		t.Run(fmt.Sprintf("version %d of the current %d", version, current), func(t *testing.T) {
			t.Run("with the current tables", func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "index.db")
				db, err := Open(ctx, path)
				if err != nil {
					t.Fatal(err)
				}
				if err := db.Vaults().Register(ctx, "01OLDMARKER"); err != nil {
					t.Fatal(err)
				}
				if _, err := db.write.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
					t.Fatal(err)
				}
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
				checkEmptiedAndIndexedAgain(ctx, t, path, current)
			})

			t.Run("with tables of another shape", func(t *testing.T) {
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
				checkEmptiedAndIndexedAgain(ctx, t, path, current)
			})
		})
	}
}

func checkEmptiedAndIndexedAgain(ctx context.Context, t *testing.T, path string, current int) {
	t.Helper()
	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("an index of another version stopped this build: %v", err)
	}
	defer testsupport.Close(t, db)

	if v := getUserVersion(ctx, t, db.write); v != current {
		t.Errorf("the index is at version %d, want %d", v, current)
	}
	for _, e := range readCatalogue(ctx, t, db.write) {
		if e.name == "old_marker_rows" {
			t.Errorf("a table of the old index is still there: %s", e.name)
		}
	}
	var held int
	if err := db.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults`).Scan(&held); err != nil {
		t.Fatalf("the vaults table of the current schema is not there: %v", err)
	}
	if held != 0 {
		t.Errorf("the index holds %d vaults of the old file, want none", held)
	}

	// The vault is indexed again: the registration of the next scan finds an empty index.
	if err := db.Vaults().Register(ctx, "01AGAIN"); err != nil {
		t.Fatalf("the vault cannot be registered again: %v", err)
	}
	if err := db.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults`).Scan(&held); err != nil {
		t.Fatal(err)
	}
	if held != 1 {
		t.Errorf("the index holds %d vaults after one registration, want 1", held)
	}
}
