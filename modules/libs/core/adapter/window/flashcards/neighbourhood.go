package flashcards

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// GetDeckNeighbourhood hands over what the deck being sat to is joined to.
//
// Nothing here writes and no run is named: a person may read around a deck
// before they have started on it, and reading is not part of a session.
func (a *API) GetDeckNeighbourhood(
	ctx context.Context, r *connect.Request[v1.GetDeckNeighbourhoodRequest],
) (*connect.Response[v1.GetDeckNeighbourhoodResponse], error) {
	v, err := a.Vault(r.Msg.GetVault())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	joined, err := a.Neighbourhood.Execute(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.GetDeckNeighbourhoodResponse{
		Notes:  make([]*v1.DeckNeighbour, 0, len(joined.Notes)),
		Unread: int32(joined.Unread),
	}
	for _, one := range joined.Notes {
		next := &v1.DeckNeighbour{
			Written:     one.Written,
			Path:        one.Path,
			Title:       one.Title,
			Body:        one.Body,
			Label:       one.Label,
			IsPointedAt: !one.Backlink,
			IsAmbiguous: one.Ambiguous,
		}
		if reason, refused := wire.ErrorCodeOf(one.Outcome); refused {
			next.Error = &reason
		}
		out.Notes = append(out.Notes, next)
	}
	return connect.NewResponse(out), nil
}
