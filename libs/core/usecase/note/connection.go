package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/note/query"
)

// connectionRoles are the roles whose records make up a connection.
var connectionRoles = [...]domain.LinkRole{domain.RoleParent, domain.RoleChild, domain.RoleJump}

// noteChange is the edit one note of a connection receives.
type noteChange struct {
	path   string
	change func(*markdown.Document) error
}

// connectionEditor writes the records of a connection into the notes that hold
// them. A write either leaves every note changed or none.
type connectionEditor struct {
	readers   vaultport.VaultReaders
	writers   vaultport.VaultWriters
	index     Levels
	now       machineport.Clock
	links     noteport.LinkQueries
	addresses *GetLinkAddress

	connections query.GetConnection
}

// getAddresses returns every address under which `from` writes a link to `to`,
// with the canonical address first.
func (e connectionEditor) getAddresses(
	ctx context.Context, v domain.Vault, from, to string,
) ([]domain.Address, error) {
	canonical, err := e.addresses.Execute(ctx, v, to)
	if err != nil {
		return nil, err
	}
	out := []domain.Address{canonical}
	links, err := e.links.GetLinks(ctx, v.ID, from)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if l.ToVault != v.ID || l.To != to {
			continue
		}
		isKnown := false
		for _, a := range out {
			isKnown = isKnown || a == l.Target
		}
		if !isKnown {
			out = append(out, l.Target)
		}
	}
	return out, nil
}

// getParents returns the notes that have path as a child, from both ends of the
// links: `parent: X` written in path, and `child: path` written in X.
func (e connectionEditor) getParents(ctx context.Context, v domain.Vault, path string) ([]string, error) {
	links, err := e.links.GetLinks(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	var parents []string
	for _, l := range links {
		if l.ToVault == v.ID && l.Role == domain.RoleParent {
			parents = append(parents, l.To)
		}
	}
	backlinks, err := e.links.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	for _, l := range backlinks {
		if l.Role == domain.RoleChild {
			parents = append(parents, l.From)
		}
	}
	return parents, nil
}

// check refuses a connection that cannot be written: a malformed one, or one that
// would put a note above itself.
func (e connectionEditor) check(ctx context.Context, v domain.Vault, c domain.Connection) error {
	if err := c.Validate(); err != nil {
		return err
	}
	return c.CheckAcyclic(func(path string) ([]string, error) { return e.getParents(ctx, v, path) })
}

// setChange makes the note hold `side` as its record about `other`, or no record
// of type `kind` when side is nil.
func (e connectionEditor) setChange(
	ctx context.Context, v domain.Vault, from, other, kind string, side *domain.Side,
) (noteChange, error) {
	addresses, err := e.getAddresses(ctx, v, from, other)
	if err != nil {
		return noteChange{}, err
	}
	change := func(doc *markdown.Document) error {
		if side == nil {
			for _, target := range addresses {
				for _, role := range connectionRoles {
					if _, err := doc.RemoveLink(domain.LinkKey{Target: target, Role: role, Type: kind}); err != nil {
						return err
					}
				}
			}
			return nil
		}
		return doc.SetLink(domain.Link{
			Target: addresses[0], Role: side.Role, Type: side.Type, Why: side.Why,
			Label: side.Label, Marker: side.Marker,
		}, addresses[1:]...)
	}
	return noteChange{path: from, change: change}, nil
}

// getChanges returns the edit of each of the two notes of a connection.
func (e connectionEditor) getChanges(
	ctx context.Context, v domain.Vault, c domain.Connection, kind string,
) ([]noteChange, error) {
	ab, err := e.setChange(ctx, v, c.A, c.B, kind, c.AB)
	if err != nil {
		return nil, err
	}
	ba, err := e.setChange(ctx, v, c.B, c.A, kind, c.BA)
	if err != nil {
		return nil, err
	}
	return []noteChange{ab, ba}, nil
}

// apply runs the changes against their notes. Every change is first applied to
// the note's current content in memory, so a note that cannot be edited leaves
// all of them unwritten. A write that fails after an earlier one landed restores
// the earlier notes to the content they had.
func (e connectionEditor) apply(ctx context.Context, v domain.Vault, changes []noteChange) error {
	reader, err := e.readers.Open(v)
	if err != nil {
		return err
	}
	originals := make([][]byte, len(changes))
	for i, c := range changes {
		raw, err := reader.Read(ctx, c.path)
		if err != nil {
			return fmt.Errorf("read %s: %w", c.path, mapMissingNote(err))
		}
		doc, err := markdown.Open(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", c.path, err)
		}
		if err := c.change(doc); err != nil {
			return fmt.Errorf("%s: %w", c.path, err)
		}
		originals[i] = raw
	}

	edit := NewEdit(e.readers, e.writers, nil, e.now)
	stats := make([]domain.FileStat, 0, len(changes))
	for i, c := range changes {
		stat, err := edit.Apply(ctx, v, c.path, c.change)
		if err != nil {
			rollbackErr := e.restore(ctx, v, changes[:i], originals, stats)
			return errors.Join(fmt.Errorf("write %s: %w", c.path, err), rollbackErr)
		}
		stats = append(stats, stat)
	}
	return e.level(ctx, v, changes)
}

// restore writes the original content back into the notes already written.
func (e connectionEditor) restore(
	ctx context.Context, v domain.Vault, landed []noteChange, originals [][]byte, stats []domain.FileStat,
) error {
	if len(landed) == 0 {
		return nil
	}
	writer, err := e.writers.Open(v)
	if err != nil {
		return err
	}
	var failures []error
	for i := len(landed) - 1; i >= 0; i-- {
		if _, err := writer.Write(ctx, landed[i].path, originals[i], stats[i]); err != nil {
			failures = append(failures, fmt.Errorf("restore %s: %w", landed[i].path, err))
		}
	}
	if err := e.level(ctx, v, landed); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

// level brings the index level with the notes written.
func (e connectionEditor) level(ctx context.Context, v domain.Vault, changes []noteChange) error {
	if e.index == nil {
		return nil
	}
	paths := make([]string, len(changes))
	for i, c := range changes {
		paths[i] = c.path
	}
	return WrapUnlevelled(e.index(ctx, v, paths), paths...)
}

// getKind is the type the records of a connection are addressed by: the type of
// a side it holds.
func getKind(c domain.Connection) string {
	switch {
	case c.AB != nil:
		return c.AB.Type
	case c.BA != nil:
		return c.BA.Type
	default:
		return ""
	}
}

// SaveConnection makes the records between two notes exactly the ones given.
type SaveConnection struct {
	editor connectionEditor
}

// NewSaveConnection builds the use case over the vault, the index that is brought
// level after a write, the link queries, and the clock.
func NewSaveConnection(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels,
	now machineport.Clock, links noteport.LinkQueries, addresses *GetLinkAddress,
) SaveConnection {
	return SaveConnection{editor: connectionEditor{
		readers: readers, writers: writers, index: index, now: now, links: links, addresses: addresses,
		connections: query.NewGetConnection(links),
	}}
}

// Execute replaces the records of both notes. A side that is nil is removed from
// its note. A connection with no side is refused: RemoveConnection removes one.
func (u SaveConnection) Execute(ctx context.Context, v domain.Vault, c domain.Connection) error {
	if c.IsEmpty() {
		return fmt.Errorf("%w: a connection with no record is removed, not saved", domain.ErrInvalidConnection)
	}
	if err := u.editor.check(ctx, v, c); err != nil {
		return err
	}
	changes, err := u.editor.getChanges(ctx, v, c, getKind(c))
	if err != nil {
		return err
	}
	return u.editor.apply(ctx, v, changes)
}

// RemoveConnection removes the records of one type that two notes write about each
// other.
type RemoveConnection struct {
	editor connectionEditor
}

// NewRemoveConnection builds the use case over the same collaborators as
// NewSaveConnection.
func NewRemoveConnection(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels,
	now machineport.Clock, links noteport.LinkQueries, addresses *GetLinkAddress,
) RemoveConnection {
	return RemoveConnection{editor: connectionEditor{
		readers: readers, writers: writers, index: index, now: now, links: links, addresses: addresses,
		connections: query.NewGetConnection(links),
	}}
}

// Execute removes the records of the type from both notes. The empty type names the
// records that carry none.
func (u RemoveConnection) Execute(ctx context.Context, v domain.Vault, a, b, kind string) error {
	c := domain.Connection{A: a, B: b}
	if err := c.Validate(); err != nil {
		return err
	}
	changes, err := u.editor.getChanges(ctx, v, c, kind)
	if err != nil {
		return err
	}
	return u.editor.apply(ctx, v, changes)
}

// RemoveSide removes the record one note writes about another and leaves the other
// note as it is.
type RemoveSide struct {
	editor connectionEditor
}

// NewRemoveSide builds the use case over the same collaborators as
// NewSaveConnection.
func NewRemoveSide(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels,
	now machineport.Clock, links noteport.LinkQueries, addresses *GetLinkAddress,
) RemoveSide {
	return RemoveSide{editor: connectionEditor{
		readers: readers, writers: writers, index: index, now: now, links: links, addresses: addresses,
		connections: query.NewGetConnection(links),
	}}
}

// Execute removes the records of the type that `from` writes about `to`.
func (u RemoveSide) Execute(ctx context.Context, v domain.Vault, from, to, kind string) error {
	if err := (domain.Connection{A: from, B: to}).Validate(); err != nil {
		return err
	}
	change, err := u.editor.setChange(ctx, v, from, to, kind, nil)
	if err != nil {
		return err
	}
	return u.editor.apply(ctx, v, []noteChange{change})
}

// Join writes the record one note holds about another, keeping the record the
// other note already holds about it.
type Join struct {
	editor connectionEditor
}

// NewJoin builds the use case over the same collaborators as NewSaveConnection.
func NewJoin(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels,
	now machineport.Clock, links noteport.LinkQueries, addresses *GetLinkAddress,
) Join {
	return Join{editor: connectionEditor{
		readers: readers, writers: writers, index: index, now: now, links: links, addresses: addresses,
		connections: query.NewGetConnection(links),
	}}
}

// Execute makes `from` write `side` about `to`. The record of `to` about `from` is
// kept, and must be the inverse of the new one.
func (u Join) Execute(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error {
	c := domain.Connection{A: from, B: to, AB: &side}
	if err := c.Validate(); err != nil {
		return err
	}
	existing, err := u.editor.connections.Execute(ctx, v, from, to, side.Type)
	if err != nil {
		return err
	}
	c.BA = existing.BA
	if err := u.editor.check(ctx, v, c); err != nil {
		return err
	}
	change, err := u.editor.setChange(ctx, v, from, to, side.Type, &side)
	if err != nil {
		return err
	}
	return u.editor.apply(ctx, v, []noteChange{change})
}
