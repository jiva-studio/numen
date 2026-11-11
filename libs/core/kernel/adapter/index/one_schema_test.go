package index

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

// expectedVersion is the schema version of this build. 10 is what the build before the collapse left,
// and an index of that build is not readable by this one.
const expectedVersion = 1

type catalogEntry struct{ kind, name, body string }

func readCatalog(ctx context.Context, t *testing.T, db *sql.DB) []catalogEntry {
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
	var out []catalogEntry
	for rows.Next() {
		var e catalogEntry
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

func TestAFreshIndexHasExactlyTheRecordedSchema(t *testing.T) {
	ctx := t.Context()
	raw, err := os.ReadFile(filepath.Join("testdata", "schema_of_the_ten_steps.json"))
	if err != nil {
		t.Fatal(err)
	}
	var dump []struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
		Body string `json:"body"`
	}
	if err := json.Unmarshal(raw, &dump); err != nil {
		t.Fatal(err)
	}
	var want []catalogEntry
	for _, d := range dump {
		want = append(want, catalogEntry{d.Kind, d.Name, d.Body})
	}

	db, err := Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)

	got := readCatalog(ctx, t, db.write)
	if !slices.Equal(got, want) {
		for _, e := range got {
			if !slices.Contains(want, e) {
				t.Errorf("a fresh index has %s %s as\n%s\nthe recorded schema does not hold it so", e.kind, e.name, e.body)
			}
		}
		for _, e := range want {
			if !slices.Contains(got, e) {
				t.Errorf("a fresh index lacks %s %s as\n%s", e.kind, e.name, e.body)
			}
		}
	}
	if v := getUserVersion(ctx, t, db.write); v != expectedVersion {
		t.Errorf("a fresh index is at version %d, want %d", v, expectedVersion)
	}
}

func TestTheMigrationFolderHasOneFileNamedForTheIndexVersion(t *testing.T) {
	entries, err := os.ReadDir("migration")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 {
		t.Fatalf("the folder migration holds %v, want one file", names)
	}
	if want := fmt.Sprintf("%04d_", expectedVersion); !strings.HasPrefix(names[0], want) {
		t.Errorf("the file %s is not numbered %s", names[0], want)
	}
}

func TestAnIndexOfTheCurrentBuildIsOpenedAndUsedAsItIs(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Vaults().Register(ctx, "01KEPT"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("an index of the current build was refused: %v", err)
	}
	defer testsupport.Close(t, again)
	var remaining int
	if err := again.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults WHERE identifier = '01KEPT'`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Errorf("%d rows of the vault are left, want 1: the index was not used as it is", remaining)
	}
	if v := getUserVersion(ctx, t, again.write); v != expectedVersion {
		t.Errorf("the index is at version %d, want %d", v, expectedVersion)
	}
}

func TestAnIndexOfAnotherVersionStopsTheStartAndStaysByteForByteTheSame(t *testing.T) {
	ctx := t.Context()

	for _, version := range []int{0, 2, 10, 110} {
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
