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
		named, _ := newWireFault(fault)
		return named
	})
}
