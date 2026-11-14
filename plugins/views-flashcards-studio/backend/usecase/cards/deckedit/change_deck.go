package deckedit

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// deckChange is what a change wrote: the deck as it became and the card ids given to cards that carried none.
type deckChange struct {
	result  EditResult
	cardIDs []format.CardID
}

func changeDeck(
	ctx context.Context, read ports.DeckReader, write ports.DeckWriter, v domain.Vault, pos DeckVersion,
	change func(contents format.DeckContents, file *format.DeckFile) error,
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
	if err != nil && !errors.Is(err, edit.ErrIndexOutdated) {
		return deckChange{}, err
	}
	return deckChange{
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Zero-valued FileStat Size is unused when counting deck cards."
		result:  EditResult{FileStat: wrote.FileStat, Cards: len(file.Deck(domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}).Cards)},
		cardIDs: wrote.Given,
	}, nil
}
