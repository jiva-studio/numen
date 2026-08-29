package flashcardsui

import (
	"context"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Reviewed is how much of a vault was answered on each day it was reviewed.
//
// The days come back in order, oldest first, because what draws them draws them
// along a line of time.
func (a *API) Reviewed(
	ctx context.Context, r *connect.Request[v1.ReviewedRequest],
) (*connect.Response[v1.ReviewedResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	said, err := a.Counted.Execute(ctx, v)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.ReviewedResponse{
		Days:     inOrder(said.Days),
		Due:      inOrder(said.Due),
		Streak:   int32(said.Streak),
		Answered: int32(said.Answered),
	}), nil
}

// inOrder is what each day came to, oldest first.
func inOrder(days map[string]int) []*v1.Reviewing {
	out := make([]*v1.Reviewing, 0, len(days))
	for day, answered := range days {
		out = append(out, &v1.Reviewing{Day: day, Answered: int32(answered)})
	}
	slices.SortFunc(out, func(one, other *v1.Reviewing) int {
		return cmpDay(one.GetDay(), other.GetDay())
	})
	return out
}

// cmpDay orders two days. A day is written as the year, the month and the day,
// each padded, so it sorts as text the way it sorts in time.
func cmpDay(one, other string) int {
	switch {
	case one < other:
		return -1
	case one > other:
		return 1
	}
	return 0
}
