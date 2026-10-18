package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// newChatProbe is a chat API that answers "done" and keeps the Authorization
// header of every request.
func newChatProbe(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		content := "done"
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{Message: chatMessage{Role: "assistant", Content: &content}, FinishReason: "stop"}}})
	}))
	t.Cleanup(chat.Close)
	return chat, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func askAgent(t *testing.T, svc *agentServer) {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(svc))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, srv.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{Prompt: "hello"}))
	if err != nil {
		t.Fatal(err)
	}
	stopped := false
	for stream.Receive() {
		if stream.Msg().GetStopped() != "" {
			stopped = true
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if !stopped {
		t.Fatal("the agent did not stop")
	}
}

// The key in the environment of the plugin is not a key: only the one Configure
// hands over opens the chat API.
func TestAskAgentUsesOnlyTheKeyConfigureHandsOver(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("NUMEN_AGENT_KEY", "env-key")
	chat, seen := newChatProbe(t)
	svc := &agentServer{}
	svc.updateConfig(map[string]string{"baseUrl": chat.URL})

	askAgent(t, svc)
	if got := seen(); len(got) != 0 {
		t.Fatalf("the chat API was called with %v while Configure handed over no key", got)
	}

	svc.updateConfig(map[string]string{"apiKey": "host-key"})
	askAgent(t, svc)
	if got := seen(); len(got) != 1 || got[0] != "Bearer host-key" {
		t.Fatalf("authorization = %v, want the key Configure handed over", got)
	}
}

// An empty key in a later Configure call drops the one the plugin holds.
func TestAnEmptyKeyFromConfigureDropsTheKey(t *testing.T) {
	chat, seen := newChatProbe(t)
	svc := &agentServer{}
	svc.updateConfig(map[string]string{"apiKey": "host-key", "baseUrl": chat.URL})
	svc.updateConfig(map[string]string{"apiKey": ""})

	askAgent(t, svc)
	if got := seen(); len(got) != 0 {
		t.Fatalf("the chat API was called with %v after the key was dropped", got)
	}
}
