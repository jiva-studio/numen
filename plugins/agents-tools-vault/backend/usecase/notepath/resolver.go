// Package notepath turns what an agent names a note into the path of its file.
package notepath

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

// Resolver finds the path of a note named by a path or by the name a wikilink uses.
type Resolver struct {
	notes ports.NoteLookup
	files ports.FileReader
}

// NewResolver creates a Resolver that looks names up in the notes and checks files in the files.
func NewResolver(notes ports.NoteLookup, files ports.FileReader) *Resolver {
	return &Resolver{notes: notes, files: files}
}

// Resolve returns the path of the note: the name as given when it ends in .md, else the first note
// with that name, else the name with .md when that file exists, else the name as given.
func (r *Resolver) Resolve(ctx context.Context, name string) string {
	if strings.TrimSpace(name) == "" {
		return name
	}
	name = strings.TrimSpace(name)
	if strings.HasSuffix(name, ".md") {
		return name
	}
	named, err := r.notes.GetNamedPaths(ctx, name)
	if err == nil && len(named) > 0 {
		return named[0]
	}
	if _, err := r.files.StatFile(ctx, name+".md"); err == nil {
		return name + ".md"
	}
	return name
}
