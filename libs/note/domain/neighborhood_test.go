package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// A note can answer to two relations and is shown in one place, so it takes the
// first it qualifies for.
func TestARelationIsTakenInOneOrder(t *testing.T) {
	order := []domain.Relation{domain.RelationParent, domain.RelationChild, domain.RelationJump, domain.RelationSibling}
	for i, rel := range order {
		if got := domain.GetRelationRank(rel); got != i {
			t.Errorf("GetRelationRank(%q) = %d, want %d", rel, got, i)
		}
	}
	if got := domain.GetRelationRank("whatever"); got != 4 {
		t.Errorf("GetRelationRank(whatever) = %d, want 4", got)
	}
}

func TestTheFocusIsNotRelatedToItself(t *testing.T) {
	target := domain.Neighborhood{Focus: domain.NoteRef{Path: "notes/Entropy.md"}}
	target.AddNeighbor(domain.NoteRef{Path: "notes/Entropy.md"}, domain.Neighbor{Relation: domain.RelationParent})
	if len(target.Related) != 0 {
		t.Fatalf("the focus was added beside itself: %v", target.Related)
	}
	target.AddNeighbor(domain.NoteRef{Path: "notes/Order.md", Title: "Order"}, domain.Neighbor{Relation: domain.RelationChild})
	if len(target.Related) != 1 {
		t.Fatalf("got %v", target.Related)
	}
	if target.Related[0].Path != "notes/Order.md" || target.Related[0].Title != "Order" {
		t.Errorf("the note added is %+v", target.Related[0])
	}
}
