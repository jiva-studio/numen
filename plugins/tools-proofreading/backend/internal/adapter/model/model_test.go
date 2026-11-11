package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/ai"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/prompt"
)

func TestAskSendsTheQuestionWithTheProofreadingPurpose(t *testing.T) {
	var asked []*ai.Question
	complete := func(_ context.Context, question *ai.Question) (string, error) {
		asked = append(asked, question)
		return "the answer", nil
	}

	answer, err := NewAsk(complete, nil)(t.Context(), "be careful", "the prompt")
	if err != nil {
		t.Fatal(err)
	}

	q := asked[0]
	if answer != "the answer" || q.GetPurpose() != "proofreading" || len(q.GetMessages()) != 2 ||
		q.GetMessages()[0].GetRole() != "system" || q.GetMessages()[0].GetText() != "be careful" ||
		q.GetMessages()[1].GetRole() != "user" || q.GetMessages()[1].GetText() != "the prompt" {
		t.Errorf("answer %q to question %v", answer, q)
	}
}

func TestAskSendsTheQuestionOverHTTPWhenTheProviderIsConfigured(t *testing.T) {
	var gotAuth, gotModel, gotSystem, gotPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotModel = req.Model
		if len(req.Messages) == 2 {
			gotSystem, gotPrompt = req.Messages[0].Content, req.Messages[1].Content
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": "the reply"}}},
		}); err != nil {
			t.Errorf("encode the response: %v", err)
		}
	}))
	defer srv.Close()
	completed := 0
	complete := func(context.Context, *ai.Question) (string, error) {
		completed++
		return "", nil
	}
	endpoint := Endpoint{Provider: "custom", BaseURL: srv.URL, Model: "google/gemini-2.5-flash", APIKey: "secret-key"}

	answer, err := NewAsk(complete, func() Endpoint { return endpoint })(t.Context(), "be careful", "the prompt")
	if err != nil {
		t.Fatal(err)
	}

	if answer != "the reply" || completed != 0 {
		t.Errorf("answer %q, completer called %d times", answer, completed)
	}
	if gotAuth != "Bearer secret-key" || gotModel != "google/gemini-2.5-flash" || gotSystem != "be careful" || gotPrompt != "the prompt" {
		t.Errorf("auth %q, model %q, system %q, prompt %q", gotAuth, gotModel, gotSystem, gotPrompt)
	}
}

func TestAskWithoutAServiceURLFails(t *testing.T) {
	endpoint := Endpoint{Provider: "custom", BaseURL: "", Model: "", APIKey: ""}

	if _, err := NewAsk(nil, func() Endpoint { return endpoint })(t.Context(), "be careful", "the prompt"); err == nil {
		t.Error("a custom provider with no URL was asked")
	}
}

func TestTheMockAnswersTheSameQuestionTheSameWay(t *testing.T) {
	question := &ai.Question{Messages: []ai.Message{
		{Role: "system", Text: prompt.SpeechInstruction},
		{Role: "user", Text: "What it holds\n\n" + prompt.BuildPrompt(numbered.Batch{Lines: []numbered.Line{{Number: 4, Text: "one"}, {Number: 5, Text: "two"}}})},
	}}

	first, err := CompleteMock(t.Context(), question)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompleteMock(t.Context(), question)
	if err != nil {
		t.Fatal(err)
	}

	if want := "4|one " + MockMark + "\n5|two " + MockMark; first != want || second != want {
		t.Errorf("the mock answered %q and %q, want %q", first, second, want)
	}
}
