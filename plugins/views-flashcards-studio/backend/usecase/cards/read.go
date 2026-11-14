package cards

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// Read reads a deck or a card model from the vault.
type Read struct {
	readers ports.VaultReaders
	links   ports.LinkQueries
}

// NewRead is what a deck or a card model is read through: the vault its file is
// read out of, and where the wikilink each card names its card model by lands.
func NewRead(readers ports.VaultReaders, links ports.LinkQueries) Read {
	return Read{readers: readers, links: links}
}

// Deck reads the deck at path.
//
// An error means the vault is unavailable. A problem with the file itself is
// reported in the outcome.
func (u Read) Deck(ctx context.Context, v domain.Vault, path string) (format.DeckContents, error) {
	out := format.DeckContents{
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Initial FileStat is overwritten before Deck returns."
		Path: path, Outcome: "", Type: "", Body: format.Deck{Ref: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, Preamble: "", Tail: "", Sections: nil, Cards: nil, Problems: nil},
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Initial FileStat is overwritten before Deck returns."
		Raw: "", CardModels: nil, FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
	}
	n, ref, outcome, err := u.readNote(ctx, v, path, format.MaxDeckBytes)
	if err != nil {
		return format.DeckContents{}, err
	}
	out.FileStat, out.Outcome, out.Type = ref, outcome, n.Type
	// An outcome other than Ok and TooLarge is reported in the outcome above and
	// leaves the body empty.
	//exhaustive:ignore
	switch outcome {
	case edit.Ok:
		out.Body, out.Raw = format.ReadDeck(n), n.Body
		out.CardModels, err = getCardModelPaths(ctx, u.links, v.ID, path, out.Body)
		if err != nil {
			return format.DeckContents{}, err
		}
		// A card is laid out by the card model its wikilink lands on, so a name
		// reaching a note that is not one is a card no template shows.
		_, ordinary, err := u.readCardModels(ctx, v, out.CardModels)
		if err != nil {
			return format.DeckContents{}, err
		}
		out.Body.Problems = append(out.Body.Problems, getNonCardModelProblems(out.Body, ordinary)...)
	case edit.TooLarge:
		out.Body = format.Deck{Ref: ref,
			Preamble: "",
			Tail:     "",
			Sections: nil,
			Cards:    nil, Problems: []format.Problem{format.OnFile(
				format.FaultTooLarge,
				fmt.Sprintf("this deck is %d bytes, and %d is the most one is read at", ref.Size, format.MaxDeckBytes),
			)}}
	}
	return out, nil
}

// GetCardCardModels returns the card model each card of a deck is cut by, keyed by the wikilink
// name written under the card heading. A name that resolves to a note that is not a card model
// maps to an empty card model. The deck and the card models are read together, which determines
// the first field of each card.
func (u Read) GetCardCardModels(
	ctx context.Context, v domain.Vault, path string, d format.Deck,
) (map[string]format.CardModel, error) {
	pos, err := getCardModelPaths(ctx, u.links, v.ID, path, d)
	if err != nil {
		return nil, err
	}
	delta, _, err := u.readCardModels(ctx, v, pos)
	return delta, err
}

func getCardModelPaths(
	ctx context.Context, links ports.LinkQueries, vaultID domain.VaultID, path string, d format.Deck,
) (map[string]string, error) {
	if links == nil {
		return nil, nil
	}
	content := make([]string, 0, len(d.Cards))
	for _, card := range d.Cards {
		if card.CardModelLink != "" {
			content = append(content, card.CardModelLink)
		}
	}
	if len(content) == 0 {
		return nil, nil
	}
	reached, err := links.ResolveLinks(ctx, vaultID, path, content)
	if err != nil {
		return nil, err
	}
	pos := make(map[string]string, len(reached))
	for raw, one := range reached {
		pos[raw] = one.To
	}
	return pos, nil
}

func (u Read) readCardModels(
	ctx context.Context, v domain.Vault, pos map[string]string,
) (cardModels map[string]format.CardModel, ordinary map[string]bool, err error) {
	if len(pos) == 0 {
		return nil, nil, nil
	}
	read := make(map[string]format.CardModel, len(pos))
	isLoose := make(map[string]bool, len(pos))
	cardModels = make(map[string]format.CardModel, len(pos))
	ordinary = make(map[string]bool, len(pos))
	for content, path := range pos {
		target, observed := read[path]
		if !observed {
			found, err := u.CardModel(ctx, v, path)
			if err != nil {
				return nil, nil, err
			}
			if found.Type == format.TypeCardModel {
				target = found.Body
			}
			read[path] = target
			isLoose[path] = found.Outcome == edit.Ok && found.Type != format.TypeCardModel
		}
		cardModels[content] = target
		ordinary[content] = isLoose[path]
	}
	return cardModels, ordinary, nil
}

func getNonCardModelProblems(d format.Deck, ordinary map[string]bool) []format.Problem {
	var out []format.Problem
	for item, card := range d.Cards {
		if !ordinary[card.CardModelLink] {
			continue
		}
		out = append(out, format.OnCard(item, format.FaultNotACardModel,
			card.CardModelLink+" is a note and not a card model, so this card is shown by no template"))
	}
	return out
}

func (u Read) CardModel(ctx context.Context, v domain.Vault, path string) (format.CardModelContents, error) {
	out := format.CardModelContents{
		Path: path, Title: "", Outcome: "", Type: "",
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Initial FileStat is overwritten before CardModel returns."
		Body: format.CardModel{Ref: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
			Fields: nil, Preamble: "", Tail: "", Templates: nil, Problems: nil},
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Initial FileStat is overwritten before CardModel returns."
		FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
	}
	n, ref, outcome, err := u.readNote(ctx, v, path, edit.MaxBytes)
	if err != nil {
		return format.CardModelContents{}, err
	}
	out.FileStat, out.Outcome, out.Type = ref, outcome, n.Type
	out.Title = n.Title
	if outcome == edit.Ok {
		out.Body = format.ReadCardModel(n)
	}
	return out, nil
}

func (u Read) readNote(
	ctx context.Context, v domain.Vault, path string, maxSize int64,
) (domain.Note, domain.FileStat, edit.ReadOutcome, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.Note{}, domain.FileStat{}, "", err
	}

	// The vault reports what is at a path without opening it: a note, a file of
	// another type, or nothing. The metadata includes the size, which keeps a
	// deck over the limit unread.
	ref, err := reader.StatFile(ctx, path)
	target := err == nil
	switch {
	case target:
		if ref.Kind != domain.KindNote {
			// gomutants:disable-next-line RETURN_ZERO reason="emptyNote returns the zero value of domain.Note."
			return emptyNote(), ref, edit.NotANote, nil
		}
		if ref.Size > maxSize {
			// gomutants:disable-next-line RETURN_ZERO reason="emptyNote returns the zero value of domain.Note."
			return emptyNote(), ref, edit.TooLarge, nil
		}
	case errors.Is(err, ports.ErrNotANote):
		// gomutants:disable-next-line RETURN_ZERO,INTEGER_DECREMENT,INTEGER_INCREMENT reason="emptyNote returns the zero value of domain.Note and NotANote callers ignore FileStat Size."
		return emptyNote(), domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, edit.NotANote, nil
	case !ports.IsNotExist(err):
		return domain.Note{}, domain.FileStat{}, "", fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case ports.IsNotExist(err):
		// gomutants:disable-next-line RETURN_ZERO reason="emptyNote returns the zero value of domain.Note."
		return emptyNote(), ref, edit.Missing, nil
	case !target && err == nil:
		// gomutants:disable-next-line RETURN_ZERO,INTEGER_DECREMENT,INTEGER_INCREMENT reason="emptyNote returns the zero value of domain.Note and NotANote callers ignore FileStat Size."
		return emptyNote(), domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}, edit.NotANote, nil
	case err != nil:
		return domain.Note{}, domain.FileStat{}, "", fmt.Errorf("read %s: %w", path, err)
	}

	return parseNoteBytes(ref, raw, path)
}

func parseNoteBytes(ref domain.FileStat, raw []byte, path string) (domain.Note, domain.FileStat, edit.ReadOutcome, error) {
	// The values of a card are sent in string fields. A browser replaces an
	// ill-formed UTF-8 sequence with U+FFFD, and the next save would write those
	// characters over the original bytes.
	if !utf8.Valid(raw) {
		// gomutants:disable-next-line RETURN_ZERO reason="emptyNote returns the zero value of domain.Note."
		return emptyNote(), ref, edit.NotText, nil
	}
	if _, err := markdown.Open(raw); errors.Is(err, markdown.ErrUnreadable) {
		// gomutants:disable-next-line RETURN_ZERO reason="emptyNote returns the zero value of domain.Note."
		return emptyNote(), ref, edit.Unreadable, nil
		// gomutants:disable-next-line BRANCH_IF reason="markdown.Open returns only nil or markdown.ErrUnreadable."
	} else if err != nil {
		return domain.Note{}, domain.FileStat{}, "", fmt.Errorf("read %s: %w", path, err)
	}
	return markdown.Parse(ref, raw), ref, edit.Ok, nil
}

func emptyNote() domain.Note {
	// gomutants:disable-next-line RETURN_ZERO reason="emptyNote constructs the explicit zero value of domain.Note."
	return domain.Note{
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="emptyNote FileStat is unused by callers."
		FileStat:       domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
		Title:          "",
		Type:           "",
		ID:             "",
		Frontmatter:    nil,
		Headings:       nil,
		Links:          nil,
		Body:           "",
		Problems:       nil,
		FrontmatterErr: "",
	}
}
