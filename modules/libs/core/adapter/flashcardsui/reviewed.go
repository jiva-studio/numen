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

	out := &v1.ReviewedResponse{
		Days:     make([]*v1.Reviewing, 0, len(said.Days)),
		Streak:   int32(said.Streak),
		Answered: int32(said.Answered),
	}
	for day, answered := range said.Days {
		out.Days = append(out.Days, &v1.Reviewing{Day: day, Answered: int32(answered)})
	}
	slices.SortFunc(out.Days, func(one, other *v1.Reviewing) int {
		return cmpDay(one.GetDay(), other.GetDay())
	})
	return connect.NewResponse(out), nil
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
