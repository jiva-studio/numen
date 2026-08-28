package cards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"unicode/utf8"

	format "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// MaxBytes is the most a deck may be and still be read here.
//
// A deck is a file holding what would otherwise be a folder of notes, so it is
// bounded several times a note's. The size is asked of the file before it is
// opened, so a deck over the bound is refused with none of its bytes read.
const MaxBytes = 8 << 20

// Deck is one deck as a read hands it over.
type Deck struct {
	Path string
	// Outcome is how the read ended, out of the list an ordinary note's read
	// answers with.
	Outcome note.Outcome
	// Type is what the note at the path says it is, so a caller that asked for
	// a deck and was handed a stencil is told so.
	Type domain.NoteType
	// Deck is what the file says. It holds no cards for any outcome but Ok, and
	// a deck over the bound carries the problem that says why.
	Deck format.Deck
	// Stencils is where the wikilink under each card's heading lands, keyed by
	// what stands in the brackets. A name that reaches no note is absent.
	Stencils map[string]string
	// Ref is what the file was when it was asked about, which is before its
	// bytes were read.
	Ref domain.FileRef
}

// Stencil is one stencil as a read hands it over.
type Stencil struct {
	Path    string
	Outcome note.Outcome
	Type    domain.NoteType
	Stencil format.Stencil
	Ref     domain.FileRef
}

// Read hands over a deck or a stencil, read out of the vault.
//
// A file is asked about before it is opened, so what is refused for its size is
// refused without being read.
type Read struct {
	Readers port.VaultReaders
	// Links answers where the wikilink a card names its stencil by lands. A
	// build holding none reads the cards and says where no stencil is filed.
	Links port.LinkQueries
}

// Deck reads the deck at path.
//
// An error is the vault being out of reach. What is wrong with the file itself
// is an outcome, and the caller is told which one.
func (u Read) Deck(ctx context.Context, v domain.Vault, path string) (Deck, error) {
	out := Deck{Path: path}
	n, ref, outcome, err := u.looked(ctx, v, path, MaxBytes)
	if err != nil {
		return Deck{}, err
	}
	out.Ref, out.Outcome, out.Type = ref, outcome, n.Type
	switch outcome {
	case note.Ok:
		out.Deck = format.ReadDeck(n)
		out.Stencils, err = cutting(ctx, u.Links, v.ID, path, out.Deck)
		if err != nil {
			return Deck{}, err
		}
		// A heading is the first field's value, and which field that is stands
		// in the stencil, so the two files are read against each other here.
		by, ordinary, err := u.stencils(ctx, v, out.Stencils)
		if err != nil {
			return Deck{}, err
		}
		out.Deck.Problems = append(out.Deck.Problems, format.Cut(out.Deck, by)...)
		out.Deck.Problems = append(out.Deck.Problems, notStencils(out.Deck, ordinary)...)
	case note.TooLarge:
		out.Deck = format.Deck{Ref: ref, Problems: []format.Problem{format.OnFile(
			format.CheckTooLarge,
			fmt.Sprintf("this deck is %d bytes, and %d is the most one is read at", ref.Size, MaxBytes),
		)}}
	}
	return out, nil
}

// cutting is where each card's wikilink lands, keyed by what stands in the
// brackets. The link is written in the deck, so it resolves against the deck's
// own folder the way every name in that file does.
func cutting(
	ctx context.Context, links port.LinkQueries, vaultID, path string, d format.Deck,
) (map[string]string, error) {
	if links == nil {
		return nil, nil
	}
	written := make([]string, 0, len(d.Cards))
	for _, card := range d.Cards {
		if card.Stencil != "" {
			written = append(written, card.Stencil)
		}
	}
	if len(written) == 0 {
		return nil, nil
	}
	return links.Resolve(ctx, vaultID, path, written)
}

// stencils is the stencil the cards of a deck are cut by, keyed by what stands
// in the brackets. Each file is opened once however many cards name it.
//
// A name landing on a note that is not a stencil is left out of the stencils
// and stands in ordinary, so the cards written under it are marked.
func (u Read) stencils(
	ctx context.Context, v domain.Vault, at map[string]string,
) (by map[string]format.Stencil, ordinary map[string]bool, err error) {
	if len(at) == 0 {
		return nil, nil, nil
	}
	read := make(map[string]format.Stencil, len(at))
	loose := make(map[string]bool, len(at))
	by = make(map[string]format.Stencil, len(at))
	ordinary = make(map[string]bool, len(at))
	for written, path := range at {
		held, seen := read[path]
		if !seen {
			found, err := u.Stencil(ctx, v, path)
			if err != nil {
				return nil, nil, err
			}
			if found.Outcome == note.Ok && found.Type == domain.TypeStencil {
				held = found.Stencil
			}
			read[path] = held
			loose[path] = found.Outcome == note.Ok && found.Type != domain.TypeStencil
		}
		by[written] = held
		ordinary[written] = loose[path]
	}
	return by, ordinary, nil
}

// notStencils is one problem per card whose wikilink reaches a note that is not
// a stencil. The values are read, and no face lays the card out.
func notStencils(d format.Deck, ordinary map[string]bool) []format.Problem {
	if len(ordinary) == 0 {
		return nil
	}
	var out []format.Problem
	for at, card := range d.Cards {
		if !ordinary[card.Stencil] {
			continue
		}
		out = append(out, format.OnCard(at, format.CheckNotAStencil,
			card.Stencil+" is a note and not a stencil, so this card is shown by no face"))
	}
	return out
}

// Stencil reads the stencil at path. A stencil is a note and is bounded as one.
func (u Read) Stencil(ctx context.Context, v domain.Vault, path string) (Stencil, error) {
	out := Stencil{Path: path}
	n, ref, outcome, err := u.looked(ctx, v, path, note.MaxBytes)
	if err != nil {
		return Stencil{}, err
	}
	out.Ref, out.Outcome, out.Type = ref, outcome, n.Type
	if outcome == note.Ok {
		out.Stencil = format.ReadStencil(n)
	}
	return out, nil
}

// looked is the file at a path: what the vault says is there, and the note its
// bytes parse to once everything that would refuse them has been asked.
func (u Read) looked(
	ctx context.Context, v domain.Vault, path string, bound int64,
) (domain.Note, domain.FileRef, note.Outcome, error) {
	reader, err := u.Readers.Open(v)
	if err != nil {
		return domain.Note{}, domain.FileRef{}, "", err
	}

	// The vault says what is at a path without opening it: a note, a file it
	// leaves alone, or nothing. The size is among what it says, which is what
	// keeps a deck over the bound unread.
	ref, err := reader.Stat(ctx, path)
	held := err == nil
	switch {
	case held:
		if ref.Kind != domain.KindNote {
			return domain.Note{}, ref, note.NotANote, nil
		}
		if ref.Size > bound {
			return domain.Note{}, ref, note.TooLarge, nil
		}
	case errors.Is(err, port.ErrNotANote):
		return domain.Note{}, domain.FileRef{}, note.NotANote, nil
	case !errors.Is(err, fs.ErrNotExist):
		return domain.Note{}, domain.FileRef{}, "", fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return domain.Note{}, ref, note.Missing, nil
	case !held && err == nil:
		return domain.Note{}, ref, note.NotANote, nil
	case err != nil:
		return domain.Note{}, ref, "", fmt.Errorf("read %s: %w", path, err)
	}

	// The values of a card go out in string fields. A browser turns an
	// ill-formed sequence into U+FFFD, and the next save puts those characters
	// where the person's bytes were.
	if !utf8.Valid(raw) {
		return domain.Note{}, ref, note.NotText, nil
	}
	if _, err := markdown.Open(raw); err != nil {
		return domain.Note{}, ref, note.Unreadable, nil
	}
	return markdown.Parse(ref, raw), ref, note.Ok, nil
}
