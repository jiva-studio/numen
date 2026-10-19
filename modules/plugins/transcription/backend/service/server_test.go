package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/engine"
)

func TestLifecyclePing(t *testing.T) {
	life := server.NewLifecycleServer(make(chan struct{}))
	resp, err := life.Ping(context.Background(), connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if resp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want %q", resp.Msg.GetStatus(), "ok")
	}
}

func TestLifecycleShutdown(t *testing.T) {
	life := server.NewLifecycleServer(make(chan struct{}))
	resp, err := life.Shutdown(context.Background(), connect.NewRequest(&numenv1.ShutdownRequest{}))
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("shutdown success = false, want true")
	}
}

func TestLifecycleConfigure(t *testing.T) {
	ts := newTranscriptionServer(engine.Config{}, defaultEngineFactory)
	life := server.NewLifecycleServer(make(chan struct{}), server.WithLifecycleUpdateConfig(ts.updateConfig))

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		PluginId: "md.numen.transcription",
		Values: map[string]string{
			"language": "fr",
		},
	})
	resp, err := life.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("configure success = false, want true")
	}
	if ts.cfg.Model.Name != "fr" {
		t.Errorf("expected model name 'fr', got %q", ts.cfg.Model.Name)
	}
}
