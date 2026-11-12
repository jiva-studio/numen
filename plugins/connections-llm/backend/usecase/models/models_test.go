package models_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/models"
)

type gateway struct {
	operation string
	limit     time.Duration
}

func (g *gateway) Complete(context.Context, connection.Call, time.Duration, connection.ProviderCompleteRequest) (connection.CompleteResult, error) {
	return connection.CompleteResult{}, nil
}

func (g *gateway) Chat(context.Context, connection.Call, time.Duration, connection.ProviderChatRequest, func(connection.ChatEvent) error) error {
	return nil
}

func (g *gateway) Embed(context.Context, connection.Call, time.Duration, []string) ([][]float32, error) {
	return nil, nil
}

func (g *gateway) ListModels(_ context.Context, _ connection.Call, limit time.Duration, operation string) ([]connection.Model, error) {
	g.operation, g.limit = operation, limit
	return []connection.Model{{ID: "m1"}}, nil
}

func newModels(t *testing.T, g *gateway) *models.Models {
	t.Helper()
	subject, err := models.New(g, &limits.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func TestTheModelsAreListedByTheServiceForTheOperation(t *testing.T) {
	g := &gateway{}
	call := connection.Call{Type: "openai", Fields: map[string]string{"baseUrl": "https://api.test/v1"}}

	list, err := newModels(t, g).ListModels(t.Context(), call, "embed")

	if err != nil || len(list) != 1 || g.operation != "embed" || g.limit != limits.DefaultOtherLimit {
		t.Errorf("list = %v, err = %v, operation = %q, limit = %v", list, err, g.operation, g.limit)
	}
}

func TestTheReasoningLevelsAreThoseOfTheModel(t *testing.T) {
	subject := newModels(t, &gateway{})

	levels, err := subject.ListReasoningLevels(t.Context(), connection.Call{Type: "openrouter", Model: "any"})
	if err != nil || len(levels) != 3 || levels[0].Value != "low" {
		t.Errorf("levels = %v, err = %v", levels, err)
	}
	none, err := subject.ListReasoningLevels(t.Context(), connection.Call{Type: "openai", Model: "gpt-4o"})
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model without reasoning = %v, err = %v", none, err)
	}
}

func TestModelsNeedAGatewayAndLimits(t *testing.T) {
	if _, err := models.New(nil, &limits.Limits{}); err == nil {
		t.Error("models were made with no gateway")
	}
}
