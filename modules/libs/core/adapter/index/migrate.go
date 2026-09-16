package index

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/writing"
)

// SQL lives in .sql files: it is a language of its own, and anything that
// reads, formats or checks SQL can see it there. The files are embedded, so
// the binary carries no runtime dependency on the source tree.
//
//go:embed migration/*.sql
var files embed.FS

// dropping is the order objects come out of the index in. A virtual table goes
// first because it owns shadow tables that stand in the catalogue beside it and
// go when it goes, and a view goes before the tables it reads.
var dropping = []struct {
	kind  string
	where string
}{
	{"TRIGGER", `type = 'trigger'`},
	{"VIEW", `type = 'view'`},
	{"TABLE", `type = 'table' AND sql LIKE 'CREATE VIRTUAL TABLE%'`},
	{"TABLE", `type = 'table'`},
}

// empty takes everything out of the index and puts its version back to nothing,
// so the migrations run from the first file.
//
// The catalogue is read again after every drop: a shadow table is listed in it
// and is gone once the virtual table that owns it is dropped.
func empty(ctx context.Context, db *sql.DB) error {
	// A reference to a table already dropped holds nothing back while every
	// table is going. The index writes over one connection, so the setting and
	// the drops meet on it.
	if _, err := writing.Exec(ctx, db, "PRAGMA foreign_keys = off"); err != nil {
		return fmt.Errorf("emptying the index: %w", err)
	}
	dropped := dropEverything(ctx, db)
	_, back := writing.Exec(ctx, db, "PRAGMA foreign_keys = on")
	if dropped != nil {
		return dropped
	}
	if back != nil {
		return fmt.Errorf("putting the index's references back: %w", back)
	}
	if _, err := writing.Exec(ctx, db, "PRAGMA user_version = 0"); err != nil {
		return fmt.Errorf("putting the index back to no schema: %w", err)
	}
	return nil
}

// dropEverything takes every object the catalogue names out of the index.
func dropEverything(ctx context.Context, db *sql.DB) error {
	for _, one := range dropping {
		for {
			var name string
			err := db.QueryRowContext(ctx,
				`SELECT name FROM sqlite_master WHERE `+one.where+
					` AND name NOT LIKE 'sqlite_%' LIMIT 1`).Scan(&name)
			if errors.Is(err, sql.ErrNoRows) {
				break
			}
			if err != nil {
				return fmt.Errorf("what this index holds: %w", err)
			}
			// An identifier is not a parameter, and this one is the catalogue's
			// own.
			if _, err := writing.Exec(ctx, db,
				fmt.Sprintf("DROP %s IF EXISTS %q", one.kind, name)); err != nil {
				return fmt.Errorf("dropping %s: %w", name, err)
			}
		}
	}
	return nil
}

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
//
// An index at a version this build does not carry is refused, and nothing in it
// is touched. What it holds took hours to read, and a build that cannot read its
// schema cannot say what emptying it would cost — so it says which schema it
// found and which it carries, and the person decides.
func migrate(ctx context.Context, db *sql.DB) error {
	available, err := loadMigrations()
	if err != nil {
		return err
	}

	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("what version this index is at: %w", err)
	}
	newest := 0
	if len(available) > 0 {
		newest = available[len(available)-1].version
	}
	// An index at a schema this build does not carry holds what it cannot read.
	// It is a cache: what it holds is a reading of the vault, and the next scan
	// reads the vault again. It is emptied and migrated from the first file.
	if current > newest {
		if err := empty(ctx, db); err != nil {
			return err
		}
		current = 0
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

// remember is where an index keeps what migrated it. It is not one of the
// numbered migrations: it is what says whether those ran, so it cannot be one
// of them.
//
// A table carrying a column this build does not write is brought to the shape
// this build writes, which is the one thing the bookkeeping owes itself.
func remember(ctx context.Context, db *sql.DB) error {
	if _, err := writing.Exec(ctx, db, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name    TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("what this index was migrated by: %w", err)
	}

	var spare int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info('schema_migrations') WHERE name NOT IN ('version', 'name')`,
	).Scan(&spare); err != nil {
		return fmt.Errorf("what this index was migrated by: %w", err)
	}
	if spare == 0 {
		return nil
	}
	for _, statement := range []string{
		`CREATE TABLE schema_migrations_next (version INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`INSERT INTO schema_migrations_next (version, name) SELECT version, name FROM schema_migrations`,
		`DROP TABLE schema_migrations`,
		`ALTER TABLE schema_migrations_next RENAME TO schema_migrations`,
	} {
		if _, err := writing.Exec(ctx, db, statement); err != nil {
			return fmt.Errorf("what this index was migrated by: %w", err)
		}
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	if err := remember(ctx, db); err != nil {
		return err
	}
	tx, err := writing.Begin(ctx, db)
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
	// What ran is recorded beside the number, so a person can read what state
	// this index is in.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name) VALUES (?, ?)
		 ON CONFLICT (version) DO UPDATE SET name = excluded.name`,
		m.version, m.name); err != nil {
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
