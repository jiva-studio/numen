package note

import (
	"context"
	"errors"
	"fmt"

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
	Index   Levels
	Now     port.Clock
}

// NewEditLinks is what a note's relationships are written through: the vault it
// is read and written through, what brings it level in the index, and what time
// it is.
//
// All four are named here for the reason NewWrite names them: a link written
// and not levelled is a relationship the vault cannot be asked about, and a
// note stamped off the machine's clock is stamped where nobody said it may be.
func NewEditLinks(
	readers port.VaultReaders, writers port.VaultWriters, index Levels, now port.Clock,
) EditLinks {
	return EditLinks{Readers: readers, Writers: writers, Index: index, Now: now}
}

// Add writes relationships into a note, in one read and one write. A link to
// the same place with the same role is already there, and adding it again
// changes nothing.
//
// One link is named on its own, so there is always at least one: no links is
// still a read and a write, and stamps an identifier into a note without one.
//
// Fingerprint is what the caller believes is on disk, and every one of these
// takes it for the reason PointAt does: a note that changed since it was read
// is left alone and port.ErrStale comes back. The empty fingerprint is a
// caller holding itself to whatever the note is at the moment of the write.
func (u EditLinks) Add(
	ctx context.Context, v domain.Vault, from string, fingerprint domain.Fingerprint,
	add domain.Link, more ...domain.Link,
) (domain.Fingerprint, error) {
	links := append([]domain.Link{add}, more...)
	for _, link := range links {
		if err := Writable(link); err != nil {
			return domain.Fingerprint{}, err
		}
	}
	e := u.editing()
	e.Fingerprint = fingerprint
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		for _, link := range links {
			if err := doc.AddLink(link); err != nil {
				return err
			}
		}
		return nil
	})
}

// Update changes what an existing link says about itself — its role, its type,
// its label, the reason it exists — without moving where it goes.
//
// Every field left out is cleared, so what this writes is decided from what the
// caller read. That is what the fingerprint is for here: the label a person
// wrote in the meantime is not silently thrown away.
func (u EditLinks) Update(
	ctx context.Context, v domain.Vault, from string, to domain.Address,
	change domain.Link, fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	if change.Role != "" && !domain.KnownRole(change.Role) {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a role a link can carry", change.Role)
	}
	e := u.editing()
	e.Fingerprint = fingerprint
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		changed, err := doc.UpdateLink(to, change)
		if err != nil {
			return err
		}
		if changed == 0 {
			return fmt.Errorf("no link to %s is written here", to)
		}
		return nil
	})
}

// PointAt makes a note name one place under a link `type`, in one read and one
// write. An empty address takes the entry out, so the note names none.
//
// The entry is written where a feature reads it: one type, one place. What the
// person wrote on it — its role, its label, why it exists — is kept, and every
// other entry of the block comes out of the write as the bytes it went in as.
//
// Fingerprint, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrStale comes
// back. A write that landed answers with the fingerprint of the file it
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
func (u EditLinks) Remove(
	ctx context.Context, v domain.Vault, from string, to domain.Address,
	role domain.LinkRole, fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	e := u.editing()
	e.Fingerprint = fingerprint
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		removed, err := doc.RemoveLink(to, role)
		if err != nil {
			return err
		}
		if removed == 0 {
			return fmt.Errorf("no link to %s is written here", to)
		}
		return nil
	})
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

func (u EditLinks) editing() Edit {
	return Edit{Readers: u.Readers, Writers: u.Writers, Index: u.Index, Now: u.Now}
}
