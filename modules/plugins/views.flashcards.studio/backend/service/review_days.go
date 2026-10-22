package service

import (
	"cmp"
	"context"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// ListReviewDays is how much of a vault was answered on each day it was
// reviewed.
//
// The days come back in order, oldest first, because what draws them draws them
// along a line of time. A caller naming a stretch is answered that stretch; the
// streak and the count are over the whole of the vault either way.
func (a *reviewHandler) ListReviewDays(
	ctx context.Context, r *connect.Request[v1.ListReviewDaysRequest],
) (*connect.Response[v1.ListReviewDaysResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	response, err := a.scenarios().flash.GetReviewDays.Execute(ctx, v, r.Msg.GetFrom(), r.Msg.GetTo())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	days := make([]*v1.ReviewDay, 0, len(response.Days))
	for day, one := range response.Days {
		came := response.Retained[day]
		days = append(days, &v1.ReviewDay{
			Day:      day,
			Answered: int32(one.Answered),
			Again:    int32(one.Again),
			Hard:     int32(one.Hard),
			Good:     int32(one.Good),
			Easy:     int32(one.Easy),
			Asked:    int32(came.Asked),
			Recalled: int32(came.Recalled),
		})
	}
	due := make([]*v1.ReviewDay, 0, len(response.Due))
	for day, falls := range response.Due {
		due = append(due, &v1.ReviewDay{Day: day, Answered: int32(falls)})
	}

	return connect.NewResponse(&v1.ListReviewDaysResponse{
		Days:     sortReviewDays(days),
		Due:      sortReviewDays(due),
		Streak:   int32(response.Streak),
		Answered: int32(response.Answered),
	}), nil
}

// sortReviewDays puts the days oldest first.
func sortReviewDays(days []*v1.ReviewDay) []*v1.ReviewDay {
	slices.SortFunc(days, func(one, other *v1.ReviewDay) int {
		return cmp.Compare(one.GetDay(), other.GetDay())
	})
	return days
}
