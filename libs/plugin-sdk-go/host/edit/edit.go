// Package edit exposes the note editing types of the notes library to plugins.
package edit

import (
	notesedit "github.com/jiva-studio/numen/libs/notes/edit"
)

type Files = notesedit.Files
type Contents = notesedit.Contents
type ReadOutcome = notesedit.ReadOutcome
type Edit = notesedit.Edit
type Levels = notesedit.Levels

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

var (
	CheckSize      = notesedit.CheckSize
	Read           = notesedit.Read
	GetAddress     = notesedit.GetAddress
	WrapUnlevelled = notesedit.WrapUnlevelled
)
