package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleConfigure(t *testing.T) {
	life := server.NewLifecycleServer(make(chan struct{}))

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{
			"theme": "sepia",
		},
	})
	resp, err := life.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Errorf("Configure success = false, want true")
	}

	got, ok := life.GetConfig("theme")
	if !ok || got != "sepia" {
		t.Errorf("theme = %q, want %q", got, "sepia")
	}
}
