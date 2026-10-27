package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

func TestLifecycleServer(t *testing.T) {
	ch := make(chan struct{})
	srv := server.NewLifecycleServer(ch)

	ctx := context.Background()
	pingResp, err := srv.Ping(ctx, connect.NewRequest(&server.PingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("expected ok, got %s", pingResp.Msg.GetStatus())
	}

	shutResp, err := srv.Shutdown(ctx, connect.NewRequest(&server.ShutdownRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !shutResp.Msg.GetSuccess() {
		t.Error("expected shutdown success")
	}
}
