package chatapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/chatapi"
)

func TestTemperatureAndMaxTokensOfTheSettingsAreSentWithTheRequest(t *testing.T) {
	var receivedBody map[string]any
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Errorf("decode the request: %v", err)
		}
		reply := map[string]any{"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": "hi"}}}}
		if err := json.NewEncoder(w).Encode(reply); err != nil {
			t.Errorf("encode the response: %v", err)
		}
	}))
	defer chat.Close()
	store := settings.NewStore()
	store.Configure(map[string]string{"baseUrl": chat.URL, "temperature": "0.5", "maxTokens": "2048"})
	turn := conversation.Turn{
		Model: "m1", Effort: conversation.Effort{}, Tools: nil,
		Messages: []conversation.Message{{Role: "user", Content: new("hello")}},
	}

	_, err := chatapi.New(chatapi.WithHTTPClient(chat.Client())).Send(t.Context(), store.Get(), turn)
	if err != nil {
		t.Fatal(err)
	}

	if temp, ok := receivedBody["temperature"].(float64); !ok || temp != 0.5 {
		t.Errorf("expected temperature 0.5, got %v", receivedBody["temperature"])
	}
	if maxT, ok := receivedBody["max_tokens"].(float64); !ok || int(maxT) != 2048 {
		t.Errorf("expected max_tokens 2048, got %v", receivedBody["max_tokens"])
	}
}

// openrouter-models.json holds three entries of the listing OpenRouter serves without a key: a model
// that names its depths, one that lists the fields and no depths, and one that lists neither.
func TestTheDepthsAModelConsumesAreReadFromTheListingOfTheChatAPI(t *testing.T) {
	raw, err := os.ReadFile("testdata/openrouter-models.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(raw); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)

	models, err := chatapi.New(chatapi.WithHTTPClient(srv.Client())).ListModels(t.Context(), settings.Settings{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]conversation.Model{}
	for _, one := range models {
		byName[one.Name] = one
	}
	luna := byName["openai/gpt-5.6-luna"]
	var names []string
	for _, one := range luna.ReasoningLevels {
		names = append(names, one.Name)
	}
	if !slices.Equal(names, []string{"low", "medium", "high", "xhigh", "max"}) || luna.DefaultReasoning != "medium" {
		t.Errorf("the model names %v and %q", names, luna.DefaultReasoning)
	}
	if luna.ReasoningField != conversation.FieldReasoningEffort {
		t.Errorf("the field is %q", luna.ReasoningField)
	}
	router := byName["typesafe/jev-router"]
	if len(router.ReasoningLevels) != len(conversation.GetCommonLevels()) || router.DefaultReasoning != "" {
		t.Errorf("a model of fields and no depths offers %v", router.ReasoningLevels)
	}
	if unreasoningModel := byName["unbiased/pareto-26.10"]; len(unreasoningModel.ReasoningLevels) != 0 || unreasoningModel.ReasoningField != "" {
		t.Errorf("a model of no reasoning offers %v", unreasoningModel)
	}
	if request := conversation.NewEffort(luna, "max"); request.Level != "max" {
		t.Errorf("a depth the model names failed")
	}
	if requestVal := conversation.NewEffort(router, "max"); requestVal.Level != "" {
		t.Errorf("a depth the model does not name was asked")
	}
}
