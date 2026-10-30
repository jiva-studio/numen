// Package markdown exposes the note parsing types of the notes library to plugins.
package markdown

import (
	notesmarkdown "github.com/jiva-studio/numen/libs/notes/markdown"
)

type Document = notesmarkdown.Document
type Entry = notesmarkdown.Entry
type Fence = notesmarkdown.Fence

const (
	NoRole = notesmarkdown.NoRole
)

var (
	ErrUnterminated = notesmarkdown.ErrUnterminated
	ErrUnreadable   = notesmarkdown.ErrUnreadable
	ErrInline       = notesmarkdown.ErrInline
)

var (
	Parse              = notesmarkdown.Parse
	Open               = notesmarkdown.Open
	OpenBody           = notesmarkdown.OpenBody
	Create             = notesmarkdown.Create
	IsFrontmatterStart = notesmarkdown.IsFrontmatterStart
	Normalise          = notesmarkdown.Normalise
	WikilinksIn        = notesmarkdown.WikilinksIn
)
