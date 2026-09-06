// Package sqlfile loads the SQL that lives beside each repository.
//
// Statements are files: SQL is a language of its own, and anything that reads,
// formats or checks SQL can see it there — the person reviewing a change to it
// included.
package sqlfile

import (
	"io/fs"
	"strings"
)

// Statements is a set of statements keyed by filename without the extension.
type Statements map[string]string

// Load reads every .sql file in dir. It panics: the files are embedded in the
// binary, so a failure here is a build that should not have been produced.
//
// A file ends in a newline and a statement does not: what follows the semicolon
// is a second statement to the driver, and a statement text carrying one is
// compiled again on every call rather than kept compiled. The text is the file
// with the whitespace around it taken off.
func Load(fsys fs.FS, dir string) Statements {
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
		panic("sqlfile: " + err.Error())
	}
	return out
}

// Get returns a statement by name. A missing name is a programming error,
// caught on the first call.
func (s Statements) Get(name string) string {
	stmt, ok := s[name]
	if !ok {
		panic("sqlfile: no statement named " + name)
	}
	return stmt
}
