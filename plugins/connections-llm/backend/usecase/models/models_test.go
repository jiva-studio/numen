package models_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	models     []connection.Option
	err        error
	calls      int
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
	g.calls++
	if g.err != nil {
		return nil, g.err
	}
	if g.models != nil {
		return g.models, nil
	}
	return []connection.Option{{Value: "m1"}}, nil
}

func newModels(t *testing.T, g *gateway) *models.Models {
	t.Helper()
	return newModelsIn(t, g, t.TempDir())
}

func newModelsIn(t *testing.T, g *gateway, dir string) *models.Models {
	t.Helper()
	subject, err := models.New(g, &limits.Limits{}, models.Lists{Dir: dir, Report: func(error) {}})
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
	if _, err := models.New(nil, &limits.Limits{}, models.Lists{Dir: "", Report: func(error) {}}); err == nil {
		t.Error("models were made with no gateway")
	}
}

const openRouterKey = "sk-or-invented-key"

func newOpenRouterCall(key string) connection.Call {
	return connection.Call{
		Type: "openrouter", Fields: map[string]string{"baseUrl": "https://openrouter.test/api/v1"},
		Secrets: map[string]connection.Secret{"apiKey": {Value: key}},
	}
}

func TestAListOfModelsOutlivesARestartAndIsAnsweredWhileTheServiceFails(t *testing.T) {
	dir := t.TempDir()
	listed := []connection.Option{{Value: "invented/model-a", Title: "Model A"}, {Value: "invented/model-b", Title: "Model B"}}
	if _, err := newModelsIn(t, &gateway{models: listed}, dir).ListModels(t.Context(), newOpenRouterCall(openRouterKey), []string{llm.OperationChat}); err != nil {
		t.Fatal(err)
	}

	down := &gateway{err: errors.New("the service is down")}
	list, err := newModelsIn(t, down, dir).ListModels(t.Context(), newOpenRouterCall(openRouterKey), []string{llm.OperationChat})

	if err != nil || !slices.Equal(list, listed) || down.calls != 0 {
		t.Errorf("after a restart: list = %v, err = %v, calls to the service = %d", list, err, down.calls)
	}
}

func TestListModelsCallsTheServiceOncePerCachedList(t *testing.T) {
	g := &gateway{}
	subject := newModels(t, g)
	for range 3 {
		if _, err := subject.ListModels(t.Context(), newOpenRouterCall(openRouterKey), []string{llm.OperationChat}); err != nil {
			t.Fatal(err)
		}
	}
	if g.calls != 1 {
		t.Errorf("calls to the service = %d, want 1", g.calls)
	}
}

func TestAnotherKeyOrAnotherOperationIsListedApart(t *testing.T) {
	g := &gateway{}
	subject := newModels(t, g)
	calls := []struct {
		key       string
		operation string
	}{{openRouterKey, llm.OperationChat}, {"sk-or-other-invented-key", llm.OperationChat}, {openRouterKey, llm.OperationEmbed}}
	for _, one := range calls {
		if _, err := subject.ListModels(t.Context(), newOpenRouterCall(one.key), []string{one.operation}); err != nil {
			t.Fatal(err)
		}
	}
	if g.calls != len(calls) {
		t.Errorf("calls to the service = %d, want %d", g.calls, len(calls))
	}
}

func TestTheCacheFileContainsTheKeyDigestAndNotTheKey(t *testing.T) {
	dir := t.TempDir()
	if _, err := newModelsIn(t, &gateway{}, dir).ListModels(t.Context(), newOpenRouterCall(openRouterKey), []string{llm.OperationChat}); err != nil {
		t.Fatal(err)
	}
	var stored strings.Builder
	files, err := os.ReadDir(dir)
	if err != nil || len(files) == 0 {
		t.Fatalf("files = %v, err = %v, want the kept list", files, err)
	}
	for _, one := range files {
		data, err := os.ReadFile(filepath.Join(dir, one.Name()))
		if err != nil {
			t.Fatal(err)
		}
		stored.Write(data)
	}
	digest := sha256.Sum256([]byte(openRouterKey))
	if strings.Contains(stored.String(), openRouterKey) || !strings.Contains(stored.String(), hex.EncodeToString(digest[:])) {
		t.Errorf("stored = %s, want the digest of the key and not the key", stored.String())
	}
}

func TestAFirstListThatFailsReturnsTheFailureAndKeepsNothing(t *testing.T) {
	g := &gateway{err: errors.New("the service is down")}
	subject := newModels(t, g)
	if _, err := subject.ListModels(t.Context(), newOpenRouterCall(openRouterKey), []string{llm.OperationChat}); err == nil {
		t.Fatal("a failed list returned no error")
	}
	g.err = nil
	list, err := subject.ListModels(t.Context(), newOpenRouterCall(openRouterKey), []string{llm.OperationChat})
	if err != nil || len(list) != 1 || g.calls != 2 {
		t.Errorf("list = %v, err = %v, calls = %d, want a second call to the service", list, err, g.calls)
	}
}
