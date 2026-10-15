package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleConfigure(t *testing.T) {
	server := &lifecycleServer{shutdownTrigger: make(chan struct{})}

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{
			"zoom": "1.5",
		},
	})
	resp, err := server.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Errorf("Configure success = false, want true")
	}

	server.mu.RLock()
	got := server.config["zoom"]
	server.mu.RUnlock()
	if got != "1.5" {
		t.Errorf("zoom = %q, want %q", got, "1.5")
	}
}
