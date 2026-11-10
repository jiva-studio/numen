package note

import (
	"context"
	"strings"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// TrackWrites runs a write of a window so that the window waits for it before it closes, and tells
// whether the index followed it.
type TrackWrites struct {
	gate   windowport.WriteGate
	follow windowport.IndexFollow
}

// NewTrackWrites returns the use case over the gate of the window and the index.
func NewTrackWrites(gate windowport.WriteGate, follow windowport.IndexFollow) TrackWrites {
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
	write  Write
	track  TrackWrites
	follow windowport.IndexFollow
}

// NewSaveProse returns the use case over the writer of notes.
func NewSaveProse(write Write, gate windowport.WriteGate, follow windowport.IndexFollow) SaveProse {
	return SaveProse{write: write, track: NewTrackWrites(gate, follow), follow: follow}
}

// Execute saves body to the note at path. A write that reached the vault answers with the file stat
// the caller uses at its next save, and with whether the index did not follow.
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
	create Create
	track  TrackWrites
}

// NewCreateNote returns the use case over the creator of notes.
func NewCreateNote(create Create, gate windowport.WriteGate, follow windowport.IndexFollow) CreateNote {
	return CreateNote{create: create, track: NewTrackWrites(gate, follow)}
}

// Execute creates the note. A note that is on disk is answered with where it is, whether or not the index
// followed.
func (u CreateNote) Execute(ctx context.Context, v domain.Vault, in NewNote) (CreateResult, bool, error) {
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
