package reviewui

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// ErrNoSuchCard is a card the deck named does not hold.
var ErrNoSuchCard = errors.New("the deck holds no card of that mark")

// ReadCard is one card's values, in the order its stencil asks for them.
//
// Every field the stencil declares stands in the answer, the ones the card
// leaves out among them: a person putting a card right is shown every field
// there is to fill.
func (a *API) ReadCard(
	ctx context.Context, r *connect.Request[v1.ReadCardRequest],
) (*connect.Response[v1.ReadCardResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	deck, card, stencil, err := a.card(ctx, v, r.Msg.GetDeck(), r.Msg.GetCard())
	if err != nil {
		return connect.NewResponse(&v1.ReadCardResponse{Refused: err.Error()}), nil
	}
	return connect.NewResponse(&v1.ReadCardResponse{
		Values:  filled(stencil, card),
		At:      fingerprintOf(deck.Ref),
		Stencil: card.Stencil,
	}), nil
}

// fingerprintOf is what a file stood at when it was read, as the wire carries
// it. A read that saw nothing carries nothing, and a write presenting nothing
// lands on whatever the file now holds.
func fingerprintOf(ref domain.FileRef) *v1.Fingerprint {
	if ref == (domain.FileRef{}) {
		return nil
	}
	return &v1.Fingerprint{Path: ref.Path, Size: ref.Size, Mtime: ref.MTime}
}

func refOf(at *v1.Fingerprint) domain.FileRef {
	if at == nil {
		return domain.FileRef{}
	}
	return domain.FileRef{Path: at.GetPath(), Size: at.GetSize(), MTime: at.GetMtime()}
}

// WriteCard puts those values back into the deck.
func (a *API) WriteCard(
	ctx context.Context, r *connect.Request[v1.WriteCardRequest],
) (*connect.Response[v1.WriteCardResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	deck, err := a.Read.Deck(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	at, held := held(deck.Deck, r.Msg.GetCard())
	if !held {
		return connect.NewResponse(&v1.WriteCardResponse{
			Refused: fmt.Sprintf("%v: %s", ErrNoSuchCard, r.Msg.GetCard()),
		}), nil
	}

	// The values arrive whole, so what the card holds is what was typed. A
	// field the person emptied is a field the card writes empty, which is what
	// the format calls an empty value.
	cards := append([]format.Card(nil), deck.Deck.Cards...)
	values := make([]format.Value, 0, len(r.Msg.GetValues()))
	for _, one := range r.Msg.GetValues() {
		values = append(values, format.Value{Field: one.GetField(), Text: one.GetText()})
	}
	cards[at].Values = values
	whole := deck.Deck
	whole.Cards = cards

	body, err := format.DeckBody(whole)
	if err != nil {
		return connect.NewResponse(&v1.WriteCardResponse{Refused: err.Error()}), nil
	}

	written, err := a.Write.Deck(ctx, v, r.Msg.GetDeck(), body, refOf(r.Msg.GetAt()))
	if errors.Is(err, port.ErrChanged) {
		return connect.NewResponse(&v1.WriteCardResponse{Changed: true}), nil
	}
	if err != nil {
		return connect.NewResponse(&v1.WriteCardResponse{Refused: err.Error()}), nil
	}

	// What the card now lays out is read from the file again, so the person is
	// shown what stands there and not what was sent.
	out := &v1.WriteCardResponse{At: fingerprintOf(written.At)}
	if _, card, stencil, err := a.card(ctx, v, r.Msg.GetDeck(), r.Msg.GetCard()); err == nil {
		out.Heading = card.Heading
		for _, face := range stencil.Faces {
			if face.Name != r.Msg.GetFace() {
				continue
			}
			out.Front, out.Back = format.Lay(stencil, face, card)
		}
	}
	return connect.NewResponse(out), nil
}

// card is one card of a deck, and the stencil that cuts it.
func (a *API) card(
	ctx context.Context, v domain.Vault, path, mark string,
) (cards.Deck, format.Card, format.Stencil, error) {
	deck, err := a.Read.Deck(ctx, v, path)
	if err != nil {
		return cards.Deck{}, format.Card{}, format.Stencil{}, err
	}
	at, held := held(deck.Deck, mark)
	if !held {
		return cards.Deck{}, format.Card{}, format.Stencil{},
			fmt.Errorf("%w: %s", ErrNoSuchCard, mark)
	}
	card := deck.Deck.Cards[at]

	stencil := format.Stencil{}
	if to, named := deck.Stencils[card.Stencil]; named {
		if one, err := a.Read.Stencil(ctx, v, to); err == nil {
			stencil = one.Stencil
		}
	}
	return deck, card, stencil, nil
}

// held is where a card of that mark stands among a deck's.
func held(d format.Deck, mark string) (int, bool) {
	for at, one := range d.Cards {
		if one.Mark == mark {
			return at, true
		}
	}
	return 0, false
}

// filled is every field the stencil declares, with what the card holds under
// it. A stencil that could not be read leaves the card's own values, in the
// order the file wrote them.
func filled(s format.Stencil, c format.Card) []*v1.CardValue {
	if len(s.Fields) == 0 {
		out := make([]*v1.CardValue, 0, len(c.Values))
		for _, one := range c.Values {
			out = append(out, &v1.CardValue{Field: one.Field, Text: one.Text})
		}
		return out
	}
	out := make([]*v1.CardValue, 0, len(s.Fields))
	for _, field := range s.Fields {
		text, _ := c.Value(field)
		out = append(out, &v1.CardValue{Field: field, Text: text})
	}
	return out
}
