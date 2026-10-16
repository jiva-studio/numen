package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
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
