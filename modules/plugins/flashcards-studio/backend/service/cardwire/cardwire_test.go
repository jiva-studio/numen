package cardwire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/protocol/schematest"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/format"
)

func TestEveryFaultIsWrittenFromOne(t *testing.T) {
	schematest.CheckProduced(t, map[v1.ProblemKind]format.Fault{
		v1.ProblemKind_PROBLEM_KIND_FIELD_DECLARED_TWICE:   format.FaultTwoFields,
		v1.ProblemKind_PROBLEM_KIND_STENCIL_WITHOUT_FIELDS: format.FaultNoFields,
		v1.ProblemKind_PROBLEM_KIND_FACE_MISSING_A_SIDE:    format.FaultFaceSide,
		v1.ProblemKind_PROBLEM_KIND_PLACEHOLDER_UNDECLARED: format.FaultPlaceholder,
		v1.ProblemKind_PROBLEM_KIND_CARD_WITHOUT_A_STENCIL: format.FaultNoStencil,
		v1.ProblemKind_PROBLEM_KIND_STENCIL_IS_NOT_ONE:     format.FaultNotAStencil,
		v1.ProblemKind_PROBLEM_KIND_FIELD_WRITTEN_TWICE:    format.FaultTwoValues,
		v1.ProblemKind_PROBLEM_KIND_FIELD_NOT_RENAMED:      format.FaultNotWritten,
		v1.ProblemKind_PROBLEM_KIND_MARK_CARRIED_TWICE:     format.FaultTwoMarks,
	}, func(fault format.Fault) v1.ProblemKind {
		named, isNamed := newWireProblemKind(fault)
		if !isNamed {
			t.Errorf("the schema names no fault for %q", fault)
		}
		return named
	})
}

func TestAFaultTheSchemaDoesNotNameIsLeftOut(t *testing.T) {
	named, isNamed := newWireProblemKind(format.FaultTooLarge)
	if isNamed || named != v1.ProblemKind_PROBLEM_KIND_UNSPECIFIED {
		t.Errorf("a fault with no name in the schema was answered %v, %v", named, isNamed)
	}
	got := newWireProblems([]format.Problem{
		{Fault: format.FaultTooLarge, Card: format.NoPosition, Face: format.NoPosition},
		{Fault: format.FaultNoFields, Card: 3, Face: format.NoPosition, Detail: "d"},
	})
	if len(got) != 1 || got[0].GetKind() != v1.ProblemKind_PROBLEM_KIND_STENCIL_WITHOUT_FIELDS {
		t.Fatalf("got %v", got)
	}
	if got[0].Card == nil || *got[0].Card != 3 || got[0].Face != nil {
		t.Errorf("the problem stands at card %v, face %v", got[0].Card, got[0].Face)
	}
}

func TestAnAnswerCarriesAtMostTwoHundredStencils(t *testing.T) {
	if MaxStencils != 200 {
		t.Errorf("an answer carries %d stencils", MaxStencils)
	}
}
