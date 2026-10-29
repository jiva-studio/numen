package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// EditLinks writes the relationships one note declares.
//
// It works on the `links:` block, which is where a link that carries a role
// lives. A mention in prose is a link too, and is written by writing prose:
// this is not the place for it.
//
// Every change here applies to one entry. The surrounding entries belong to the
// user, including ones the application could not parse, and are written back
// byte for byte.
type EditLinks struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	index   Levels
	now     machineport.Clock
}

// NewEditLinks returns the use case that writes a note's relationships. It takes the
// vault readers and writers, the index updater that makes written links searchable, and
// the clock.
//
// All four are required for the reason NewWrite requires them: a written link
// that is not indexed cannot be queried, and timestamps come from the injected
// clock and not from the machine clock directly.
func NewEditLinks(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels, now machineport.Clock,
) EditLinks {
	return EditLinks{readers: readers, writers: writers, index: index, now: now}
}

// WithReaders replaces the vault readers.
func (u EditLinks) WithReaders(readers vaultport.VaultReaders) EditLinks {
	u.readers = readers
	return u
}

// WithWriters replaces the vault writers.
func (u EditLinks) WithWriters(writers vaultport.VaultWriters) EditLinks {
	u.writers = writers
	return u
}

// Add writes relationships into a note, in one read and one write. A link to
// the same place with the same role is already there, and adding it again
// changes nothing.
//
// One link is named on its own, so there is always at least one: no links is
// still a read and a write, and stamps an identifier into a note without one.
//
// FileStat is what the caller believes is on disk, and every one of these
// takes it for the reason PointAt does: a note that changed since it was read
// is left alone and port.ErrStale comes back. The empty file stat is a
// caller holding itself to whatever the note is at the moment of the write.
func (u EditLinks) Add(
	ctx context.Context, v domain.Vault, from string, stat domain.FileStat,
	add domain.Link, more ...domain.Link,
) (domain.FileStat, error) {
	links := append([]domain.Link{add}, more...)
	for _, link := range links {
		if err := CheckWritable(link); err != nil {
			return domain.FileStat{}, err
		}
	}
	e := u.newEdit()
	e.FileStat = stat
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		for _, link := range links {
			if err := doc.AddLink(link); err != nil {
				return err
			}
		}
		return nil
	})
}

// Update changes the properties of an existing link, its role, type,
// label and reason, without changing its target.
//
// Every omitted field is cleared, so the result is derived from what the
// caller read. The file stat protects against this: a label the user
// wrote in the meantime is not silently overwritten.
func (u EditLinks) Update(
	ctx context.Context, v domain.Vault, from string, to domain.Address,
	change domain.Link, stat domain.FileStat,
) (domain.FileStat, error) {
	if change.Role != "" && !domain.IsKnownRole(change.Role) {
		return domain.FileStat{}, fmt.Errorf("%q is not a role a link can carry", change.Role)
	}
	e := u.newEdit()
	e.FileStat = stat
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
// FileStat, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrStale comes
// back. A write that landed answers with the file stat of the file it
// produced.
func (u EditLinks) PointAt(
	ctx context.Context, v domain.Vault, from, of string,
	to domain.Address, role domain.LinkRole, stat domain.FileStat,
) (domain.FileStat, error) {
	e := u.newEdit()
	e.FileStat = stat
	return e.shared().PointAt(ctx, v, from, of, to, role)
}

// Remove takes a relationship out. The note at the other end is untouched: what
// is removed is one end's account of the relationship, which is all a link ever
// was.
func (u EditLinks) Remove(
	ctx context.Context, v domain.Vault, from string, to domain.Address,
	role domain.LinkRole, stat domain.FileStat,
) (domain.FileStat, error) {
	e := u.newEdit()
	e.FileStat = stat
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

// CheckWritable is what a link must carry before anything will write it.
func CheckWritable(link domain.Link) error {
	if !domain.IsKnownRole(link.Role) {
		return fmt.Errorf("%q is not a role a link can carry", link.Role)
	}
	if link.Target.Value == "" {
		return errors.New("a link needs somewhere to go")
	}
	return nil
}

func (u EditLinks) newEdit() Edit {
	return NewEdit(u.readers, u.writers, u.index, u.now)
}
