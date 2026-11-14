package format

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
)

// MaxDeckBytes is the largest deck that is read.
//
// A deck is a single file holding a replacement for a folder of notes, so
// its limit is several times the limit of a note. The size is read from the
// file metadata before the file is opened.
const MaxDeckBytes = 8 << 20

// DeckContents is the result of reading one deck.
type DeckContents struct {
	Path       string
	Outcome    edit.ReadOutcome
	Type       domain.NoteType
	Body       Deck
	Raw        string
	CardModels map[string]string
	FileStat   domain.FileStat
}

// CardModelContents is the result of reading one card model.
type CardModelContents struct {
	Path     string
	Title    string
	Outcome  edit.ReadOutcome
	Type     domain.NoteType
	Body     CardModel
	FileStat domain.FileStat
}

// DeckWriteResult is the file stat of a written deck, which is what the caller presents at its next
// write, and the card id every card that carried none was given.
type DeckWriteResult struct {
	FileStat domain.FileStat
	Given    []CardID
}

// NoteLinks is what one note is joined to: what it points at, and what points at it.
type NoteLinks struct {
	Links     []domain.ResolvedLink
	Backlinks []domain.ResolvedLink
}
