// Package edit exposes the note editing types of the notes library to plugins.
package edit

import (
	"context"

	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	notesedit "github.com/jiva-studio/numen/libs/note/edit"
)

type Files = notesedit.Files
type Contents = notesedit.Contents
type ReadOutcome = notesedit.ReadOutcome
type Edit = notesedit.Edit
type IndexUpdater = notesedit.IndexUpdater

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
	ErrAnchored       = notesedit.ErrAnchored
	ErrBodyUnwritable = notesedit.ErrBodyUnwritable
	ErrStale          = notesedit.ErrStale
	ErrIndexOutdated  = notesedit.ErrIndexOutdated
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

func WrapIndexOutdated(err error, paths ...string) error {
	return notesedit.WrapIndexOutdated(err, paths...)
}
