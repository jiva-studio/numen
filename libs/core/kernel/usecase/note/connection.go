package note

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

func getConnectionRoles() [3]domain.LinkRole {
	return [...]domain.LinkRole{domain.RoleParent, domain.RoleChild, domain.RoleJump}
}

// noteChange is the edit one note of a connection receives.
type noteChange struct {
	path   string
	change func(*markdown.Document) error
}

// connectionEditor writes the records of a connection into the notes that hold
// them. A write either leaves every note changed or none.
type connectionEditor struct {
	readers vaultport.VaultReaders
	writers NoteWriting
	index   edit.IndexUpdater
	now     machineport.Clock
	links   noteport.LinkQueries
	names   edit.NameQueries
	gate    windowport.WriteGate
	follow  windowport.IndexFollow
}

// ConnectionDeps are what the use cases that write connections work over. A write passes Gate. Where Follow
// is nil the index is taken to follow every write. Where Names is nil, a link names a note by its basename.
type ConnectionDeps struct {
	Readers vaultport.VaultReaders
	Writers NoteWriting
	Index   edit.IndexUpdater
	Now     machineport.Clock
	Links   noteport.LinkQueries
	Names   edit.NameQueries
	Gate    windowport.WriteGate
	Follow  windowport.IndexFollow
}

func newConnectionEditor(d ConnectionDeps) connectionEditor {
	return connectionEditor{
		readers: d.Readers, writers: d.Writers, index: d.Index, now: d.Now, links: d.Links, names: d.Names,
		gate: d.Gate, follow: d.Follow,
	}
}

func (e connectionEditor) run(ctx context.Context, write func(ctx context.Context) error) error {
	_, err := writeUnder(e.gate, func() (struct{}, error) { return struct{}{}, write(ctx) })
	if e.follow != nil && e.follow.IsIndexOutdated(err) {
		return nil
	}
	return err
}

func (e connectionEditor) getAddresses(
	ctx context.Context, v domain.Vault, from, to string,
) ([]domain.Address, error) {
	canonical, err := getLinkAddress(ctx, e.names, v, to)
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

func (e connectionEditor) check(ctx context.Context, v domain.Vault, c domain.Connection) error {
	if err := c.Validate(); err != nil {
		return err
	}
	return c.CheckAcyclic(func(path string) ([]string, error) { return e.getParents(ctx, v, path) })
}

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
				for _, role := range getConnectionRoles() {
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

func (e connectionEditor) apply(ctx context.Context, v domain.Vault, changes []noteChange) error {
	reader, err := e.readers.Open(v)
	if err != nil {
		return err
	}
	originals := make([][]byte, len(changes))
	for i, c := range changes {
		raw, err := reader.Read(ctx, c.path)
		if err != nil {
			return fmt.Errorf("read %s: %w", c.path, edit.MapMissingNote(err))
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
	return e.updateIndex(ctx, v, changes)
}

func (e connectionEditor) restore(
	ctx context.Context, v domain.Vault, landed []noteChange, originals [][]byte, stats []domain.FileStat,
) error {
	if len(landed) == 0 {
		return nil
	}
	writer, err := e.writers.OpenNoteWriter(v)
	if err != nil {
		return err
	}
	var failures []error
	for i, l := range slices.Backward(landed) {
		if _, err := writer.Write(ctx, l.path, originals[i], stats[i]); err != nil {
			failures = append(failures, fmt.Errorf("restore %s: %w", l.path, err))
		}
	}
	if err := e.updateIndex(ctx, v, landed); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

func (e connectionEditor) updateIndex(ctx context.Context, v domain.Vault, changes []noteChange) error {
	if e.index == nil {
		return nil
	}
	paths := make([]string, len(changes))
	for i, c := range changes {
		paths[i] = c.path
	}
	// The files are written, so the index follows them even when the caller is canceled.
	return edit.WrapIndexOutdated(e.index(context.WithoutCancel(ctx), v, paths), paths...)
}

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

// NewSaveConnection builds the use case over d.
func NewSaveConnection(d ConnectionDeps) SaveConnection {
	return SaveConnection{editor: newConnectionEditor(d)}
}

// Execute replaces the records of both notes. A side that is nil is removed from
// its note. A connection with no side is refused: RemoveConnection removes one.
func (u SaveConnection) Execute(ctx context.Context, v domain.Vault, c domain.Connection) error {
	return u.editor.run(ctx, func(ctx context.Context) error {
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
	})
}

// RemoveConnection removes the records of one type that two notes write about each
// other.
type RemoveConnection struct {
	editor connectionEditor
}

// NewRemoveConnection builds the use case over d.
func NewRemoveConnection(d ConnectionDeps) RemoveConnection {
	return RemoveConnection{editor: newConnectionEditor(d)}
}

// Execute removes the records of the type from both notes. The empty type names the
// records that carry none.
func (u RemoveConnection) Execute(ctx context.Context, v domain.Vault, a, b, kind string) error {
	return u.editor.run(ctx, func(ctx context.Context) error {
		c := domain.Connection{A: a, B: b}
		if err := c.Validate(); err != nil {
			return err
		}
		changes, err := u.editor.getChanges(ctx, v, c, kind)
		if err != nil {
			return err
		}
		return u.editor.apply(ctx, v, changes)
	})
}

// RemoveSide removes the record one note writes about another and leaves the other
// note as it is.
type RemoveSide struct {
	editor connectionEditor
}

// NewRemoveSide builds the use case over d.
func NewRemoveSide(d ConnectionDeps) RemoveSide {
	return RemoveSide{editor: newConnectionEditor(d)}
}

// Execute removes the records of the type that `from` writes about `to`.
func (u RemoveSide) Execute(ctx context.Context, v domain.Vault, from, to, kind string) error {
	return u.editor.run(ctx, func(ctx context.Context) error {
		if err := (domain.Connection{A: from, B: to}).Validate(); err != nil {
			return err
		}
		change, err := u.editor.setChange(ctx, v, from, to, kind, nil)
		if err != nil {
			return err
		}
		return u.editor.apply(ctx, v, []noteChange{change})
	})
}

// Join writes the record one note holds about another, keeping the record the
// other note already holds about it.
type Join struct {
	editor connectionEditor
}

// NewJoin builds the use case over d.
func NewJoin(d ConnectionDeps) Join {
	return Join{editor: newConnectionEditor(d)}
}

// Execute makes `from` write `side` about `to`. The record of `to` about `from` is
// kept, and must be the inverse of the new one.
func (u Join) Execute(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error {
	return u.editor.run(ctx, func(ctx context.Context) error {
		c := domain.Connection{A: from, B: to, AB: &side}
		if err := c.Validate(); err != nil {
			return err
		}
		links, err := u.editor.links.GetLinks(ctx, v.ID, to)
		if err != nil {
			return err
		}
		c.BA = domain.FindSide(links, v.ID, from, side.Type)
		if err := u.editor.check(ctx, v, c); err != nil {
			return err
		}
		change, err := u.editor.setChange(ctx, v, from, to, side.Type, &side)
		if err != nil {
			return err
		}
		return u.editor.apply(ctx, v, []noteChange{change})
	})
}

func getLinkAddress(ctx context.Context, names edit.NameQueries, v domain.Vault, path string) (domain.Address, error) {
	if names == nil {
		return domain.Address{Scheme: domain.SchemeName, Value: domain.Basename(path)}, nil
	}
	return edit.GetAddress(ctx, names, v.ID, path)
}
