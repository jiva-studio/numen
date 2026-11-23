// Package sqlfile loads the SQL that lives beside each repository.
//
// Statements are files: SQL is a language of its own, and tools that read,
// format or check SQL can process it there, as can a reviewer of a change
// to it.
package sqlfile

import (
	"io/fs"
	"strings"
)

func buildRejection(reason string) string {
	return `SELECT 1 FROM "` + strings.ReplaceAll(reason, `"`, `""`) + `"`
}

// Read returns the statement in the file dir/name.sql of fsys, with the whitespace around it
// taken off. A file that cannot be read yields a statement the database rejects, naming the cause.
func Read(fsys fs.FS, dir, name string) string {
	raw, err := fs.ReadFile(fsys, dir+"/"+name+".sql")
	if err != nil {
		return buildRejection("sqlfile: read " + name + ": " + err.Error())
	}
	return strings.TrimSpace(string(raw))
}
