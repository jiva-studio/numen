package models_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/models"
)

type gateway struct {
	operations []string
	limit      time.Duration
}

func (g *gateway) Complete(context.Context, connection.Call, time.Duration, *llm.CompleteRequest) (*llm.CompleteResult, error) {
	return &llm.CompleteResult{}, nil
}

func (g *gateway) Chat(context.Context, connection.Call, time.Duration, *llm.ChatRequest, func(*llm.ChatFrame) error) (*llm.ChatFinish, error) {
	return &llm.ChatFinish{}, nil
}

func (g *gateway) Embed(context.Context, connection.Call, time.Duration, []string) ([][]float32, error) {
	return nil, nil
}

func (g *gateway) ListModels(_ context.Context, _ connection.Call, limit time.Duration, operations []string) ([]connection.Option, error) {
	g.operations, g.limit = operations, limit
	return []connection.Option{{Value: "m1"}}, nil
}

func newModels(t *testing.T, g *gateway) *models.Models {
	t.Helper()
	subject, err := models.New(g, &limits.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func TestTheModelsAreListedByTheServiceForTheOperationsOfTheSetting(t *testing.T) {
	g := &gateway{}
	call := connection.Call{Type: "openai", Fields: map[string]string{"baseUrl": "https://api.test/v1"}}

	list, err := newModels(t, g).ListModels(t.Context(), call, []string{llm.OperationEmbed})

	if err != nil || len(list) != 1 || !slices.Equal(g.operations, []string{llm.OperationEmbed}) || g.limit != limits.DefaultOtherLimit {
		t.Errorf("list = %v, err = %v, operations = %v, limit = %v", list, err, g.operations, g.limit)
	}
}

func TestTheReasoningLevelsAreThoseOfTheModel(t *testing.T) {
	subject := newModels(t, &gateway{})

	levels, err := subject.ListReasoningLevels(t.Context(), connection.Call{Type: "openrouter", Options: map[string]string{llm.OptionModel: "any"}})
	if err != nil || len(levels) != 3 || levels[0].Value != "low" {
		t.Errorf("levels = %v, err = %v", levels, err)
	}
	none, err := subject.ListReasoningLevels(t.Context(), connection.Call{Type: "openai", Options: map[string]string{llm.OptionModel: "gpt-4o"}})
	if err != nil || len(none) != 0 {
		t.Errorf("levels of a model without reasoning = %v, err = %v", none, err)
	}
}

func TestModelsNeedAGatewayAndLimits(t *testing.T) {
	if _, err := models.New(nil, &limits.Limits{}); err == nil {
		t.Error("models were made with no gateway")
	}
}
