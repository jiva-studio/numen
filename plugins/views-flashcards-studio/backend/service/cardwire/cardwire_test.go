package cardwire

import (
	"testing"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"github.com/jiva-studio/numen/libs/protocol/schematest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

func TestEveryFaultIsSavedFromOne(t *testing.T) {
	schematest.CheckProduced(t, map[flashcardsv1.ProblemKind]format.Fault{
		flashcardsv1.ProblemKind_PROBLEM_KIND_FIELD_DECLARED_TWICE:   format.FaultTwoFields,
		flashcardsv1.ProblemKind_PROBLEM_KIND_STENCIL_WITHOUT_FIELDS: format.FaultNoFields,
		flashcardsv1.ProblemKind_PROBLEM_KIND_FACE_MISSING_A_SIDE:    format.FaultFaceSide,
		flashcardsv1.ProblemKind_PROBLEM_KIND_PLACEHOLDER_UNDECLARED: format.FaultPlaceholder,
		flashcardsv1.ProblemKind_PROBLEM_KIND_CARD_WITHOUT_A_STENCIL: format.FaultNoStencil,
		flashcardsv1.ProblemKind_PROBLEM_KIND_STENCIL_IS_NOT_ONE:     format.FaultNotAStencil,
		flashcardsv1.ProblemKind_PROBLEM_KIND_FIELD_WRITTEN_TWICE:    format.FaultTwoValues,
		flashcardsv1.ProblemKind_PROBLEM_KIND_FIELD_NOT_RENAMED:      format.FaultNotWritten,
		flashcardsv1.ProblemKind_PROBLEM_KIND_MARK_CARRIED_TWICE:     format.FaultTwoMarks,
	}, func(fault format.Fault) flashcardsv1.ProblemKind {
		named, isNamed := newWireProblemKind(fault)
		if !isNamed {
			t.Errorf("the schema names no fault for %q", fault)
		}
		return named
	})
}

func TestAFaultTheSchemaDoesNotNameIsLeftOut(t *testing.T) {
	named, isNamed := newWireProblemKind(format.FaultTooLarge)
	if isNamed || named != flashcardsv1.ProblemKind_PROBLEM_KIND_UNSPECIFIED {
		t.Errorf("a fault with no name in the schema was answered %v, %v", named, isNamed)
	}
	got := newWireProblems([]format.Problem{
		{Fault: format.FaultTooLarge, Card: format.NoPosition, Face: format.NoPosition},
		{Fault: format.FaultNoFields, Card: 3, Face: format.NoPosition, Detail: "d"},
	})
	if len(got) != 1 || got[0].GetKind() != flashcardsv1.ProblemKind_PROBLEM_KIND_STENCIL_WITHOUT_FIELDS {
		t.Fatalf("got %v", got)
	}
	if got[0].Card == nil || *got[0].Card != 3 || got[0].Face != nil {
		t.Errorf("problem targets card %v, face %v", got[0].Card, got[0].Face)
	}
}

func TestAnAnswerCarriesAtMostTwoHundredStencils(t *testing.T) {
	if MaxStencils != 200 {
		t.Errorf("an answer carries %d stencils", MaxStencils)
	}
}
