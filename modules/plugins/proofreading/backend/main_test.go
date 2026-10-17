package main

import (
	"context"
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
