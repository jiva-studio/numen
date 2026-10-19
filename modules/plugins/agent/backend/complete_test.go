package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func askCompletion(text string) *connect.Request[numenv1.CompleteRequest] {
	return connect.NewRequest(&numenv1.CompleteRequest{Question: &numenv1.CompletionQuestion{
		Messages: []*numenv1.CompletionMessage{
			{Role: "system", Text: "be brief"},
			{Role: "user", Text: text},
		},
	}})
}

func TestCompleteAnswersFromTheChatAPIWithTheConfiguredKeyAndNoTools(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	var auth string
	var got chatRequest
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		content := "the answer"
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{Message: chatMessage{Role: "assistant", Content: &content}}}})
	}))
	defer chat.Close()
	svc := &agentServer{}
	svc.updateConfig(map[string]string{"apiKey": "host-key", "baseUrl": chat.URL, "model": "m1"})
	svc.tools = []*numenv1.AgentToolDefinition{{Name: "knowledge_search"}}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetAnswer().GetText() != "the answer" {
		t.Fatalf("got %+v", res.Msg)
	}
	if auth != "Bearer host-key" {
		t.Errorf("authorization %q", auth)
	}
	if got.Model != "m1" || len(got.Tools) != 0 || len(got.Messages) != 2 || *got.Messages[0].Content != "be brief" || *got.Messages[1].Content != "why?" {
		t.Errorf("request %+v", got)
	}
}

func TestCompleteReleasesTheReadLock(t *testing.T) {
	svc := &agentServer{}
	_, _ = svc.Complete(context.Background(), askCompletion("why?"))
	if !svc.mu.TryLock() {
		t.Fatal("read lock was not released")
	}
	svc.mu.Unlock()
}

func TestCompleteIsUnavailableWithoutAKey(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("NUMEN_AGENT_KEY", "env-key")
	called := false
	chat := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer chat.Close()
	svc := &agentServer{baseURL: chat.URL}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE || res.Msg.GetAnswer() != nil {
		t.Errorf("got %+v", res.Msg)
	}
	if !strings.Contains(logBuf.String(), errNoKey.Error()) {
		t.Errorf("log %q did not contain %q", logBuf.String(), errNoKey.Error())
	}
	if called {
		t.Error("the chat API was called without a key")
	}
}

func TestCompleteIsUnavailableWhenTheChatAPIFails(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer chat.Close()
	svc := &agentServer{baseURL: chat.URL, apiKey: "k"}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE {
		t.Errorf("got %+v", res.Msg)
	}
	if !strings.Contains(logBuf.String(), "status 401") {
		t.Errorf("log %q did not contain status 401", logBuf.String())
	}
}

func TestCompleteWhenChatReplyContentIsNil(t *testing.T) {
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{Message: chatMessage{Role: "assistant", Content: nil}}}})
	}))
	defer chat.Close()
	svc := &agentServer{baseURL: chat.URL, apiKey: "k"}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE {
		t.Errorf("got %+v", res.Msg)
	}
}

func TestCompleteStopsTheRequestWhenCancelled(t *testing.T) {
	release := make(chan struct{})
	chat := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer chat.Close()
	defer close(release)
	svc := &agentServer{baseURL: chat.URL, apiKey: "k"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := svc.Complete(ctx, askCompletion("why?"))
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v", err)
	}
}

func TestADefaultToolThatCannotRunIsAnError(t *testing.T) {
	t.Setenv("NUMEN_SEARCH_URL", "")
	svc := &agentServer{}
	for _, name := range []string{"knowledge_search", "note_edit"} {
		if out, err := svc.defaultExecuteTool(context.Background(), name, `{"query":"q"}`); err == nil {
			t.Errorf("%s: expected an error, got %q", name, out)
		}
	}
}
