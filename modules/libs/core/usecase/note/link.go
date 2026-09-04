package note

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// EditLinks writes the relationships one note declares.
//
// It works on the `links:` block, which is where a link that carries a role
// lives. A mention in prose is a link too, and is written by writing prose:
// this is not the place for it.
//
// Every change here is to one entry. The entries around it are the person's —
// including ones the application could not read — and come out of a write as
// the bytes they went in as.
type EditLinks struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	Now     func() time.Time
}

// Add writes relationships into a note, in one read and one write. A link to
// the same place with the same role is already there, and adding it again
// changes nothing.
//
// One link is named on its own, so there is always at least one: no links is
// still a read and a write, and stamps an identifier into a note without one.
func (u EditLinks) Add(ctx context.Context, v domain.Vault, from string, add domain.Link, more ...domain.Link) error {
	links := append([]domain.Link{add}, more...)
	for _, link := range links {
		if err := Writable(link); err != nil {
			return err
		}
	}
	_, err := u.editing().Apply(ctx, v, from, func(doc *markdown.Document) error {
		for _, link := range links {
			if err := doc.AddLink(link); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// Update changes what an existing link says about itself — its role, its type,
// its label, the reason it exists — without moving where it goes.
func (u EditLinks) Update(ctx context.Context, v domain.Vault, from string, to domain.Address, change domain.Link) error {
	if change.Role != "" && !domain.KnownRole(change.Role) {
		return fmt.Errorf("%q is not a role a link can carry", change.Role)
	}
	_, err := u.editing().Apply(ctx, v, from, func(doc *markdown.Document) error {
		changed, err := doc.UpdateLink(to, change)
		if err != nil {
			return err
		}
		if changed == 0 {
			return fmt.Errorf("no link to %s is written here", to)
		}
		return nil
	})
	return err
}

// PointAt makes a note name one place under a link `type`, in one read and one
// write. An empty address takes the entry out, so the note names none.
//
// The entry is written where a feature reads it: one type, one place. What the
// person wrote on it — its role, its label, why it exists — is kept, and every
// other entry of the block comes out of the write as the bytes it went in as.
//
// Fingerprint, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrChanged comes
// back. What comes back otherwise is the fingerprint of the file this write
// produced.
func (u EditLinks) PointAt(
	ctx context.Context, v domain.Vault, from, of string,
	to domain.Address, role domain.LinkRole, fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	if of == "" {
		return domain.Fingerprint{}, errors.New("a link is pointed at under a type")
	}
	if to.Value != "" && !domain.KnownRole(role) {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a role a link can carry", role)
	}
	e := u.editing()
	e.Fingerprint = fingerprint
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		return doc.SetLinkOfType(of, to, role)
	})
}

// Remove takes a relationship out. The note at the other end is untouched: what
// is removed is one end's account of the relationship, which is all a link ever
// was.
func (u EditLinks) Remove(ctx context.Context, v domain.Vault, from string, to domain.Address, role domain.LinkRole) error {
	_, err := u.editing().Apply(ctx, v, from, func(doc *markdown.Document) error {
		removed, err := doc.RemoveLink(to, role)
		if err != nil {
			return err
		}
		if removed == 0 {
			return fmt.Errorf("no link to %s is written here", to)
		}
		return nil
	})
	return err
}

// NameQueries is the one question writing a link asks of the vault.
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

// Writable is what a link must carry before anything will write it.
func Writable(link domain.Link) error {
	if !domain.KnownRole(link.Role) {
		return fmt.Errorf("%q is not a role a link can carry", link.Role)
	}
	if link.Target.Value == "" {
		return errors.New("a link needs somewhere to go")
	}
	return nil
}

func (u EditLinks) editing() Editing {
	return Editing{Readers: u.Readers, Writers: u.Writers, Index: u.Index, Now: u.Now}
}
