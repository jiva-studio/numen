package flashcards

import (
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
func (a *API) ListReviewDays(
	ctx context.Context, r *connect.Request[v1.ListReviewDaysRequest],
) (*connect.Response[v1.ListReviewDaysResponse], error) {
	v, err := a.Vault(r.Msg.GetVault())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	said, err := a.Counted.Execute(ctx, v)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	from, to := r.Msg.GetFrom(), r.Msg.GetTo()

	days := make([]*v1.ReviewDay, 0, len(said.Days))
	for day, one := range said.Days {
		if !isWithin(day, from, to) {
			continue
		}
		came := said.Retained[day]
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
	due := make([]*v1.ReviewDay, 0, len(said.Due))
	for day, falls := range said.Due {
		if !isWithin(day, from, to) {
			continue
		}
		due = append(due, &v1.ReviewDay{Day: day, Answered: int32(falls)})
	}

	return connect.NewResponse(&v1.ListReviewDaysResponse{
		Days:     inOrder(days),
		Due:      inOrder(due),
		Streak:   int32(said.Streak),
		Answered: int32(said.Answered),
	}), nil
}

// inOrder puts the days oldest first.
func inOrder(days []*v1.ReviewDay) []*v1.ReviewDay {
	slices.SortFunc(days, func(one, other *v1.ReviewDay) int {
		return cmpDay(one.GetDay(), other.GetDay())
	})
	return days
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

// isWithin says whether a day falls inside the stretch asked for, both ends
// inside it. An empty end is unbounded at that end.
//
// A day is written as the year, the month and the day, each padded, so it is
// compared as text the way it is compared in time.
func isWithin(day, from, to string) bool {
	if from != "" && day < from {
		return false
	}
	return to == "" || day <= to
}
