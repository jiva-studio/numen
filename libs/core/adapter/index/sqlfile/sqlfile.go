// Package sqlfile loads the SQL that lives beside each repository.
//
// Statements are files: SQL is a language of its own, and anything that reads,
// formats or checks SQL can see it there — the person reviewing a change to it
// included.
package sqlfile

import (
	"io/fs"
	"maps"
	"strings"
)

// Statements is a set of statements keyed by filename without the extension.
type Statements struct {
	byName  map[string]string
	loadErr error
}

// Load reads every .sql file in dir. A file that cannot be read is recorded
// and reported by Get as a statement the database rejects, naming the cause.
//
// A file ends in a newline and a statement does not: what follows the semicolon
// is a second statement to the driver. The text is the file with the whitespace
// around it taken off.
func Load(fsys fs.FS, dir string) Statements {
	out := Statements{byName: map[string]string{}}
	out.loadErr = fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return readErr
		}
		name := strings.TrimSuffix(strings.TrimPrefix(p, dir+"/"), ".sql")
		out.byName[name] = strings.TrimSpace(string(raw))
		return nil
	})
	return out
}

// Get returns a statement by name. A missing name, or a set that failed to
// load, yields a statement that references a table whose name is the failure,
// so the database rejects it with that reason on the first call.
func (s Statements) Get(name string) string {
	if s.loadErr != nil {
		return rejected("sqlfile: load: " + s.loadErr.Error())
	}
	stmt, ok := s.byName[name]
	if !ok {
		return rejected("sqlfile: no statement named " + name)
	}
	return stmt
}

// rejected returns a SELECT from a table named reason, which no database has.
func rejected(reason string) string {
	return `SELECT 1 FROM "` + strings.ReplaceAll(reason, `"`, `""`) + `"`
}

// All returns a copy of the statements keyed by name.
func (s Statements) All() map[string]string { return maps.Clone(s.byName) }
