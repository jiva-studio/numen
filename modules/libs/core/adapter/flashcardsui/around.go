package flashcardsui

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/refusal"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Around hands over what the deck being sat to is joined to.
//
// Nothing here writes and no run is named: a person may read around a deck
// before they have started on it, and reading is not part of a sitting.
func (a *API) Around(
	ctx context.Context, r *connect.Request[v1.AroundRequest],
) (*connect.Response[v1.AroundResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	joined, err := a.Joined.Execute(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.AroundResponse{
		Notes:  make([]*v1.Neighbour, 0, len(joined.Notes)),
		Unread: int32(joined.Unread),
	}
	for _, one := range joined.Notes {
		next := &v1.Neighbour{
			Written:   one.Written,
			Path:      one.Path,
			Title:     one.Title,
			Body:      one.Body,
			Label:     one.Label,
			Points:    one.Points,
			Ambiguous: one.Ambiguous,
		}
		if reason, refused := refusal.Of(one.Outcome); refused {
			next.Refusal = &reason
		}
		out.Notes = append(out.Notes, next)
	}
	return connect.NewResponse(out), nil
}
