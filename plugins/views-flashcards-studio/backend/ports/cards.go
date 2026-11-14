package ports

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
)

// DeckReader reads one deck. An error is the vault being out of reach; what is wrong with the file
// itself is the outcome.
type DeckReader interface {
	Deck(ctx context.Context, v domain.Vault, path string) (format.DeckContents, error)
}

// CardNoteReader reads the two notes a card is made of: its deck and its card model.
type CardNoteReader interface {
	DeckReader
	CardModel(ctx context.Context, v domain.Vault, path string) (format.CardModelContents, error)
}

// DeckWriter puts a body in a deck over the file stat the caller read. A landed write whose index
// could not be updated is returned with edit.ErrIndexOutdated beside it.
type DeckWriter interface {
	Deck(ctx context.Context, v domain.Vault, path, body string, stat domain.FileStat) (format.DeckWriteResult, error)
}
