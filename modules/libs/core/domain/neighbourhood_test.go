package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A note can answer to two seats and is shown in one place, so it takes the
// first it qualifies for.
func TestASeatIsTakenInOneOrder(t *testing.T) {
	order := []domain.Relation{domain.SeatParent, domain.SeatChild, domain.SeatJump, domain.SeatSibling}
	for i := 1; i < len(order); i++ {
		if domain.SeatRank(order[i-1]) >= domain.SeatRank(order[i]) {
			t.Errorf("%q does not come before %q", order[i-1], order[i])
		}
	}
	if domain.SeatRank("whatever") <= domain.SeatRank(domain.SeatSibling) {
		t.Error("a seat nobody decided on comes before one that was")
	}
}

func TestTheFocusIsNotRelatedToItself(t *testing.T) {
	held := domain.Neighbourhood{Focus: domain.NoteRef{Path: "notes/Entropy.md"}}
	held.Take(domain.NoteRef{Path: "notes/Entropy.md"}, domain.Neighbour{Seat: domain.SeatParent})
	if len(held.Related) != 0 {
		t.Fatalf("the focus was seated beside itself: %v", held.Related)
	}
	held.Take(domain.NoteRef{Path: "notes/Order.md", Title: "Order"}, domain.Neighbour{Seat: domain.SeatChild})
	if len(held.Related) != 1 {
		t.Fatalf("got %v", held.Related)
	}
	if held.Related[0].Path != "notes/Order.md" || held.Related[0].Title != "Order" {
		t.Errorf("the note seated is %+v", held.Related[0])
	}
}
