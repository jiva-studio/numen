package cardwire

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/enumtest"
)

func TestEveryFaultIsSavedFromOne(t *testing.T) {
	enumtest.CheckProduced(t, api.ProblemKindValues, map[api.ProblemKind]format.Fault{
		api.ProblemKindFieldDeclaredTwice:     format.FaultTwoFields,
		api.ProblemKindCardModelWithoutFields: format.FaultNoFields,
		api.ProblemKindTemplateMissingASide:   format.FaultTemplateSide,
		api.ProblemKindPlaceholderUndeclared:  format.FaultPlaceholder,
		api.ProblemKindCardWithoutACardModel:  format.FaultNoCardModel,
		api.ProblemKindCardModelIsNotOne:      format.FaultNotACardModel,
		api.ProblemKindFieldWrittenTwice:      format.FaultTwoValues,
		api.ProblemKindFieldNotRenamed:        format.FaultNotWritten,
		api.ProblemKindCardIDCarriedTwice:     format.FaultDuplicateCardID,
	}, func(fault format.Fault) api.ProblemKind {
		named, isNamed := newWireProblemKind(fault)
		if !isNamed {
			t.Errorf("the schema names no fault for %q", fault)
		}
		return named
	})
}

func TestAFaultTheSchemaDoesNotNameIsLeftOut(t *testing.T) {
	named, isNamed := newWireProblemKind(format.FaultTooLarge)
	if isNamed || named != api.ProblemKindUnspecified {
		t.Errorf("a fault with no name in the schema was answered %v, %v", named, isNamed)
	}
	got := newWireProblems([]format.Problem{
		{Fault: format.FaultTooLarge, Card: format.NoPosition, Template: format.NoPosition},
		{Fault: format.FaultNoFields, Card: 3, Template: format.NoPosition, Detail: "d"},
	})
	if len(got) != 1 || got[0].Kind != api.ProblemKindCardModelWithoutFields {
		t.Fatalf("got %v", got)
	}
	if got[0].Card == nil || *got[0].Card != 3 || got[0].Template != nil {
		t.Errorf("problem targets card %v, template %v", got[0].Card, got[0].Template)
	}
}

func TestAnAnswerCarriesAtMostTwoHundredCardModels(t *testing.T) {
	if MaxCardModels != 200 {
		t.Errorf("an answer carries %d card models", MaxCardModels)
	}
}
