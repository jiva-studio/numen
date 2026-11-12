// Package markdown exposes the note parsing types of the notes library to plugins.
package markdown

import (
	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	notesmarkdown "github.com/jiva-studio/numen/libs/note/markdown"
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

func Parse(ref notesdomain.FileStat, raw []byte) notesdomain.Note {
	return notesmarkdown.Parse(ref, raw)
}

func Open(raw []byte) (*Document, error) { return notesmarkdown.Open(raw) }

func OpenBody(body []byte) *Document { return notesmarkdown.OpenBody(body) }

func Create(identifier, body string) []byte { return notesmarkdown.Create(identifier, body) }

func IsFrontmatterStart(text string) bool { return notesmarkdown.IsFrontmatterStart(text) }

func Normalize(text string) string { return notesmarkdown.Normalize(text) }

func WikilinksIn(line string) []notesmarkdown.Wikilink { return notesmarkdown.WikilinksIn(line) }
