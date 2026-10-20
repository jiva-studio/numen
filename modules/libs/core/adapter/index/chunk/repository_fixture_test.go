package chunk_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"modernc.org/sqlite"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const (
	repoVaultA domain.VaultID = "01VAULTA"
	repoVaultB domain.VaultID = "01VAULTB"
)

var repoTemplate = testsupport.NewTemplate(func(ctx context.Context, path string) error {
	db, err := index.Open(ctx, path)
	if err != nil {
		return err
	}
	return db.Close()
})

// store is an index holding two empty vaults, and a second connection to the
// same file that a test reads and breaks it through.
type store struct {
	t    *testing.T
	db   *index.DB
	side *sql.DB
}

func newStore(t *testing.T) *store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := filepath.Join(t.TempDir(), "index.db")
	repoTemplate.CopyTo(t, path)
	db, err := index.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, id := range []domain.VaultID{repoVaultA, repoVaultB} {
		if err := db.Vaults().Register(t.Context(), id); err != nil {
			t.Fatal(err)
		}
	}
	side, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { side.Close() })
	return &store{t: t, db: db, side: side}
}

func (s *store) exec(statement string, args ...any) {
	s.t.Helper()
	if _, err := s.side.ExecContext(s.t.Context(), statement, args...); err != nil {
		s.t.Fatalf("%s: %v", statement, err)
	}
}

func (s *store) count(query string, args ...any) int {
	s.t.Helper()
	var n int
	if err := s.side.QueryRowContext(s.t.Context(), query, args...).Scan(&n); err != nil {
		s.t.Fatalf("%s: %v", query, err)
	}
	return n
}

func (s *store) ints(query string, args ...any) []int64 {
	s.t.Helper()
	rows, err := s.side.QueryContext(s.t.Context(), query, args...)
	if err != nil {
		s.t.Fatalf("%s: %v", query, err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			s.t.Fatal(err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		s.t.Fatal(err)
	}
	return out
}

func (s *store) strings(query string, args ...any) []string {
	s.t.Helper()
	rows, err := s.side.QueryContext(s.t.Context(), query, args...)
	if err != nil {
		s.t.Fatalf("%s: %v", query, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			s.t.Fatal(err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		s.t.Fatal(err)
	}
	return out
}

// rowsOf is the chunk rows a source holds, in order.
func (s *store) rowsOf(vault domain.VaultID, path string) []int64 {
	s.t.Helper()
	return s.ints(`SELECT c.id FROM chunks c JOIN sources s ON s.id = c.source_id
		JOIN vaults v ON v.id = s.vault_id WHERE v.identifier = ? AND s.path = ? ORDER BY c.id`, string(vault), path)
}

// paths is every source path a vault holds, in order.
func (s *store) paths(vault domain.VaultID) []string {
	s.t.Helper()
	return s.strings(`SELECT s.path FROM sources s JOIN vaults v ON v.id = s.vault_id
		WHERE v.identifier = ? ORDER BY s.path`, string(vault))
}

// cut records a source and the chunks it is cut into.
func (s *store) cut(vault domain.VaultID, kind, path string, chunks []chunk.Chunk) {
	s.t.Helper()
	repo := s.db.Chunks()
	if err := repo.SaveSource(s.t.Context(), vault, chunk.Source{Path: path, Kind: kind, Size: 10, MTime: 1}); err != nil {
		s.t.Fatal(err)
	}
	if err := repo.ReplaceChunks(s.t.Context(), vault, kind, path, chunks); err != nil {
		s.t.Fatal(err)
	}
}

// vectorFor is a vector for the chunk row given, addressed by the text the
// chunk holds.
func (s *store) vectorFor(row int64, recipe string, fill byte) chunk.Vector {
	s.t.Helper()
	var hash []byte
	if err := s.side.QueryRowContext(s.t.Context(), `SELECT unhex(hash) FROM chunks WHERE id = ?`, row).Scan(&hash); err != nil {
		s.t.Fatal(err)
	}
	return withHash(row, hash, recipe, fill)
}

func withHash(row int64, hash []byte, recipe string, fill byte) chunk.Vector {
	value := make([]byte, 1024)
	coarse := make([]byte, 128)
	for i := range value {
		value[i] = fill
	}
	for i := range coarse {
		coarse[i] = fill
	}
	return chunk.Vector{Chunk: row, Hash: hash, Recipe: recipe, Value: value, Coarse: coarse}
}

// wantSQLiteError fails unless the error still carries the database's own, so a
// caller can ask what the database said.
func wantSQLiteError(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error, want one saying %q", contains)
	}
	var cause *sqlite.Error
	if !errors.As(err, &cause) {
		t.Fatalf("the error %q no longer carries the database's own", err)
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("the error %q does not say %q", err, contains)
	}
}

func canceled(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func isCanceled(err error) bool { return errors.Is(err, context.Canceled) }

// failOn makes the database refuse every statement of one kind on a table with
// the message "boom". A virtual table is first replaced by an ordinary one of
// the same columns, since a trigger cannot sit on a virtual table.
func (s *store) failOn(table, event string) {
	s.t.Helper()
	type replacement struct{ columns, fill string }
	replaced := map[string]replacement{
		"chunks_fts":   {`(text TEXT)`, `(rowid, text) SELECT rowid, COALESCE(text, '') FROM chunks_fts`},
		"sections_fts": {`(text TEXT)`, `(rowid, text) SELECT rowid, COALESCE(text, '') FROM sections_fts`},
		"titles_fts":   {`(text TEXT)`, `(rowid, text) SELECT rowid, text FROM titles_fts`},
		"chunks_vec": {
			`(chunk_id INTEGER PRIMARY KEY, vault_id INTEGER, coarse BLOB)`,
			`SELECT chunk_id, vault_id, coarse FROM chunks_vec`,
		},
	}
	if with, virtual := replaced[table]; virtual {
		s.exec(`CREATE TABLE ` + table + `_copy ` + with.columns)
		s.exec(`INSERT INTO ` + table + `_copy ` + with.fill)
		s.exec(`DROP TABLE ` + table)
		s.exec(`ALTER TABLE ` + table + `_copy RENAME TO ` + table)
	}
	s.exec(`CREATE TRIGGER refuse_` + strings.ToLower(event) + `_` + table + ` BEFORE ` + event + ` ON ` + table +
		` BEGIN SELECT RAISE(ABORT, 'boom'); END`)
}
