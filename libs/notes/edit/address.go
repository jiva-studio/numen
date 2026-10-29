package edit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// NameQueries is the query that writing a link, or naming a new note, makes of the vault.
type NameQueries interface {
	// GetNamedPaths returns the paths of every note with one name. More than one
	// path makes a link written by that name ambiguous.
	GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
}

// ErrUnaddressable reports a note no link can address: its name contains a
// character that terminates a link target. Nothing is written.
var ErrUnaddressable = errors.New("no link can address a note with this name")

// GetAddress returns how a note known by path is written into a link: by its
// name, or by its path when the name would match another note.
//
// A name is resolved as an exact path from the root before it is resolved
// relative to the linking note, so a note in a subfolder with the same name as
// a root note is reachable only by writing the path. The vault decides which
// case applies, so GetAddress queries it.
func GetAddress(ctx context.Context, names NameQueries, vaultID domain.VaultID, path string) (domain.Address, error) {
	name := domain.Basename(path)
	if name == "" {
		return domain.Address{}, errors.New("a link needs a note to go to")
	}
	// A link target ends at the first `#` or `|`; the remainder names a heading
	// or the display text. A name containing either character resolves to the
	// text before it, so the note cannot be addressed.
	if strings.ContainsAny(name, "#|") {
		return domain.Address{}, fmt.Errorf("%w: %s", ErrUnaddressable, name)
	}
	shares, err := names.GetNamedPaths(ctx, vaultID, name)
	if err != nil {
		return domain.Address{}, err
	}
	if len(shares) < 2 {
		return domain.Address{Scheme: domain.SchemeName, Value: name}, nil
	}
	return domain.Address{Scheme: domain.SchemeName, Value: withoutExtension(path)}, nil
}

// withoutExtension returns a path in the form a link carries it: the form a
// user writes by hand, and the form the same resolution reads back.
func withoutExtension(path string) string {
	if i := strings.LastIndexByte(path, '.'); i > strings.LastIndexByte(path, '/')+1 {
		return path[:i]
	}
	return path
}
