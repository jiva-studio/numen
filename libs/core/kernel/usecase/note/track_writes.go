package note

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// WriteGate admits the writes a window waits for before it closes.
type WriteGate interface {
	// BeginWrite registers a write and reports false once the window is closing, when nothing may be written.
	BeginWrite() bool
	// FinishWrite ends a write that BeginWrite registered.
	FinishWrite()
}

// IndexFollow reports whether the index followed a write and what a written note requires of it.
type IndexFollow interface {
	// IsIndexOutdated reports whether err says the write reached the vault and the index did not follow.
	IsIndexOutdated(err error) bool
	// NoteWritten tells the embedder that a note holds new prose.
	NoteWritten()
}

type openWriteGate struct{}

func (openWriteGate) BeginWrite() bool { return true }
func (openWriteGate) FinishWrite()     {}

type noIndexFollow struct{}

func (noIndexFollow) IsIndexOutdated(error) bool { return false }
func (noIndexFollow) NoteWritten()               {}

// TrackWrites runs a write of a window so that the window waits for it before it closes, and tells
// whether the index followed it.
type TrackWrites struct {
	gate   WriteGate
	follow IndexFollow
}

// NewTrackWrites returns the use case over the gate of the window and the index.
func NewTrackWrites(gate WriteGate, follow IndexFollow) TrackWrites {
	return TrackWrites{gate: gate, follow: follow}
}

// Execute runs write and reports whether it reached the vault while the index did not follow. It fails
// with window.ErrClosing, without running write, once the window is closing.
func (u TrackWrites) Execute(ctx context.Context, write func(ctx context.Context) error) (isIndexOutdated bool, err error) {
	if !u.gate.BeginWrite() {
		return false, windowdomain.ErrClosing
	}
	defer u.gate.FinishWrite()
	err = write(ctx)
	return u.follow.IsIndexOutdated(err), err
}

// SaveProse saves the prose the user wrote into a note as a write of the window, and tells the embedder
// that the note holds new prose.
type SaveProse struct {
	write  *Write
	track  TrackWrites
	follow IndexFollow
}

// NewSaveProse returns the use case over the writer of notes.
func NewSaveProse(write *Write, gate WriteGate, follow IndexFollow) SaveProse {
	return SaveProse{write: write, track: NewTrackWrites(gate, follow), follow: follow}
}

// Execute saves body to the note at path. A write that reached the vault returns the file stat
// the caller uses at its next save, and whether the index did not follow.
func (u SaveProse) Execute(
	ctx context.Context, v domain.Vault, path, body string, baseline *edit.Baseline,
) (domain.FileStat, bool, error) {
	var stat domain.FileStat
	isIndexOutdated, err := u.track.Execute(ctx, func(ctx context.Context) error {
		var saveErr error
		stat, saveErr = u.write.Save(ctx, v, path, body, baseline)
		if saveErr == nil || u.follow.IsIndexOutdated(saveErr) {
			// The written prose requires new vectors. The pending chunks are tracked in the index, so
			// several saves are embedded in one pass.
			u.follow.NoteWritten()
		}
		return saveErr
	})
	return stat, isIndexOutdated, err
}

// CreateNote creates a note as a write of the window. A blank title makes an untitled note.
type CreateNote struct {
	create *Create
	track  TrackWrites
}

// NewCreateNote returns the use case over the creator of notes.
func NewCreateNote(create *Create, gate WriteGate, follow IndexFollow) CreateNote {
	return CreateNote{create: create, track: NewTrackWrites(gate, follow)}
}

// Execute creates the note. A note that is on disk is answered with where it is, whether or not the index
// followed.
// A link that names no note, or one with no address, is ErrInvalidLink and nothing is created.
func (u CreateNote) Execute(ctx context.Context, v domain.Vault, in NewNote) (CreateResult, bool, error) {
	links, err := resolveNamedLinks(ctx, u.create.names, v, in.Named)
	if err != nil {
		return CreateResult{}, false, err
	}
	in.Links = append(slices.Clone(in.Links), links...)
	in.Named = nil
	var made CreateResult
	isIndexOutdated, err := u.track.Execute(ctx, func(ctx context.Context) error {
		var createErr error
		if strings.TrimSpace(in.Title) == "" {
			made, createErr = u.create.ExecuteUntitled(ctx, v, in)
		} else {
			made, createErr = u.create.Execute(ctx, v, in)
		}
		return createErr
	})
	return made, isIndexOutdated, err
}

// ErrInvalidLink is a link that names no note to go to, or whose target has no address.
var ErrInvalidLink = errors.New("the link does not name a note")

// NamedLink is a link to the note at a path of the vault, written as the name of the note or the address
// that the index says is unique.
type NamedLink struct {
	To     string
	Role   domain.LinkRole
	Type   string
	Why    string
	Label  string
	Marker domain.LinkMarker
}

func resolveNamedLink(ctx context.Context, names edit.NameQueries, v domain.Vault, link NamedLink) (domain.Link, error) {
	if link.To == "" {
		return domain.Link{}, fmt.Errorf("%w: a link needs a note to go to", ErrInvalidLink)
	}
	target, err := getLinkAddress(ctx, names, v, link.To)
	if err != nil {
		return domain.Link{}, fmt.Errorf("%w: %w", ErrInvalidLink, err)
	}
	return domain.Link{
		Target: target, Role: link.Role, Type: link.Type, Why: link.Why, Label: link.Label, Marker: link.Marker,
	}, nil
}

func resolveNamedLinks(ctx context.Context, names edit.NameQueries, v domain.Vault, links []NamedLink) ([]domain.Link, error) {
	out := make([]domain.Link, 0, len(links))
	for _, one := range links {
		link, err := resolveNamedLink(ctx, names, v, one)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, nil
}

// WriteLink writes one link into a note as a write of the window: what is already written there is
// kept, and the note at the other end is left alone.
type WriteLink struct {
	links EditLinks
	names edit.NameQueries
	track TrackWrites
}

// NewWriteLink returns the use case over the editing of links and the names that give a link its address.
func NewWriteLink(links EditLinks, names edit.NameQueries, gate WriteGate, follow IndexFollow) WriteLink {
	return WriteLink{links: links, names: names, track: NewTrackWrites(gate, follow)}
}

// Execute writes the link into the note at from. A link that names no note, or one with no address, is
// ErrInvalidLink and nothing is written. It fails with window.ErrClosing once the window is closing.
func (u WriteLink) Execute(ctx context.Context, v domain.Vault, from string, link NamedLink) error {
	resolved, err := resolveNamedLink(ctx, u.names, v, link)
	if err != nil {
		return err
	}
	_, err = u.track.Execute(ctx, func(ctx context.Context) error {
		_, addErr := u.links.Add(ctx, v, from, domain.FileStat{}, resolved)
		return addErr
	})
	return err
}

// CreateRelatedNote creates a note joined to an existing one as a write of the window.
type CreateRelatedNote struct {
	create CreateInRelation
	track  TrackWrites
}

// NewCreateRelatedNote returns the use case over the creation of a note in relation to another.
func NewCreateRelatedNote(create CreateInRelation, gate WriteGate, follow IndexFollow) CreateRelatedNote {
	return CreateRelatedNote{create: create, track: NewTrackWrites(gate, follow)}
}

// Execute creates the note; see CreateInRelation. A note that is on disk is answered with where it is,
// whether or not the index followed. It fails with window.ErrClosing once the window is closing.
func (u CreateRelatedNote) Execute(ctx context.Context, v domain.Vault, in domain.RelatedNote) (CreateResult, bool, error) {
	var made CreateResult
	isIndexOutdated, err := u.track.Execute(ctx, func(ctx context.Context) error {
		var createErr error
		made, createErr = u.create.Execute(ctx, v, in)
		return createErr
	})
	return made, isIndexOutdated, err
}

// RenameNote renames a note as a write of the window.
type RenameNote struct {
	rename *Rename
	track  TrackWrites
}

// NewRenameNote returns the use case over the renaming of a note.
func NewRenameNote(rename *Rename, gate WriteGate, follow IndexFollow) RenameNote {
	return RenameNote{rename: rename, track: NewTrackWrites(gate, follow)}
}

// Execute renames the note at path to title and reports whether the index did not follow. It fails
// with window.ErrClosing once the window is closing.
func (u RenameNote) Execute(ctx context.Context, v domain.Vault, path, title string) (RenameResult, bool, error) {
	var renamed RenameResult
	isIndexOutdated, err := u.track.Execute(ctx, func(ctx context.Context) error {
		var renameErr error
		renamed, renameErr = u.rename.Execute(ctx, v, path, title)
		return renameErr
	})
	return renamed, isIndexOutdated, err
}

// RemoveFile moves a file or a folder of the vault to the trash as a write of the window.
type RemoveFile struct {
	remove *Remove
	track  TrackWrites
}

// NewRemoveFile returns the use case over the removal of a file or a folder.
func NewRemoveFile(remove *Remove, gate WriteGate, follow IndexFollow) RemoveFile {
	return RemoveFile{remove: remove, track: NewTrackWrites(gate, follow)}
}

// Execute moves the file or folder at path to the vault's trash and reports whether the index did not
// follow. It fails with window.ErrClosing once the window is closing.
func (u RemoveFile) Execute(ctx context.Context, v domain.Vault, path string) (RemoveResult, bool, error) {
	var removed RemoveResult
	isIndexOutdated, err := u.track.Execute(ctx, func(ctx context.Context) error {
		var removeErr error
		removed, removeErr = u.remove.Execute(ctx, v, path)
		return removeErr
	})
	return removed, isIndexOutdated, err
}
