package review

import (
	"cmp"
	"math"
	"slices"
)

// GetShares is a budget handed out in proportion to what each deck owes of it.
//
// A deck owing nine times another's takes nine times the share, a deck owing
// nothing takes nothing, and no deck takes more than it owes — the remainder is
// handed out a card at a time. What the proportions leave over goes by the
// largest fraction, and where the fractions stand equal the deck owing more
// takes it; decks owing the same take it in the order they are given.
//
// The proportion itself is worked out in floating point: a budget of
// nanoseconds times what one deck owes of them overflows an integer. Every
// share handed back is whole.
func GetShares(budget int, owes []int) []int {
	out := make([]int, len(owes))
	total := 0
	for _, one := range owes {
		total += one
	}
	if total <= 0 {
		return out
	}
	if budget >= total {
		copy(out, owes)
		return out
	}
	over := make([]int, len(owes))
	parts := make([]float64, len(owes))
	left := budget
	for at, one := range owes {
		exact := float64(budget) * float64(one) / float64(total)
		out[at] = int(math.Floor(exact))
		parts[at] = exact - float64(out[at])
		left -= out[at]
		over[at] = at
	}
	slices.SortStableFunc(over, func(a, b int) int {
		return cmp.Or(
			cmp.Compare(parts[b], parts[a]),
			cmp.Compare(owes[b], owes[a]),
		)
	})
	for _, at := range over[:min(len(over), left)] {
		out[at]++
	}
	return out
}
