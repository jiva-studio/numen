package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleServer(t *testing.T) {
	ch := make(chan struct{})
	srv := server.NewLifecycleServer(ch)

	ctx := context.Background()
	pingResp, err := srv.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("expected ok, got %s", pingResp.Msg.GetStatus())
	}

	shutResp, err := srv.Shutdown(ctx, connect.NewRequest(&numenv1.ShutdownRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !shutResp.Msg.GetSuccess() {
		t.Error("expected shutdown success")
	}
}
