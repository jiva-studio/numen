package review

import (
	"context"
	"time"

	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Standing is one seat as the vault holds it: a card, shown through one face of
// the stencil that cuts it, and where both of them are written.
type Standing struct {
	// Deck is the path of the file the card stands in, and Section the name of
	// the section it stands under. A card standing before the first section has
	// none.
	Deck    string
	Section string
	// Seat is what a schedule belongs to: the card's mark and the face's name.
	Seat history.Seat
	// Heading is what the card's heading shows, which is the first line of its
	// first field.
	Heading string

	// What the seat is laid out from. It is read once with the deck and kept,
	// so laying out is the last thing done and only for a card about to be
	// shown: counting what a vault owes fills in no template at all.
	stencil format.Stencil
	face    format.Face
	card    format.Card
}

// Lay is the face filled with this card's values: what stands before the answer
// and what stands after it, as markdown.
func (s Standing) Lay() (front, back string) { return format.Lay(s.stencil, s.face, s.card) }

// Seats is every seat a vault holds, read out of its files.
//
// A deck holding a card that carries no mark is written, which mints one for
// every card in it, and read again. The seats come from that second read, so a
// card is only ever answered under a mark the file holds.
type Seats struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Notes   port.NoteQueries
	Links   port.LinkQueries
	// Index brings what a write touched up to date before the deck is read
	// again. A build holding none writes the file and reads it back off disk.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
	Now   func() time.Time
}

// Execute reads every deck the vault holds and says what stands in it.
//
// A deck that cannot be read contributes no seat and is not an error: one
// unreadable file is not a reason to refuse a person the rest of their cards.
// What was wrong with it is the deck's own problem, and the editor is where it
// is settled.
func (u Seats) Execute(ctx context.Context, v domain.Vault) ([]Standing, error) {
	paths, err := u.Notes.OfType(ctx, v.ID, domain.TypeDeck)
	if err != nil {
		return nil, err
	}

	read := cards.Read{Readers: u.Readers, Links: u.Links}
	stencils := make(map[string]format.Stencil)
	var out []Standing
	for _, path := range paths {
		deck, err := u.deck(ctx, v, read, path)
		if err != nil {
			return nil, err
		}
		if deck.Outcome != note.Ok {
			continue
		}
		seats, err := u.standing(ctx, v, read, deck, stencils)
		if err != nil {
			return nil, err
		}
		out = append(out, seats...)
	}
	return out, nil
}

// deck reads one deck, and writes it first where a card in it carries no mark.
func (u Seats) deck(
	ctx context.Context, v domain.Vault, read cards.Read, path string,
) (cards.Deck, error) {
	deck, err := read.Deck(ctx, v, path)
	if err != nil {
		return cards.Deck{}, err
	}
	if deck.Outcome != note.Ok || !unmarked(deck.Deck) {
		return deck, nil
	}

	// The body goes back exactly as it was read. What the write is for is the
	// deck being made whole on the way past, which is where a mark is minted.
	write := cards.Write{
		Readers: u.Readers, Writers: u.Writers, Links: u.Links, Index: u.Index, Now: u.Now,
	}
	body, err := format.DeckBody(deck.Deck)
	if err != nil {
		return deck, nil
	}
	// A write that did not land is not reported: the editor may be saving the
	// same deck, and the vault's write lock lives in one process and does not
	// reach across two. What the deck is read as afterwards is what the file
	// holds, so a stamp that did not land leaves those cards out of this
	// reading and the next one mints them again.
	_, _ = write.Deck(ctx, v, path, body, deck.Ref)
	return read.Deck(ctx, v, path)
}

// unmarked reports whether any card of a deck carries no mark. A card typed by
// hand carries none until the application writes the file.
func unmarked(d format.Deck) bool {
	for _, c := range d.Cards {
		if c.Mark == "" {
			return true
		}
	}
	return false
}

// standing is the seats one deck holds: every card of a mark, through every
// face of the stencil it names that lays anything out.
func (u Seats) standing(
	ctx context.Context, v domain.Vault, read cards.Read,
	deck cards.Deck, stencils map[string]format.Stencil,
) ([]Standing, error) {
	var out []Standing
	for _, card := range deck.Deck.Cards {
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
			if err != nil {
				return nil, err
			}
			if one.Outcome != note.Ok || one.Type != domain.TypeStencil {
				// A card whose stencil is not one is a card no face shows. It
				// is a problem against the deck, and it is settled in a window.
				stencils[path] = format.Stencil{}
				continue
			}
			stencil = one.Stencil
			stencils[path] = stencil
		}

		for _, face := range stencil.Faces {
			// A face missing a side lays out nothing, which is a problem
			// against the stencil and no seat here.
			if face.Front == "" || face.Back == "" {
				continue
			}
			out = append(out, Standing{
				Deck:    deck.Path,
				Section: section(deck.Deck, card),
				Seat:    history.Seat{Card: card.Mark, Face: face.Name},
				Heading: card.Heading,
				stencil: stencil,
				face:    face,
				card:    card,
			})
		}
	}
	return out, nil
}

// section is the name of the section a card stands under, and is empty for a
// card standing before the first.
func section(d format.Deck, c format.Card) string {
	if c.Section == format.NoSection || c.Section >= len(d.Sections) {
		return ""
	}
	return d.Sections[c.Section].Name
}
