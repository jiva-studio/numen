package completion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
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
	values.Apply(map[string]string{"requestTimeout": "1e-9"})
	if got := subject.getRequestLimit(); got != time.Nanosecond {
		t.Errorf("1ns value = %v, want 1ns", got)
	}
}

func TestNewRefusesMissingDependencies(t *testing.T) {
	values := new(settings.Settings)
	for _, deps := range []Dependencies{
		{Settings: nil, Answerer: slowProgram{}, Models: noModels{}},
		{Settings: values, Answerer: nil, Models: noModels{}},
		{Settings: values, Answerer: slowProgram{}, Models: nil},
	} {
		if _, err := New(deps); err == nil {
			t.Errorf("New(%+v) accepted missing dependency", deps)
		}
	}
}

type answerFunc func(ctx context.Context) (string, error)

func (answerFunc) FindProgram() string { return "/usr/bin/claude" }

func (answerFunc) GetEnvironment() []string { return nil }

func (f answerFunc) Answer(ctx context.Context, _ connection.Call, _, _ string) (string, error) {
	return f(ctx)
}

func TestCompleteHandlesTimeoutAndRunErrors(t *testing.T) {
	req := &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: "hi"}}}
	values := new(settings.Settings)
	values.Apply(map[string]string{"requestTimeout": "1e-4"})

	timedOut, err := New(Dependencies{Settings: values, Answerer: slowProgram{}, Models: noModels{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := timedOut.Complete(t.Context(), connection.Call{}, req); err == nil {
		t.Error("expected timeout error")
	} else if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("timeout err = %v, want TryNext", err)
	}

	values.Apply(nil)
	tryNextSubj, err := New(Dependencies{
		Settings: values,
		Answerer: answerFunc(func(context.Context) (string, error) {
			return "", connection.NewTryNextError(connection.ReasonRateLimited, 0)
		}),
		Models: noModels{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tryNextSubj.Complete(t.Context(), connection.Call{}, req); err == nil {
		t.Error("expected TryNext error")
	} else if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Errorf("err = %v, want TryNext", err)
	}

	stopSubj, err := New(Dependencies{
		Settings: values,
		Answerer: answerFunc(func(context.Context) (string, error) {
			return "", errors.New("other error")
		}),
		Models: noModels{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stopSubj.Complete(t.Context(), connection.Call{}, req); err == nil {
		t.Error("expected stop error")
	} else if _, isTryNext := connection.IsTryNext(err); isTryNext {
		t.Errorf("err = %v, want non-TryNext stop error", err)
	}

	okSubj, err := New(Dependencies{
		Settings: values,
		Answerer: answerFunc(func(context.Context) (string, error) {
			return "done", nil
		}),
		Models: noModels{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res, err := okSubj.Complete(t.Context(), connection.Call{}, req); err != nil || res.GetText() != "done" {
		t.Errorf("Complete = (%v, %v), want (\"done\", nil)", res, err)
	}
}
