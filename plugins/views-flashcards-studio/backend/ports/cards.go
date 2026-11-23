package ports

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
)

// DeckReader reads one deck. An error is the vault being out of reach; what is wrong with the file
// itself is the outcome.
type DeckReader interface {
	ReadDeck(ctx context.Context, v domain.Vault, path string) (format.DeckContents, error)
}

// CardNoteReader reads the two notes a card is made of: its deck and its card model.
type CardNoteReader interface {
	DeckReader
	ReadCardModel(ctx context.Context, v domain.Vault, path string) (format.CardModelContents, error)
}

// DeckWriter puts a body in a deck over the file stat the caller read. A landed write whose index
// could not be updated is returned with edit.ErrIndexOutdated beside it.
type DeckWriter interface {
	WriteDeck(ctx context.Context, v domain.Vault, path, body string, stat domain.FileStat) (format.DeckWriteResult, error)
}

// LinkEntry is an entry of the `links:` block of a note: the type it carries, the note it points at
// and its role. An empty address takes the entry out.
type LinkEntry struct {
	Type string
	To   domain.Address
	Role domain.LinkRole
}

// LinkWriter points an entry of the `links:` block of a note at another note, over the file stat
// the caller read.
type LinkWriter interface {
	UpdateLink(ctx context.Context, v domain.Vault, path string, entry LinkEntry, stat domain.FileStat) (domain.FileStat, error)
}
