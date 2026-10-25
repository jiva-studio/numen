package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func TestLifecycleConfigure(t *testing.T) {
	life := server.NewLifecycleServer(make(chan struct{}))

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{
			"zoom": "1.5",
		},
	})
	resp, err := life.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Errorf("Configure success = false, want true")
	}

	got, ok := life.GetConfig("zoom")
	if !ok || got != "1.5" {
		t.Errorf("zoom = %q, want %q", got, "1.5")
	}
}
