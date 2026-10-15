package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/engine"
)

func TestLifecyclePing(t *testing.T) {
	server := &lifecycleServer{shutdownTrigger: make(chan struct{})}
	resp, err := server.Ping(context.Background(), connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if resp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want %q", resp.Msg.GetStatus(), "ok")
	}
}

func TestLifecycleShutdown(t *testing.T) {
	server := &lifecycleServer{shutdownTrigger: make(chan struct{})}
	resp, err := server.Shutdown(context.Background(), connect.NewRequest(&numenv1.ShutdownRequest{}))
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("shutdown success = false, want true")
	}
}

func TestLifecycleConfigure(t *testing.T) {
	ts := newTranscriptionServer(engine.Config{})
	server := &lifecycleServer{
		shutdownTrigger: make(chan struct{}),
		ts:              ts,
	}

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		PluginId: "md.numen.transcription",
		Values: map[string]string{
			"language": "fr",
		},
	})
	resp, err := server.Configure(context.Background(), req)
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
