package index

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// SQL lives in .sql files: it is a language of its own, and anything that
// reads, formats or checks SQL can see it there. The files are embedded, so
// the binary carries no runtime dependency on the source tree.
//
//go:embed migration/*.sql
var files embed.FS

// A migration is one numbered file, applied once, in order.
//
// The index is a cache and could in principle be thrown away and rebuilt on
// every schema change. It is migrated instead because a rebuild is minutes of
// work at the sizes this is built for, and an application update that silently
// costs the user those minutes — every time — is not a good trade. A migration
// that genuinely cannot preserve what it changes is free to empty the affected
// tables; the next scan refills them, and the decision is written in the file
// that made it.
type migration struct {
	version int
	name    string
	body    string
}

// migrate brings the database up to the newest migration.
//
// Each migration runs in its own transaction together with the version bump, so
// a failure leaves the database at the last version that fully applied rather
// than half-way through one.
func migrate(ctx context.Context, db *sql.DB) error {
	available, err := loadMigrations()
	if err != nil {
		return err
	}

	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
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

func apply(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range statements(m.body) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("%w\nin statement:\n%s", err, strings.TrimSpace(stmt))
		}
	}
	// PRAGMA takes no parameters, and the number is the migration's own.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
		return err
	}
	return tx.Commit()
}

// loadMigrations reads the embedded files and orders them by version. A file
// that does not start with a number is a mistake worth failing on: silently
// skipping it would leave a schema nobody can reproduce.
func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(files, "migration")
	if err != nil {
		return nil, err
	}
	out := make([]migration, 0, len(entries))
	seen := map[int]string{}
	for _, e := range entries {
		version, err := versionOf(e.Name())
		if err != nil {
			return nil, err
		}
		if previous, clash := seen[version]; clash {
			return nil, fmt.Errorf("migrations %s and %s share version %d", previous, e.Name(), version)
		}
		seen[version] = e.Name()

		body, err := files.ReadFile(path.Join("migration", e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: version, name: e.Name(), body: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func versionOf(name string) (int, error) {
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

// statements cuts a file on semicolons. This is deliberately not a SQL parser:
// a statement that needs more than this does not belong in a migration.
func statements(raw string) []string {
	var out []string
	for _, chunk := range strings.Split(raw, ";") {
		if strings.TrimSpace(withoutComments(chunk)) != "" {
			out = append(out, chunk)
		}
	}
	return out
}

func withoutComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}
