package catalog

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
)

func TestTheAgentIsUnreachableWhileNoRowNamesAConnection(t *testing.T) {
	store := settings.NewStore()
	subject, err := New(store)
	if err != nil {
		t.Fatal(err)
	}

	if got := subject.GetState().Unreachable; got != conversation.ReasonNoConnection {
		t.Errorf("unreachable = %q, want %q", got, conversation.ReasonNoConnection)
	}
	if programs := subject.GetPrograms(); len(programs) != 1 || programs[0].IsInstalled || len(programs[0].Models) != 0 {
		t.Errorf("the programs are %+v", programs)
	}
}

func TestTheModelOfTheRowIsTheOneModelOfTheProgram(t *testing.T) {
	store := settings.NewStore()
	store.Configure(map[string]string{"connections": `[{"connection":"c1","model":"m1"}]`})
	subject, err := New(store)
	if err != nil {
		t.Fatal(err)
	}

	if got := subject.GetState().Unreachable; got != "" {
		t.Errorf("unreachable = %q, want none", got)
	}
	programs := subject.GetPrograms()
	if len(programs) != 1 || !programs[0].IsInstalled || !programs[0].LimitsTools ||
		len(programs[0].Models) != 1 || programs[0].Models[0].Name != "m1" || !programs[0].Models[0].IsDefault {
		t.Errorf("the programs are %+v", programs)
	}
}

func TestACatalogNeedsSettings(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Error("New accepted no settings")
	}
}
