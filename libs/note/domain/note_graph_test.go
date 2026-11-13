package domain_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// A note can answer to two relations and is shown in one place, so it takes the
// first it qualifies for.
func TestARelationIsTakenInOneOrder(t *testing.T) {
	order := []domain.Relation{domain.RelationParent, domain.RelationChild, domain.RelationJump, domain.RelationSibling, domain.RelationReference}
	for i, rel := range order {
		if got := domain.GetRelationRank(rel); got != i {
			t.Errorf("GetRelationRank(%q) = %d, want %d", rel, got, i)
		}
	}
	if got := domain.GetRelationRank("whatever"); got != 5 {
		t.Errorf("GetRelationRank(whatever) = %d, want 5", got)
	}
}

func TestTheFocusIsNotRelatedToItself(t *testing.T) {
	target := domain.NoteGraph{Focus: domain.NoteRef{Path: "notes/Entropy.md"}}
	target.AddRelatedNote(domain.NoteRef{Path: "notes/Entropy.md"}, domain.RelatedNote{Relation: domain.RelationParent})
	if len(target.Related) != 0 {
		t.Fatalf("the focus was added beside itself: %v", target.Related)
	}
	target.AddRelatedNote(domain.NoteRef{Path: "notes/Order.md", Title: "Order"}, domain.RelatedNote{Relation: domain.RelationChild})
	if len(target.Related) != 1 {
		t.Fatalf("got %v", target.Related)
	}
	if target.Related[0].Path != "notes/Order.md" || target.Related[0].Title != "Order" {
		t.Errorf("the note added is %+v", target.Related[0])
	}
}

func TestARelatedNoteKeepsThePathOfTheNoteItBelongsTo(t *testing.T) {
	target := domain.NoteGraph{Focus: domain.NoteRef{Path: "Entropy.md"}}
	target.AddRelatedNote(domain.NoteRef{Path: "Engine.md", Title: "Engine"},
		domain.RelatedNote{Relation: domain.RelationChild, RelatedTo: "Heat.md"})
	target.AddRelatedNote(domain.NoteRef{Path: "Heat.md", Title: "Heat"}, domain.RelatedNote{Relation: domain.RelationChild})
	if got := target.Related[0].RelatedTo; got != "Heat.md" {
		t.Errorf("an item of depth 2 belongs to %q, want Heat.md", got)
	}
	if got := target.Related[1].RelatedTo; got != "" {
		t.Errorf("a note related to the focus belongs to %q, want none", got)
	}
}

func TestAWireDepthIsReadAsNearOrFar(t *testing.T) {
	for wire, want := range map[uint32]domain.RelationDepth{0: domain.DepthNear, 1: domain.DepthNear, 2: domain.DepthFar} {
		got, err := domain.ParseRelationDepth(wire)
		if err != nil || got != want {
			t.Errorf("ParseRelationDepth(%d) = %v, %v, want %v", wire, got, err, want)
		}
	}
}

func TestAWireDepthAboveTwoIsInvalid(t *testing.T) {
	for _, wire := range []uint32{3, 100} {
		if _, err := domain.ParseRelationDepth(wire); !errors.Is(err, domain.ErrInvalidRelationDepth) {
			t.Errorf("ParseRelationDepth(%d) error = %v, want ErrInvalidRelationDepth", wire, err)
		}
	}
}
