package service

import (
	"net/http"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestOpenRouterListsTheModelsThatFitTheOperationsOfTheSetting(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{
		APIKey:    testKey,
		IsLabeled: true,
		Models:    []fakeopenai.Model{{ID: "openai/gpt-4o-mini", Name: "GPT-4o mini"}, {ID: "anthropic/claude-3.5-haiku", Name: "Claude 3.5 Haiku"}},
		EmbeddingModels: []fakeopenai.Model{
			{ID: "openai/text-embedding-3-small", Name: "Text Embedding 3 Small"},
		},
	})
	svc := newTestService(t)
	call := newCall("openrouter", address, testKey, "")

	complete, err := svc.ListModels(newContext(t), call, llm.OperationComplete)
	if err != nil {
		t.Fatal(err)
	}
	want := []connection.Option{
		{Value: "anthropic/claude-3.5-haiku", Title: "Claude 3.5 Haiku"},
		{Value: "openai/gpt-4o-mini", Title: "GPT-4o mini"},
	}
	if !slices.Equal(complete, want) {
		t.Errorf("complete models = %+v, want %+v", complete, want)
	}

	embed, err := svc.ListModels(newContext(t), call, llm.OperationEmbed)
	if err != nil {
		t.Fatal(err)
	}
	wantEmbed := []connection.Option{{Value: "openai/text-embedding-3-small", Title: "Text Embedding 3 Small"}}
	if !slices.Equal(embed, wantEmbed) {
		t.Errorf("embed models = %+v, want %+v", embed, wantEmbed)
	}
	paths := []string{fake.Requests()[0].Path, fake.Requests()[1].Path}
	if !slices.Equal(paths, []string{"/models", "/embeddings/models"}) {
		t.Errorf("paths = %v", paths)
	}
}

func TestOpenRouterListsNoModelForASettingOfNoOperationOfText(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey, IsLabeled: true, Models: []fakeopenai.Model{{ID: "openai/gpt-4o-mini"}}})

	models, err := newTestService(t).ListModels(newContext(t), newCall("openrouter", address, testKey, ""), "acp.task")

	if err != nil || len(models) != 0 || len(fake.Requests()) != 0 {
		t.Errorf("models = %+v, %v, requests = %d, want none", models, err, len(fake.Requests()))
	}
}

func TestModelsOfAServiceWithoutLabelsAreListedForEveryOperationByTheirID(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{Models: []fakeopenai.Model{{ID: "qwen2.5:14b"}, {ID: "nomic-embed-text"}}})
	for _, operation := range []string{llm.OperationComplete, llm.OperationEmbed} {
		models, err := newTestService(t).ListModels(newContext(t), newCall("openai-compatible", address, "", ""), operation)
		if err != nil {
			t.Fatal(err)
		}
		want := []connection.Option{{Value: "nomic-embed-text", Title: "nomic-embed-text"}, {Value: "qwen2.5:14b", Title: "qwen2.5:14b"}}
		if !slices.Equal(models, want) {
			t.Errorf("%s: models = %+v, want %+v", operation, models, want)
		}
	}
}

func TestListModelsReportsAKeyTheServiceRejects(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey})
	_, err := newTestService(t).ListModels(newContext(t), newCall("openai", address, "wrong-key", ""), llm.OperationComplete)
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonRejectedKey})
}

func TestListModelsReportsAServiceThatIsDown(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.Fail("/models", http.StatusServiceUnavailable, "")
	_, err := newTestService(t).ListModels(newContext(t), newCall("openai", address, testKey, ""), llm.OperationComplete)
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}

func TestReasoningLevelsAreListedWhereTheTypeKnowsThem(t *testing.T) {
	levels := []connection.Option{{Value: "low", Title: "Low"}, {Value: "medium", Title: "Medium"}, {Value: "high", Title: "High"}}
	cases := map[string]struct {
		typeID, model string
		want          []connection.Option
	}{
		"openrouter":                {"openrouter", "openai/gpt-5", levels},
		"openai reasoning model":    {"openai", "gpt-5-mini", levels},
		"openai o-series":           {"openai", "o4-mini", levels},
		"openai chat model":         {"openai", "gpt-4o-mini", nil},
		"compatible server":         {"openai-compatible", "qwen2.5:14b", nil},
		"compatible server o-named": {"openai-compatible", "o3", nil},
	}
	for name, one := range cases {
		got, err := newTestService(t).ListReasoningLevels(newContext(t), newCall(one.typeID, "http://127.0.0.1:1", "", one.model))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !slices.Equal(got, one.want) {
			t.Errorf("%s: levels = %v, want %v", name, got, one.want)
		}
	}
}

func TestOpenRouterListsTheTextModelsForChatAsForComplete(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{
		APIKey: testKey, IsLabeled: true,
		Models: []fakeopenai.Model{{ID: "openai/gpt-4o-mini", Name: "GPT-4o mini"}},
	})

	models, err := newTestService(t).ListModels(newContext(t), newCall("openrouter", address, testKey, ""), llm.OperationChat)

	want := []connection.Option{{Value: "openai/gpt-4o-mini", Title: "GPT-4o mini"}}
	if err != nil || !slices.Equal(models, want) {
		t.Errorf("chat models = %+v, %v, want %+v", models, err, want)
	}
}
