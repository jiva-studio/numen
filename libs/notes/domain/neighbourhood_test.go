package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A note can answer to two relations and is shown in one place, so it takes the
// first it qualifies for.
func TestARelationIsTakenInOneOrder(t *testing.T) {
	order := []domain.Relation{domain.RelationParent, domain.RelationChild, domain.RelationJump, domain.RelationSibling}
	for i := 1; i < len(order); i++ {
		if domain.GetRelationRank(order[i-1]) >= domain.GetRelationRank(order[i]) {
			t.Errorf("%q does not come before %q", order[i-1], order[i])
		}
	}
	if domain.GetRelationRank("whatever") <= domain.GetRelationRank(domain.RelationSibling) {
		t.Error("a relation nobody decided on comes before one that was")
	}
}

func TestTheFocusIsNotRelatedToItself(t *testing.T) {
	target := domain.Neighbourhood{Focus: domain.NoteRef{Path: "notes/Entropy.md"}}
	target.AddNeighbour(domain.NoteRef{Path: "notes/Entropy.md"}, domain.Neighbour{Relation: domain.RelationParent})
	if len(target.Related) != 0 {
		t.Fatalf("the focus was added beside itself: %v", target.Related)
	}
	target.AddNeighbour(domain.NoteRef{Path: "notes/Order.md", Title: "Order"}, domain.Neighbour{Relation: domain.RelationChild})
	if len(target.Related) != 1 {
		t.Fatalf("got %v", target.Related)
	}
	if target.Related[0].Path != "notes/Order.md" || target.Related[0].Title != "Order" {
		t.Errorf("the note added is %+v", target.Related[0])
	}
}
