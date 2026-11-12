package index

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
)

// SQL lives in .sql files: it is a language of its own, and anything that
// reads, formats or checks SQL can see it there. The files are embedded, so
// the binary carries no runtime dependency on the source tree.
//
//go:embed migration/*.sql
var files embed.FS

// A migration is one numbered file, applied once, in order.
//
// A change of the schema is a new numbered file after the last one. The index is a
// cache of the vault, so a file at a version no file carries (one a newer build
// wrote) is deleted and built again, and an unrelated database that has no
// version is left as it is and reported.
type migration struct {
	version int
	name    string
	body    string
}

// errUnknownVersion is an index at a version this build does not know.
var errUnknownVersion = errors.New("the index is at an unknown version")

func migrate(ctx context.Context, db *sql.DB, file string) error {
	available, err := loadMigrations()
	if err != nil {
		return err
	}
	current, err := checkVersion(ctx, db, file, available)
	if err != nil {
		return err
	}

	for _, m := range available {
		if m.version <= current {
			continue
		}
		if err := apply(ctx, db, m); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	return nil
}

func checkVersion(ctx context.Context, db *sql.DB, file string, available []migration) (int, error) {
	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return 0, fmt.Errorf("what version this index is at: %w", err)
	}
	newest := available[len(available)-1].version
	if slices.ContainsFunc(available, func(m migration) bool { return m.version == current }) {
		return current, nil
	}
	if current == 0 {
		empty, err := isEmpty(ctx, db)
		if err != nil || empty {
			return 0, err
		}
	}
	if current > 0 {
		return 0, fmt.Errorf("the index %s is at version %d and this build expects version %d: %w", file, current, newest, errUnknownVersion)
	}
	return 0, fmt.Errorf("the database %s is at version %d and holds tables, and this build expects version %d: it is left as it is", file, current, newest)
}

func isEmpty(ctx context.Context, db *sql.DB) (bool, error) {
	var objects int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&objects); err != nil {
		return false, fmt.Errorf("what this index holds: %w", err)
	}
	return objects == 0, nil
}

func checkFile(ctx context.Context, file string) error {
	available, err := loadMigrations()
	if err != nil {
		return err
	}
	plain, err := sql.Open("sqlite", dsnOf(file, nil))
	if err != nil {
		return fmt.Errorf("open the index at %s: %w", file, err)
	}
	_, checked := checkVersion(ctx, plain, file, available)
	return errors.Join(checked, plain.Close())
}

func createBookkeeping(ctx context.Context, tx *sqlwrite.Transaction) error {
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name    TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("what this index was migrated by: %w", err)
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	return sqlwrite.InTransaction(ctx, db, func(tx *sqlwrite.Transaction) error {
		if err := createBookkeeping(ctx, tx); err != nil {
			return err
		}
		for _, stmt := range splitStatements(m.body) {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("%w\nin statement:\n%s", err, strings.TrimSpace(stmt))
			}
		}
		// PRAGMA takes no parameters, and the number is the migration's own.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
			return err
		}
		// The migration name is recorded with the version, so the state of the index
		// can be inspected.
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, name) VALUES (?, ?)
		 ON CONFLICT (version) DO UPDATE SET name = excluded.name`,
			m.version, m.name); err != nil {
			return err
		}
		return nil
	})
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(files, "migration")
	if err != nil {
		return nil, err
	}
	out := make([]migration, 0, len(entries))
	for _, e := range entries {
		version, err := parseVersion(e.Name())
		if err != nil {
			return nil, err
		}
		body, err := files.ReadFile(path.Join("migration", e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: version, name: e.Name(), body: string(body)})
	}
	return out, nil
}

func parseVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration %q is not named <version>_<description>.sql", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("migration %q does not start with a version number", name)
	}
	return version, nil
}

func splitStatements(raw string) []string {
	var out []string
	for chunk := range strings.SplitSeq(raw, ";") {
		if strings.TrimSpace(stripComments(chunk)) != "" {
			out = append(out, chunk)
		}
	}
	return out
}

func stripComments(s string) string {
	var b strings.Builder
	for line := range strings.SplitSeq(s, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func removeIndexFiles(file string) error {
	for _, name := range []string{file, file + "-wal", file + "-shm"} {
		if err := os.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("delete the index %s: %w", name, err)
		}
	}
	return nil
}
