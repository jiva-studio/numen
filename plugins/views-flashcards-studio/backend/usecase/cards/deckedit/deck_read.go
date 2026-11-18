package deckedit

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

func readExistingDeck(ctx context.Context, read ports.DeckReader, v domain.Vault, path string) (format.DeckContents, error) {
	contents, err := read.Deck(ctx, v, path)
	if err != nil {
		return format.DeckContents{}, err
	}
	if why := getNotADeckReason(contents); why != "" {
		return format.DeckContents{}, fmt.Errorf("%s: %s", path, why)
	}
	return contents, nil
}

func getNotADeckReason(read format.DeckContents) string {
	if read.Outcome == edit.Ok && read.Type != format.TypeDeck {
		return "this note is not a deck"
	}
	switch read.Outcome {
	case edit.Ok:
		return ""
	case edit.TooLarge:
		return fmt.Sprintf("it is %d bytes, larger than the %d a deck is read at; open the file instead",
			read.FileStat.Size, format.MaxDeckBytes)
	case edit.Missing:
		return "there is no note at this path"
	case edit.NotANote:
		return "this is not a note the vault holds"
	case edit.NotText:
		return "this file is not text: some of it is not valid UTF-8, so open it as a file"
	case edit.Unreadable:
		return "the frontmatter of this note cannot be read, so it can be neither read nor written from here"
	}
	return string(read.Outcome)
}
