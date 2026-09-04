package flashcards

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// CardFace is one card face as the vault holds it: a card, shown through one
// face of the stencil that cuts it, and where both of them are written.
type CardFace struct {
	// Deck is the path of the file the card stands in, and Section the name of
	// the section it stands under. A card standing before the first section has
	// none.
	Deck    string
	Section string
	// ID is what a schedule belongs to: the card's mark and the face's name.
	ID review.CardFaceID
	// Heading is what the card's heading shows, which is the first line of its
	// first field.
	Heading string

	// What it is laid out from. It is read once with the deck and kept,
	// so laying out is the last thing done and only for a card about to be
	// shown: counting what a vault owes fills in no template at all.
	stencil format.Stencil
	face    format.FaceTemplate
	card    format.Card
}

// Lay is the face filled with this card's values: what stands before the answer
// and what stands after it, as HTML.
func (s CardFace) Lay() (front, back string) { return format.Lay(s.stencil, s.face, s.card) }

// ListCardFaces is every card face a vault holds, read out of its files.
//
// Nothing here writes. A card carrying no mark cannot be answered and is left
// out; giving it one is Marking, which a person's own vault has done to it
// before they are asked anything.
type ListCardFaces struct {
	Readers port.VaultReaders
	Notes   port.NoteQueries
	Links   port.LinkQueries
}

// ErrUnread is what a vault the index does not carry gets. It is the signal to
// read that vault, and a window holding one reads it.
var ErrUnread = errors.New("the index does not carry this vault yet")

// Execute reads every deck the vault holds and says what stands in it. A vault
// the index does not carry gets ErrUnread.
func (u ListCardFaces) Execute(ctx context.Context, v domain.Vault) ([]CardFace, error) {
	paths, err := u.Decks(ctx, v)
	if err != nil {
		return nil, err
	}
	return u.Of(ctx, v, paths), nil
}

// Decks is the path of every deck the vault holds.
//
// A vault the index does not carry gets ErrUnread. The list of decks is the
// index's answer, and a vault absent from it is not a vault holding no cards.
func (u ListCardFaces) Decks(ctx context.Context, v domain.Vault) ([]string, error) {
	if u.Notes == nil {
		return nil, ErrUnread
	}
	held, err := u.Notes.Holds(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	if !held {
		return nil, ErrUnread
	}
	return u.Notes.OfType(ctx, v.ID, domain.TypeDeck)
}

// Of reads the decks named and says what stands in them. A caller after the
// cards of one preset hands it the decks pointing there, and the rest of the
// vault is left unread.
//
// A deck that cannot be read contributes no card face and is not an error. What
// was wrong with it is the deck's own problem, and the editor is where it is
// settled.
func (u ListCardFaces) Of(ctx context.Context, v domain.Vault, paths []string) []CardFace {
	read := cards.Read{Readers: u.Readers, Links: u.Links}
	stencils := make(map[string]format.Stencil)
	var out []CardFace
	for _, path := range paths {
		deck, err := read.Deck(ctx, v, path)
		if err != nil || deck.Outcome != note.Ok {
			continue
		}
		out = append(out, u.standing(ctx, v, read, deck, stencils)...)
	}
	return out
}

// standing is the card faces one deck holds: every card of a mark, through every
// face of the stencil it names that lays anything out.
func (u ListCardFaces) standing(
	ctx context.Context, v domain.Vault, read cards.Read,
	deck cards.DeckContents, stencils map[string]format.Stencil,
) []CardFace {
	var out []CardFace
	for _, card := range deck.Body.Cards {
		if card.Mark == "" {
			continue
		}
		path, held := deck.Stencils[card.Stencil]
		if !held {
			continue
		}
		stencil, held := stencils[path]
		if !held {
			one, err := read.Stencil(ctx, v, path)
			if err != nil || one.Outcome != note.Ok || one.Type != domain.TypeStencil {
				// A card whose stencil is not one is a card no face shows. It
				// is a problem against the deck, and it is settled in a window.
				stencils[path] = format.Stencil{}
				continue
			}
			stencil = one.Body
			stencils[path] = stencil
		}

		for _, face := range stencil.Faces {
			// A face missing a side lays out nothing, which is a problem
			// against the stencil and no card face here.
			if face.Front == "" || face.Back == "" {
				continue
			}
			out = append(out, CardFace{
				Deck:    deck.Path,
				Section: section(deck.Body, card),
				ID:      review.CardFaceID{Card: string(card.Mark), Face: face.Name},
				Heading: card.Heading,
				stencil: stencil,
				face:    face,
				card:    card,
			})
		}
	}
	return out
}

// section is the name of the section a card stands under, and is empty for a
// card standing before the first.
func section(d format.Deck, c format.Card) string {
	if c.Section == format.NoSection || c.Section >= len(d.Sections) {
		return ""
	}
	return d.Sections[c.Section].Name
}
