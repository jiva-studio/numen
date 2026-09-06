package note

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// NameQueries is the one question writing a link, or naming a new note, asks
// of the vault.
type NameQueries interface {
	// Named is the paths of every note filed under one name. More than one is
	// what makes a link written by that name mean the wrong note.
	Named(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
}

// ErrUnaddressable is a note no link reaches: its name carries a character a
// link is read up to. Nothing is written.
var ErrUnaddressable = errors.New("no link reaches a note named this")

// Addressed is how a note the application knows by path is written into a
// link: by its name, or by its path where the name would mean another note.
//
// A name is read back as an exact path from the root before it is read as a
// neighbour, so a note filed beside a note of the same name at the root is
// reached only by writing the path. Which of the two it is, only the vault
// knows, and it is asked here.
func Addressed(ctx context.Context, names NameQueries, vaultID domain.VaultID, path string) (domain.Address, error) {
	name := domain.Basename(path)
	if name == "" {
		return domain.Address{}, errors.New("a link needs a note to go to")
	}
	// A link is read up to the first `#` or `|`, whichever comes first, and
	// what stands after it names a heading or the words to show. A name
	// carrying one is read back as the name in front of it.
	if strings.ContainsAny(name, "#|") {
		return domain.Address{}, fmt.Errorf("%w: %s", ErrUnaddressable, name)
	}
	shares, err := names.Named(ctx, vaultID, name)
	if err != nil {
		return domain.Address{}, err
	}
	if len(shares) < 2 {
		return domain.Address{Scheme: domain.SchemeName, Value: name}, nil
	}
	return domain.Address{Scheme: domain.SchemeName, Value: withoutExtension(path)}, nil
}

// withoutExtension is a path as a link carries one: what a person writing the
// same link by hand would write, and what the same resolution reads back.
func withoutExtension(path string) string {
	if i := strings.LastIndexByte(path, '.'); i > strings.LastIndexByte(path, '/')+1 {
		return path[:i]
	}
	return path
}
