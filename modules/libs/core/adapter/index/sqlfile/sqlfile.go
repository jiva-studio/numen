// Package sqlfile loads the SQL that lives beside each repository.
//
// Statements are files: SQL is a language of its own, and anything that reads,
// formats or checks SQL can see it there — the person reviewing a change to it
// included.
package sqlfile

import (
	"fmt"
	"io/fs"
	"strings"
)

// Statements is a set of statements keyed by filename without the extension.
type Statements map[string]string

// Load reads every .sql file in dir.
//
// A file ends in a newline and a statement does not: what follows the semicolon
// is a second statement to the driver. The text is the file with the whitespace
// around it taken off.
func Load(fsys fs.FS, dir string) (Statements, error) {
	out := Statements{}
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return readErr
		}
		name := strings.TrimSuffix(strings.TrimPrefix(p, dir+"/"), ".sql")
		out[name] = strings.TrimSpace(string(raw))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sqlfile: %w", err)
	}
	return out, nil
}

// Get returns a statement by name.
func (s Statements) Get(name string) string {
	return s[name]
}
