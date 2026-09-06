package review

import (
	"cmp"
	"math"
	"slices"
)

// Divided hands a budget out in proportion to what each deck owes of it.
//
// A deck owing nine times another's takes nine times the share, a deck owing
// nothing takes nothing, and no deck owing a whole number takes more than it
// owes — the remainder is handed out a card at a time. What the proportions
// leave over goes by the largest fraction, and where the fractions
// stand equal the deck owing more takes it: the card moves the smaller deck
// further off its proportion than the larger. Decks owing the same are alike in
// everything the division knows of them, and take it in the order they are
// given.
func Divided(budget float64, owes []float64) []float64 {
	out := make([]float64, len(owes))
	var total float64
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
		exact := budget * one / total
		out[at] = math.Floor(exact)
		parts[at] = exact - out[at]
		left -= out[at]
		over[at] = at
	}
	slices.SortStableFunc(over, func(a, b int) int {
		return cmp.Or(
			cmp.Compare(parts[b], parts[a]),
			cmp.Compare(owes[b], owes[a]),
		)
	})
	for _, at := range over[:min(len(over), int(left))] {
		out[at]++
	}
	return out
}
