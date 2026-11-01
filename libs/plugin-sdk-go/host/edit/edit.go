// Package edit exposes the note editing types of the notes library to plugins.
package edit

import (
	"context"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	notesedit "github.com/jiva-studio/numen/libs/notes/edit"
)

type Files = notesedit.Files
type Contents = notesedit.Contents
type ReadOutcome = notesedit.ReadOutcome
type Edit = notesedit.Edit
type Levels = notesedit.Levels
type Clock = notesedit.Clock

const (
	MaxBytes   = notesedit.MaxBytes
	Ok         = notesedit.Ok
	Missing    = notesedit.Missing
	NotANote   = notesedit.NotANote
	TooLarge   = notesedit.TooLarge
	NotText    = notesedit.NotText
	Unreadable = notesedit.Unreadable
)

var (
	ErrNoNote         = notesedit.ErrNoNote
	ErrNotANote       = notesedit.ErrNotANote
	ErrTooLarge       = notesedit.ErrTooLarge
	ErrUnreadable     = notesedit.ErrUnreadable
	ErrUnterminated   = notesedit.ErrUnterminated
	ErrInline         = notesedit.ErrInline
	ErrAnchored       = notesedit.ErrAnchored
	ErrBodyUnwritable = notesedit.ErrBodyUnwritable
	ErrStale          = notesedit.ErrStale
	ErrUnlevelled     = notesedit.ErrUnlevelled
)

func CheckSize(path string, size, maxSize int) error { return notesedit.CheckSize(path, size, maxSize) }

func Read(ctx context.Context, files notesedit.Reader, path string) (Contents, error) {
	return notesedit.Read(ctx, files, path)
}

func GetAddress(
	ctx context.Context, names notesedit.NameQueries, vaultID notesdomain.VaultID, path string,
) (notesdomain.Address, error) {
	return notesedit.GetAddress(ctx, names, vaultID, path)
}

func WrapUnlevelled(err error, paths ...string) error { return notesedit.WrapUnlevelled(err, paths...) }
