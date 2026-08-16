package note

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Linking writes the relationships one note declares.
//
// It works on the `links:` block, which is where a link that carries a role
// lives. A mention in prose is a link too, and is written by writing prose:
// this is not the place for it.
//
// Every change here is to one entry. The entries around it are the person's —
// including ones the application could not read — and come out of a write as
// the bytes they went in as.
type Linking struct {
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
func (u Linking) Add(ctx context.Context, v domain.Vault, from string, add domain.Link, more ...domain.Link) error {
	links := append([]domain.Link{add}, more...)
	for _, link := range links {
		if err := Writable(link); err != nil {
			return err
		}
	}
	return u.editing().apply(ctx, v, from, func(doc *markdown.Document) error {
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
func (u Linking) Update(ctx context.Context, v domain.Vault, from string, to domain.Address, change domain.Link) error {
	if change.Role != "" && !domain.KnownRole(change.Role) {
		return fmt.Errorf("%q is not a role a link can carry", change.Role)
	}
	return u.editing().apply(ctx, v, from, func(doc *markdown.Document) error {
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

// Remove takes a relationship out. The note at the other end is untouched: what
// is removed is one end's account of the relationship, which is all a link ever
// was.
func (u Linking) Remove(ctx context.Context, v domain.Vault, from string, to domain.Address, role domain.LinkRole) error {
	return u.editing().apply(ctx, v, from, func(doc *markdown.Document) error {
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

func (u Linking) editing() editing {
	return editing{readers: u.Readers, writers: u.Writers, index: u.Index, now: u.Now}
}
