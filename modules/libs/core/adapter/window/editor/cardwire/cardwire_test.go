package cardwire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

func TestEveryFaultIsWrittenFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Fault]format.Fault{
		v1.Fault_FAULT_FIELD_DECLARED_TWICE:   format.FaultTwoFields,
		v1.Fault_FAULT_STENCIL_WITHOUT_FIELDS: format.FaultNoFields,
		v1.Fault_FAULT_FACE_MISSING_A_SIDE:    format.FaultFaceSide,
		v1.Fault_FAULT_PLACEHOLDER_UNDECLARED: format.FaultPlaceholder,
		v1.Fault_FAULT_CARD_WITHOUT_A_STENCIL: format.FaultNoStencil,
		v1.Fault_FAULT_STENCIL_IS_NOT_ONE:     format.FaultNotAStencil,
		v1.Fault_FAULT_FIELD_WRITTEN_TWICE:    format.FaultTwoValues,
		v1.Fault_FAULT_FIELD_NOT_RENAMED:      format.FaultNotWritten,
		v1.Fault_FAULT_MARK_CARRIED_TWICE:     format.FaultTwoMarks,
	}, func(fault format.Fault) v1.Fault {
		named, isNamed := newWireFault(fault)
		if !isNamed {
			t.Errorf("the schema names no fault for %q", fault)
		}
		return named
	})
}

func TestAFaultTheSchemaDoesNotNameIsLeftOut(t *testing.T) {
	named, isNamed := newWireFault(format.FaultTooLarge)
	if isNamed || named != v1.Fault_FAULT_UNSPECIFIED {
		t.Errorf("a fault with no name in the schema was answered %v, %v", named, isNamed)
	}
	got := newWireProblems([]format.Problem{
		{Fault: format.FaultTooLarge, Card: format.NoPosition, Face: format.NoPosition},
		{Fault: format.FaultNoFields, Card: 3, Face: format.NoPosition, Detail: "d"},
	})
	if len(got) != 1 || got[0].GetFault() != v1.Fault_FAULT_STENCIL_WITHOUT_FIELDS {
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
