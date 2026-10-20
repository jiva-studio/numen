package flashcards

import (
	"context"
	"maps"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetReviewDays is the days of a vault that fall inside a stretch.
type GetReviewDays struct {
	counted CountReviews
}

func NewGetReviewDays(counted CountReviews) GetReviewDays {
	return GetReviewDays{counted: counted}
}

// Execute keeps the days from from to to, both ends inside it; an empty end is
// unbounded at that end. The streak and the answer count are over the whole
// vault.
//
// A day is written as the year, the month and the day, each padded, so it is
// compared as text the way it is compared in time.
func (u GetReviewDays) Execute(
	ctx context.Context, v domain.Vault, from, to string,
) (ReviewCounts, error) {
	counts, err := u.counted.Execute(ctx, v)
	if err != nil {
		return ReviewCounts{}, err
	}
	counts.Days = keepDays(counts.Days, from, to)
	counts.Due = keepDays(counts.Due, from, to)
	counts.Retained = keepDays(counts.Retained, from, to)
	return counts, nil
}

// WithDay returns a copy of GetReviewDays with the given Day.
func (u GetReviewDays) WithDay(day review.Day) GetReviewDays {
	u.counted = u.counted.WithDay(day)
	return u
}

// WithNow returns a copy of GetReviewDays with the given Clock.
func (u GetReviewDays) WithNow(now port.Clock) GetReviewDays {
	u.counted = u.counted.WithNow(now)
	return u
}

func keepDays[T any](days map[string]T, from, to string) map[string]T {
	kept := maps.Clone(days)
	maps.DeleteFunc(kept, func(day string, _ T) bool {
		return day < from || (to != "" && day > to)
	})
	return kept
}
