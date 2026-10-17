package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleConfigure(t *testing.T) {
	proof := &proofreadingServer{modelName: "initial-model"}
	life := server.NewLifecycleServer(
		make(chan struct{}),
		server.WithLifecycleUpdateConfig(proof.updateConfig),
	)

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{
			"model": "anthropic/claude-3-haiku",
		},
	})
	resp, err := life.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Errorf("Configure success = false, want true")
	}

	info, err := proof.GetProofreaderInfo(context.Background(), connect.NewRequest(&numenv1.GetProofreaderInfoRequest{}))
	if err != nil {
		t.Fatalf("GetProofreaderInfo failed: %v", err)
	}
	if info.Msg.GetName() != "anthropic/claude-3-haiku" {
		t.Errorf("model = %q, want %q", info.Msg.GetName(), "anthropic/claude-3-haiku")
	}
}

func TestUpdateConfigReadsEverySettingTheManifestDeclares(t *testing.T) {
	proof := &proofreadingServer{}
	proof.updateConfig(map[string]string{
		"model":          "m",
		"baseUrl":        "http://example.test/v1",
		"strictMode":     "true",
		"maxSuggestions": "3",
	})
	if proof.modelName != "m" || proof.baseURL != "http://example.test/v1" || !proof.strict || proof.maxSuggestions != 3 {
		t.Errorf("config = %+v", proof)
	}
	proof.updateConfig(map[string]string{"strictMode": "false", "maxSuggestions": "bad"})
	if proof.strict || proof.maxSuggestions != 3 {
		t.Errorf("config after update = %+v", proof)
	}
}

func TestProofreadAppliesTheSettingsToTheModelCall(t *testing.T) {
	var system string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		system = body.Messages[0].Content
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{
			"role": "assistant", "content": "1|the cat sat on the mat\n2|the dog sat on the mat",
		}}}})
	}))
	defer api.Close()
	t.Setenv("NUMEN_PROOFREADING_KEY", "k")

	proof := &proofreadingServer{}
	proof.updateConfig(map[string]string{"baseUrl": api.URL, "strictMode": "true", "maxSuggestions": "1"})
	prompt := Opens + "1" + Closes + "the cta sat on the mat " + Opens + "2" + Closes + "the dgo sat on the mat"
	resp, err := proof.Proofread(context.Background(), connect.NewRequest(&numenv1.ProofreadRequest{
		Instruction: "base",
		Batches:     []*numenv1.ProofreadBatch{{Number: 0, Prompt: prompt}},
	}))
	if err != nil {
		t.Fatalf("Proofread failed: %v", err)
	}
	if got := resp.Msg.GetReplies()[0]; got != "1|the cat sat on the mat" {
		t.Errorf("reply = %q", got)
	}
	if !strings.HasPrefix(system, "base") || system == "base" {
		t.Errorf("instruction = %q, want base plus the strict rule", system)
	}
}

func TestGetProofreaderInfoDefault(t *testing.T) {
	proof := &proofreadingServer{}
	info, err := proof.GetProofreaderInfo(context.Background(), connect.NewRequest(&numenv1.GetProofreaderInfoRequest{}))
	if err != nil {
		t.Fatalf("GetProofreaderInfo failed: %v", err)
	}
	if got, want := info.Msg.GetName(), "llm-proofreader"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
}

func TestUpdateConfig(t *testing.T) {
	proof := &proofreadingServer{modelName: "initial-model"}

	proof.updateConfig(map[string]string{"other": "ignored"})
	if got, want := proof.modelName, "initial-model"; got != want {
		t.Errorf("modelName = %q, want %q", got, want)
	}

	proof.updateConfig(map[string]string{"model": ""})
	if got, want := proof.modelName, "initial-model"; got != want {
		t.Errorf("modelName = %q, want %q", got, want)
	}

	proof.updateConfig(map[string]string{"model": "updated-model"})
	if got, want := proof.modelName, "updated-model"; got != want {
		t.Errorf("modelName = %q, want %q", got, want)
	}
}

func TestGetProofreaderInfoReleasesLock(t *testing.T) {
	proof := &proofreadingServer{modelName: "test-model"}
	_, err := proof.GetProofreaderInfo(context.Background(), connect.NewRequest(&numenv1.GetProofreaderInfoRequest{}))
	if err != nil {
		t.Fatalf("GetProofreaderInfo failed: %v", err)
	}

	if !proof.mu.TryLock() {
		t.Fatal("expected write lock to succeed; read lock was not released")
	}
	proof.mu.Unlock()

	proof.updateConfig(map[string]string{"model": "new-model"})
	if got, want := proof.modelName, "new-model"; got != want {
		t.Errorf("modelName = %q, want %q", got, want)
	}
}
