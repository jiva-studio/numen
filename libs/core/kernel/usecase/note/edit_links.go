package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// EditLinks writes the relationships one note declares, in the `links:` block,
// where a link that carries a role lives. A mention in prose is written by writing prose.
// Every change applies to one entry. The surrounding entries belong to the
// user, including ones the application could not parse, and are written back
// byte for byte.
type EditLinks struct {
	readers vaultport.VaultReaders
	writers NoteWriting
	index   edit.IndexUpdater
	now     machineport.Clock
}

// NewEditLinks returns the use case that writes a note's relationships. It takes the
// vault readers and writers, the index updater and the clock.
// All four are required for the reason NewWrite requires them: a written link
// that is not indexed cannot be queried, and timestamps come from the injected
// clock and not from the machine clock directly.
func NewEditLinks(
	readers vaultport.VaultReaders, writers NoteWriting, index edit.IndexUpdater, now machineport.Clock,
) EditLinks {
	return EditLinks{readers: readers, writers: writers, index: index, now: now}
}

// Add writes relationships into a note, in one read and one write. Adding a link
// to the same place with the same role again changes nothing.
// At least one link is named, so a call stamps an identifier into a note without one.
// FileStat is what the caller believes is on disk: a changed note is left alone
// and port.ErrStale is returned. The empty file stat skips that check.
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

// Update changes the properties of an existing link, its role, type, label and
// reason, without changing its target.
// Every omitted field is cleared, so the file stat protects against overwriting
// a label the user wrote since the caller read the note.
func (u EditLinks) Update(
	ctx context.Context, v domain.Vault, from string, to domain.Address,
	change domain.Link, stat domain.FileStat,
) (domain.FileStat, error) {
	if change.Role != "" && !domain.IsKnownRole(change.Role) {
		return domain.FileStat{}, fmt.Errorf("%q is not a role a link can have", change.Role)
	}
	e := u.newEdit()
	e.FileStat = stat
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		changed, err := doc.UpdateLink(to, change)
		if err != nil {
			return err
		}
		if changed == 0 {
			return fmt.Errorf("no link to %s is written here", to.GetWritten())
		}
		return nil
	})
}

// PointAt makes a note name one place under a link `type`, in one read and one
// write. An empty address takes the entry out.
// Its role, label and reason are kept, and every other entry of the block is
// written back byte for byte. FileStat, when given, is what the caller believes
// is on disk; a changed note is left alone and port.ErrStale is returned.
func (u EditLinks) PointAt(
	ctx context.Context, v domain.Vault, from string, p LinkPoint,
) (domain.FileStat, error) {
	e := u.newEdit()
	e.FileStat = p.Stat
	return e.shared().PointAt(ctx, v, from, p.Of, p.To, p.Role)
}

// Remove takes a relationship out. The note at the other end is untouched: only
// one end's account of the relationship is removed.
// The record is addressed by its target, its type (the empty type names the
// records that carry none), and its role when one is given.
func (u EditLinks) Remove(
	ctx context.Context, v domain.Vault, from string, key domain.LinkKey, stat domain.FileStat,
) (domain.FileStat, error) {
	e := u.newEdit()
	e.FileStat = stat
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		removed, err := doc.RemoveLink(key)
		if err != nil {
			return err
		}
		if removed == 0 {
			return fmt.Errorf("no link to %s is written here", key.Target.GetWritten())
		}
		return nil
	})
}

// RemoveTargetLinks removes every frontmatter record the note has to the target, of every type, in
// one read and one write; only those of the role when one is given.
func (u EditLinks) RemoveTargetLinks(
	ctx context.Context, v domain.Vault, from string, to domain.Address, role domain.LinkRole, stat domain.FileStat,
) (domain.FileStat, error) {
	e := u.newEdit()
	e.FileStat = stat
	return e.Apply(ctx, v, from, func(doc *markdown.Document) error {
		removed, err := doc.RemoveTargetLinks(to, role)
		if err != nil {
			return err
		}
		if removed == 0 {
			return fmt.Errorf("no link to %s is written here", to.GetWritten())
		}
		return nil
	})
}

// CheckWritable is what a link must carry before anything will write it.
func CheckWritable(link domain.Link) error {
	if !domain.IsKnownRole(link.Role) {
		return fmt.Errorf("%q is not a role a link can have", link.Role)
	}
	if !domain.IsWritableMarker(link.Marker) {
		return fmt.Errorf("%q is not a marker a link can have", link.Marker)
	}
	if link.Target.Value == "" {
		return errors.New("a link needs somewhere to go")
	}
	return nil
}

func (u EditLinks) newEdit() Edit {
	return NewEdit(u.readers, u.writers, u.index, u.now)
}

// LinkPoint is the relationship PointAt writes: the block it is written in (Of),
// the address it points at, its role, and the file stat the caller believes is on
// disk.
type LinkPoint struct {
	Of   string
	To   domain.Address
	Role domain.LinkRole
	Stat domain.FileStat
}

// ignoreIndexChange is the index updater of an installation that keeps no index.
func ignoreIndexChange(context.Context, domain.Vault, []string) error { return nil }

func getIndexUpdater(update edit.IndexUpdater) edit.IndexUpdater {
	if update == nil {
		return ignoreIndexChange
	}
	return update
}

func ignoreMove(context.Context, domain.Vault, domain.Move) error { return nil }

func getTellMove(tell TellMove) TellMove {
	if tell == nil {
		return ignoreMove
	}
	return tell
}

// noReaders is the vault readers of an installation whose files are not open.
type noReaders struct{}

func (noReaders) Open(domain.Vault) (vaultport.VaultReader, error) { return nil, errNotOpen }

func getReaders(readers vaultport.VaultReaders) vaultport.VaultReaders {
	if readers == nil {
		return noReaders{}
	}
	return readers
}
