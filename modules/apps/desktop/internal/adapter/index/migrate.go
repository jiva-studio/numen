package index

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
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

// hash is what this migration is, as one value. Comments count: a file whose
// prose changed is a file somebody edited, and an edited migration is the thing
// this guards against.
func (m migration) hash() string {
	sum := sha256.Sum256([]byte(m.body))
	return hex.EncodeToString(sum[:])
}

// migrate brings the database up to the newest migration.
//
// Each migration runs in its own transaction together with the version bump, so
// a failure leaves the database at the last version that fully applied rather
// than half-way through one.
//
// A version number says how many migrations ran, and nothing about which. An
// index whose applied migrations are not the ones in this binary is emptied and
// built from the first: the schema it has is not the schema its number claims,
// and no migration after it can be written to expect either one.
func migrate(ctx context.Context, db *sql.DB) error {
	available, err := loadMigrations()
	if err != nil {
		return err
	}

	current, err := agreed(ctx, db, available)
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

// agreed is how far this index is migrated by the migrations this binary holds.
//
// Every migration records what it was, so the two can be compared. Where they
// differ — an index written by another build of this application, or by this one
// before a migration was changed — the index is emptied and the answer is
// nothing: it is a cache, and a rebuild costs a scan.
func agreed(ctx context.Context, db *sql.DB, available []migration) (int, error) {
	if err := remember(ctx, db); err != nil {
		return 0, err
	}

	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	if current == 0 {
		return 0, nil
	}

	held := map[int]string{}
	rows, err := db.QueryContext(ctx, `SELECT version, hash FROM applied`)
	if err != nil {
		return 0, fmt.Errorf("what this index was migrated by: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var version int
		var hash string
		if err := rows.Scan(&version, &hash); err != nil {
			return 0, err
		}
		held[version] = hash
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, m := range available {
		if m.version > current {
			break
		}
		if held[m.version] == m.hash() {
			continue
		}
		if err := empty(ctx, db); err != nil {
			return 0, err
		}
		return 0, nil
	}
	return current, nil
}

// empty takes the index back to nothing.
//
// Every table goes, the shadow tables of the virtual ones with them, so that
// what is built next is built by the migrations alone. What is thrown away is
// what a scan puts back.
func empty(ctx context.Context, db *sql.DB) error {
	// Outside the transaction, because SQLite ignores this pragma inside one —
	// and a table pointing at one that is not there cannot be dropped while keys
	// are enforced.
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer func() { _, _ = db.ExecContext(ctx, "PRAGMA foreign_keys = ON") }()

	if err := drop(ctx, db); err != nil {
		return err
	}
	return remember(ctx, db)
}

// drop takes every table out in one transaction, so an index part-way emptied is
// not a state anything else can see.
func drop(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	names, err := dropped(ctx, tx)
	if err != nil {
		return err
	}
	for _, name := range names {
		if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS "`+name+`"`); err != nil {
			return fmt.Errorf("empty the index: drop %s: %w", name, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 0"); err != nil {
		return err
	}
	return tx.Commit()
}

// remember is where an index keeps what migrated it. It is not one of the
// numbered migrations: it is what says whether those ran, so it cannot be one of
// them.
func remember(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS applied (
		version INTEGER PRIMARY KEY,
		name    TEXT NOT NULL,
		hash    TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("what this index was migrated by: %w", err)
	}
	return nil
}

// dropped is every table this index holds, the virtual ones named before the
// tables that stand behind them.
func dropped(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT name, sql IS NOT NULL AND sql LIKE 'CREATE VIRTUAL%'
		   FROM sqlite_master
		  WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name <> 'applied'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var virtual, plain []string
	for rows.Next() {
		var name string
		var isVirtual bool
		if err := rows.Scan(&name, &isVirtual); err != nil {
			return nil, err
		}
		if isVirtual {
			virtual = append(virtual, name)
			continue
		}
		plain = append(plain, name)
	}
	// A virtual table takes its shadow tables with it, so it goes first and what
	// is left of the list is what remains.
	return append(virtual, plain...), rows.Err()
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	if err := remember(ctx, db); err != nil {
		return err
	}
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
	// What ran is recorded beside the number, so a later start can tell whether
	// this index was migrated by these migrations.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO applied (version, name, hash) VALUES (?, ?, ?)
		 ON CONFLICT (version) DO UPDATE SET name = excluded.name, hash = excluded.hash`,
		m.version, m.name, m.hash()); err != nil {
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
