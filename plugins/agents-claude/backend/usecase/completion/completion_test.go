package completion

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
)

// slowProgram is the outer edge of a completion: a program that answers once the request has ended.
type slowProgram struct{}

func (slowProgram) FindProgram() string { return "/usr/bin/claude" }

func (slowProgram) GetEnvironment() []string { return nil }

func (slowProgram) Answer(ctx context.Context, _ connection.Call, _, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

type noModels struct{}

func (noModels) List(context.Context, string, []string, []string) []model.Model { return nil }

func TestRequestLimitIsTheSettingElseTwoMinutes(t *testing.T) {
	values := new(settings.Settings)
	subject, err := New(Dependencies{Settings: values, Answerer: slowProgram{}, Models: noModels{}})
	if err != nil {
		t.Fatal(err)
	}
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

func TestNewRefusesMissingDependencies(t *testing.T) {
	if _, err := New(Dependencies{}); err == nil {
		t.Error("New accepted no dependencies")
	}
}
