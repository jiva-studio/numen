package cards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// MaxBytes is the largest deck that is read.
//
// A deck is a single file holding a replacement for a folder of notes, so
// its limit is several times the limit of a note. The size is read from the
// file metadata before the file is opened.
const MaxBytes = 8 << 20

// DeckContents is the result of reading one deck.
type DeckContents struct {
	Path string
	// Outcome is how the read ended, out of the list an ordinary note's read
	// answers with.
	Outcome note.ReadOutcome
	// Type is the note type recorded in the note at the path, so a caller that
	// requested a deck and received a card model can detect the mismatch.
	Type domain.NoteType
	// Body is the parsed deck. It contains no cards for any outcome other than
	// Ok, and a deck over the size limit carries the problem that states why.
	Body format.Deck
	// Raw is the prose of the file currently, which is what a caller changing
	// one card splices. It is empty for any outcome but Ok.
	Raw string
	// CardModels maps the wikilink name under each card's heading to the path of the
	// note it resolves to.
	CardModels map[string]string
	// FileStat is the file metadata, read before the file content.
	FileStat domain.FileStat
}

// CardModelContents is the result of reading one card model.
type CardModelContents struct {
	Path    string
	Title   string
	Outcome note.ReadOutcome
	Type    domain.NoteType
	// Body is the parsed card model: the fields of a card and the templates it is
	// rendered through. It is empty for any outcome other than Ok.
	Body     format.CardModel
	FileStat domain.FileStat
}

// Read reads a deck or a card model from the vault.
type Read struct {
	readers ports.VaultReaders
	// links resolves the wikilink by which a card names its card model. Reading a
	// card model does not use it, because a card model holds no cards.
	links ports.LinkQueries
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
func (u Read) Deck(ctx context.Context, v domain.Vault, path string) (DeckContents, error) {
	out := DeckContents{Path: path}
	n, ref, outcome, err := u.readNote(ctx, v, path, MaxBytes)
	if err != nil {
		return DeckContents{}, err
	}
	out.FileStat, out.Outcome, out.Type = ref, outcome, n.Type
	// An outcome other than Ok and TooLarge is reported in the outcome above and
	// leaves the body empty.
	//exhaustive:ignore
	switch outcome {
	case note.Ok:
		out.Body, out.Raw = format.ReadDeck(n), n.Body
		out.CardModels, err = getCardModelPaths(ctx, u.links, v.ID, path, out.Body)
		if err != nil {
			return DeckContents{}, err
		}
		// A card is laid out by the card model its wikilink lands on, so a name
		// reaching a note that is not one is a card no template shows.
		_, ordinary, err := u.readCardModels(ctx, v, out.CardModels)
		if err != nil {
			return DeckContents{}, err
		}
		out.Body.Problems = append(out.Body.Problems, getNonCardModelProblems(out.Body, ordinary)...)
	case note.TooLarge:
		out.Body = format.Deck{Ref: ref, Problems: []format.Problem{format.OnFile(
			format.FaultTooLarge,
			fmt.Sprintf("this deck is %d bytes, and %d is the most one is read at", ref.Size, MaxBytes),
		)}}
	}
	return out, nil
}

// GetCardCardModels returns the card model each card of a deck is cut by, keyed
// by the wikilink name written under the card heading. A name that resolves to
// a note that is not a card model maps to an empty card model.
//
// The deck and the card models are read together, which determines the first
// field of each card.
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

// getCardModelPaths maps the wikilink name under each card's heading to the
// path of the note it resolves to. The link is written in the deck, so it
// resolves relative to the deck's folder like every name in that file.
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

// readCardModels reads the card models that the cards of a deck are cut by,
// keyed by the wikilink name. Each file is opened once however many cards name
// it.
//
// A name that resolves to a note that is not a card model is omitted from the
// card models and marked in ordinary.
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
			isLoose[path] = found.Outcome == note.Ok && found.Type != format.TypeCardModel
		}
		cardModels[content] = target
		ordinary[content] = isLoose[path]
	}
	return cardModels, ordinary, nil
}

// getNonCardModelProblems returns one problem per card whose wikilink resolves
// to a note that is not a card model. The values are read, and no template
// renders the card.
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

// CardModel reads the card model at path. A card model is a note and is bounded as one.
func (u Read) CardModel(ctx context.Context, v domain.Vault, path string) (CardModelContents, error) {
	out := CardModelContents{Path: path}
	n, ref, outcome, err := u.readNote(ctx, v, path, note.MaxBytes)
	if err != nil {
		return CardModelContents{}, err
	}
	out.FileStat, out.Outcome, out.Type = ref, outcome, n.Type
	out.Title = n.Title
	if outcome == note.Ok {
		out.Body = format.ReadCardModel(n)
	}
	return out, nil
}

// readNote reads the file at a path: the file metadata, and the note parsed
// from its bytes when every precondition on the file holds.
func (u Read) readNote(
	ctx context.Context, v domain.Vault, path string, maxSize int64,
) (domain.Note, domain.FileStat, note.ReadOutcome, error) {
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
			return domain.Note{}, ref, note.NotANote, nil
		}
		if ref.Size > maxSize {
			return domain.Note{}, ref, note.TooLarge, nil
		}
	case errors.Is(err, ports.ErrNotANote):
		return domain.Note{}, domain.FileStat{}, note.NotANote, nil
	case !errors.Is(err, fs.ErrNotExist):
		return domain.Note{}, domain.FileStat{}, "", fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return domain.Note{}, ref, note.Missing, nil
	case !target && err == nil:
		return domain.Note{}, domain.FileStat{}, note.NotANote, nil
	case err != nil:
		return domain.Note{}, domain.FileStat{}, "", fmt.Errorf("read %s: %w", path, err)
	}

	// The values of a card are sent in string fields. A browser replaces an
	// ill-formed UTF-8 sequence with U+FFFD, and the next save would write those
	// characters over the original bytes.
	if !utf8.Valid(raw) {
		return domain.Note{}, ref, note.NotText, nil
	}
	if _, err := markdown.Open(raw); errors.Is(err, markdown.ErrUnreadable) {
		return domain.Note{}, ref, note.Unreadable, nil
	} else if err != nil {
		return domain.Note{}, domain.FileStat{}, "", fmt.Errorf("read %s: %w", path, err)
	}
	return markdown.Parse(ref, raw), ref, note.Ok, nil
}
