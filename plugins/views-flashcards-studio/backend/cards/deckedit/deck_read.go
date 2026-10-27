package deckedit

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
)

func readExistingDeck(ctx context.Context, read Read, v domain.Vault, path string) (DeckContents, error) {
	contents, err := read.Deck(ctx, v, path)
	if err != nil {
		return DeckContents{}, err
	}
	if why := getNotADeckReason(contents); why != "" {
		return DeckContents{}, fmt.Errorf("%s: %s", path, why)
	}
	return contents, nil
}

func getNotADeckReason(read DeckContents) string {
	if read.Outcome == note.Ok && read.Type != format.TypeDeck {
		return "this note is not a deck"
	}
	switch read.Outcome {
	case note.Ok:
		return ""
	case note.TooLarge:
		return fmt.Sprintf("it is %d bytes, larger than the %d a deck is read at; open the file instead",
			read.FileStat.Size, MaxBytes)
	case note.Missing:
		return "there is no note at this path"
	case note.NotANote:
		return "this is not a note the vault holds"
	case note.NotText:
		return "this file is not text: some of it is not valid UTF-8, so open it as a file"
	case note.Unreadable:
		return "the frontmatter of this note cannot be read, so it can be neither read nor written from here"
	}
	return string(read.Outcome)
}
