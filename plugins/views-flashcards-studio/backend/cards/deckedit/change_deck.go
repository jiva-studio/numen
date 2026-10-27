package deckedit

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
)

// deckChange is what a change wrote: the deck as it became and the card ids given to cards that carried none.
type deckChange struct {
	result  EditResult
	cardIDs []format.CardID
}

// changeDeck splices one change into a deck. Every byte the change does not name is written as it arrived.
func changeDeck(
	ctx context.Context, read Read, write Write, v domain.Vault, pos DeckVersion,
	change func(contents DeckContents, file *format.DeckFile) error,
) (deckChange, error) {
	contents, err := readExistingDeck(ctx, read, v, pos.Path)
	if err != nil {
		return deckChange{}, err
	}
	file := format.OpenDeckBody(contents.Raw)
	if err := change(contents, file); err != nil {
		return deckChange{}, err
	}
	wrote, err := write.Deck(ctx, v, pos.Path, file.Body(), pos.FileStat)
	if err != nil && !errors.Is(err, note.ErrUnlevelled) {
		return deckChange{}, err
	}
	return deckChange{
		result:  EditResult{FileStat: wrote.FileStat, Cards: len(file.Deck(domain.FileStat{}).Cards)},
		cardIDs: wrote.Given,
	}, nil
}
