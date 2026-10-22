package cards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/note"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
)

// MaxBytes is the most a deck may be and still be read here.
//
// A deck is a file holding what would otherwise be a folder of notes, so it is
// bounded several times a note's. The size is asked of the file before it is
const MaxBytes = 8 << 20

// DeckContents is one deck as a read hands it over.
type DeckContents struct {
	Path string
	// Outcome is how the read ended, out of the list an ordinary note's read
	// answers with.
	Outcome note.ReadOutcome
	// Type is what the note at the path says it is, so a caller that asked for
	// a deck and was handed a stencil is told so.
	Type domain.NoteType
	// Body is what the file says. It holds no cards for any outcome but Ok, and
	// a deck over the bound carries the problem that says why.
	Body format.Deck
	// Raw is the prose of the file currently, which is what a caller changing
	// one card splices. It is empty for any outcome but Ok.
	Raw string
	// Stencils is where the wikilink under each card's heading lands, keyed by
	Stencils map[string]string
	// Fingerprint is what the file was when it was asked about, which is before
	// its bytes were read.
	Fingerprint domain.Fingerprint
}

// StencilContents is one stencil as a read hands it over.
type StencilContents struct {
	Path    string
	Title   string
	Outcome note.ReadOutcome
	Type    domain.NoteType
	// Body is what the file says: the fields a card is asked for and the faces
	// it is shown through. It says nothing for any outcome but Ok.
	Body        format.Stencil
	Fingerprint domain.Fingerprint
}

// Read hands over a deck or a stencil, read out of the vault.
type Read struct {
	readers ports.VaultReaders
	// links answers where the wikilink a card names its stencil by lands. A
	// stencil holds no cards, so reading one asks nothing of it.
	links ports.LinkQueries
}

// NewRead is what a deck or a stencil is read through: the vault its file is
// read out of, and where the wikilink each card names its stencil by lands.
func NewRead(readers ports.VaultReaders, links ports.LinkQueries) Read {
	return Read{readers: readers, links: links}
}

// Deck reads the deck at path.
//
// An error is the vault being out of reach. What is wrong with the file itself
// is an outcome, and the caller is told which one.
func (u Read) Deck(ctx context.Context, v domain.Vault, path string) (DeckContents, error) {
	out := DeckContents{Path: path}
	n, ref, outcome, err := u.readNote(ctx, v, path, MaxBytes)
	if err != nil {
		return DeckContents{}, err
	}
	out.Fingerprint, out.Outcome, out.Type = ref, outcome, n.Type
	// A file that will not read is carried out in the outcome above, and adds
	// nothing to a body nobody read.
	//exhaustive:ignore
	switch outcome {
	case note.Ok:
		out.Body, out.Raw = format.ReadDeck(n), n.Body
		out.Stencils, err = getStencilPaths(ctx, u.links, v.ID, path, out.Body)
		if err != nil {
			return DeckContents{}, err
		}
		// A card is laid out by the stencil its wikilink lands on, so a name
		// reaching a note that is not one is a card no face shows.
		_, ordinary, err := u.readStencils(ctx, v, out.Stencils)
		if err != nil {
			return DeckContents{}, err
		}
		out.Body.Problems = append(out.Body.Problems, getNonStencilProblems(out.Body, ordinary)...)
	case note.TooLarge:
		out.Body = format.Deck{Ref: ref, Problems: []format.Problem{format.OnFile(
			format.FaultTooLarge,
			fmt.Sprintf("this deck is %d bytes, and %d is the most one is read at", ref.Size, MaxBytes),
		)}}
	}
	return out, nil
}

// GetCardStencils is the stencil each card of a deck is cut by, keyed by what
// that is not a stencil, carries a stencil of nothing.
//
// The two files are read against each other here, which is what says which of a
// card's fields is first.
func (u Read) GetCardStencils(
	ctx context.Context, v domain.Vault, path string, d format.Deck,
) (map[string]format.Stencil, error) {
	pos, err := getStencilPaths(ctx, u.links, v.ID, path, d)
	if err != nil {
		return nil, err
	}
	delta, _, err := u.readStencils(ctx, v, pos)
	return delta, err
}

// the brackets. The link is written in the deck, so it resolves against the
// deck's own folder the way every name in that file does.
func getStencilPaths(
	ctx context.Context, links ports.LinkQueries, vaultID domain.VaultID, path string, d format.Deck,
) (map[string]string, error) {
	if links == nil {
		return nil, nil
	}
	content := make([]string, 0, len(d.Cards))
	for _, card := range d.Cards {
		if card.StencilLink != "" {
			content = append(content, card.StencilLink)
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

// readStencils reads the stencil the cards of a deck are cut by, keyed by what stands
// in the brackets. Each file is opened once however many cards name it.
//
// A name landing on a note that is not a stencil is left out of the stencils
func (u Read) readStencils(
	ctx context.Context, v domain.Vault, pos map[string]string,
) (stencils map[string]format.Stencil, ordinary map[string]bool, err error) {
	if len(pos) == 0 {
		return nil, nil, nil
	}
	read := make(map[string]format.Stencil, len(pos))
	isLoose := make(map[string]bool, len(pos))
	stencils = make(map[string]format.Stencil, len(pos))
	ordinary = make(map[string]bool, len(pos))
	for content, path := range pos {
		target, observed := read[path]
		if !observed {
			found, err := u.Stencil(ctx, v, path)
			if err != nil {
				return nil, nil, err
			}
			if found.Type == format.TypeStencil {
				target = found.Body
			}
			read[path] = target
			isLoose[path] = found.Outcome == note.Ok && found.Type != format.TypeStencil
		}
		stencils[content] = target
		ordinary[content] = isLoose[path]
	}
	return stencils, ordinary, nil
}

// getNonStencilProblems returns one problem per card whose wikilink reaches a note that is not
// a stencil. The values are read, and no face lays the card out.
func getNonStencilProblems(d format.Deck, ordinary map[string]bool) []format.Problem {
	var out []format.Problem
	for item, card := range d.Cards {
		if !ordinary[card.StencilLink] {
			continue
		}
		out = append(out, format.OnCard(item, format.FaultNotAStencil,
			card.StencilLink+" is a note and not a stencil, so this card is shown by no face"))
	}
	return out
}

// Stencil reads the stencil at path. A stencil is a note and is bounded as one.
func (u Read) Stencil(ctx context.Context, v domain.Vault, path string) (StencilContents, error) {
	out := StencilContents{Path: path}
	n, ref, outcome, err := u.readNote(ctx, v, path, note.MaxBytes)
	if err != nil {
		return StencilContents{}, err
	}
	out.Fingerprint, out.Outcome, out.Type = ref, outcome, n.Type
	out.Title = n.Title
	if outcome == note.Ok {
		out.Body = format.ReadStencil(n)
	}
	return out, nil
}

// readNote is the file at a path: what the vault says is there, and the note
// its bytes parse to once everything that would refuse them has been asked.
func (u Read) readNote(
	ctx context.Context, v domain.Vault, path string, maxSize int64,
) (domain.Note, domain.Fingerprint, note.ReadOutcome, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.Note{}, domain.Fingerprint{}, "", err
	}

	// The vault says what is at a path without opening it: a note, a file it
	// leaves alone, or nothing. The size is among what it says, which is what
	// keeps a deck over the bound unread.
	ref, err := reader.GetFingerprint(ctx, path)
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
		return domain.Note{}, domain.Fingerprint{}, note.NotANote, nil
	case !errors.Is(err, fs.ErrNotExist):
		return domain.Note{}, domain.Fingerprint{}, "", fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return domain.Note{}, ref, note.Missing, nil
	case !target && err == nil:
		return domain.Note{}, domain.Fingerprint{}, note.NotANote, nil
	case err != nil:
		return domain.Note{}, domain.Fingerprint{}, "", fmt.Errorf("read %s: %w", path, err)
	}

	// The values of a card go out in string fields. A browser turns an
	// ill-formed sequence into U+FFFD, and the next save puts those characters
	// where the person's bytes were.
	if !utf8.Valid(raw) {
		return domain.Note{}, ref, note.NotText, nil
	}
	if _, err := markdown.Open(raw); errors.Is(err, markdown.ErrUnreadable) {
		return domain.Note{}, ref, note.Unreadable, nil
	} else if err != nil {
		return domain.Note{}, domain.Fingerprint{}, "", fmt.Errorf("read %s: %w", path, err)
	}
	return markdown.Parse(ref, raw), ref, note.Ok, nil
}
