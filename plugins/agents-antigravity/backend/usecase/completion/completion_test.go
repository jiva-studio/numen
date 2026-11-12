package completion

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/domain/settings"
)

type fakeAsker struct{}

func (fakeAsker) Ask(context.Context, *agentrun.Request, func(*agentrun.Response) error) error {
	return nil
}

type fakePrograms struct{}

func (fakePrograms) GetState(string) *agentrun.State { return &agentrun.State{} }

func (fakePrograms) GetPrograms(context.Context, string, []agentrun.ProgramCommand) []*agentrun.Program {
	return nil
}

func TestRequestLimitIsTheSettingElseTwoMinutes(t *testing.T) {
	values := new(settings.Settings)
	subject := New(Dependencies{Settings: values, Asker: fakeAsker{}, Programs: fakePrograms{}})
	if got := subject.getRequestLimit(); got != 120*time.Second {
		t.Errorf("default = %v, want 120s", got)
	}
	values.Apply(map[string]string{"requestTimeout": "45"})
	if got := subject.getRequestLimit(); got != 45*time.Second {
		t.Errorf("set to 45 = %v, want 45s", got)
	}
	values.Apply(map[string]string{"requestTimeout": "soon"})
	if got := subject.getRequestLimit(); got != 120*time.Second {
		t.Errorf("invalid value = %v, want 120s", got)
	}
}
