package completion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/agentmodel"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
)

// slowProgram is the outer edge of a completion: a program that answers once the request has ended.
type slowProgram struct{}

func (slowProgram) FindProgram() string { return "/usr/bin/codex" }

func (slowProgram) Answer(ctx context.Context, _ connection.Call, _, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

type stubProgram struct {
	answer string
	err    error
}

func (stubProgram) FindProgram() string { return "/usr/bin/codex" }

func (s stubProgram) Answer(context.Context, connection.Call, string, string) (string, error) {
	return s.answer, s.err
}

type noModels struct{}

func (noModels) Get(context.Context, []string) []agentmodel.Model { return nil }

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
	values.Apply(map[string]string{"requestTimeout": "0.000000001"})
	if got := subject.getRequestLimit(); got != time.Nanosecond {
		t.Errorf("set to 1ns = %v, want 1ns", got)
	}
	values.Apply(map[string]string{"requestTimeout": "soon"})
	if got := subject.getRequestLimit(); got != 120*time.Second {
		t.Errorf("invalid value = %v, want 120s", got)
	}
}

func TestNewRefusesMissingDependencies(t *testing.T) {
	values := new(settings.Settings)
	for name, deps := range map[string]Dependencies{
		"no settings": {Answerer: slowProgram{}, Models: noModels{}},
		"no answerer": {Settings: values, Models: noModels{}},
		"no models":   {Settings: values, Answerer: slowProgram{}},
	} {
		if _, err := New(deps); err == nil {
			t.Errorf("%s: New accepted missing dependency", name)
		}
	}
}

func TestCompleteAnswersAndClassifiesFailures(t *testing.T) {
	values := new(settings.Settings)
	req := &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: "hi"}}}

	okComp, err := New(Dependencies{Settings: values, Answerer: stubProgram{answer: "done"}, Models: noModels{}})
	if err != nil {
		t.Fatal(err)
	}
	if res, err := okComp.Complete(t.Context(), connection.Call{}, req); err != nil || res.GetText() != "done" {
		t.Errorf("Complete success = (%v, %v), want done", res, err)
	}

	failComp, err := New(Dependencies{Settings: values, Answerer: stubProgram{err: errors.New("boom")}, Models: noModels{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := failComp.Complete(t.Context(), connection.Call{}, req); !connection.IsStop(err) {
		t.Errorf("Complete runErr = %v, want a stop", err)
	}

	values.Apply(map[string]string{"requestTimeout": "0.02"})
	slowComp, err := New(Dependencies{Settings: values, Answerer: slowProgram{}, Models: noModels{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slowComp.Complete(t.Context(), connection.Call{}, req); err == nil {
		t.Error("Complete timeout succeeded, want try-next error")
	} else if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("Complete timeout = %v, want try-next error", err)
	}
}
