package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

func TestLifecycleServer(t *testing.T) {
	ch := make(chan struct{})
	srv := server.NewLifecycleServer(ch)

	ctx := t.Context()
	isShutdown, err := srv.RequestShutdown(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !isShutdown {
		t.Error("expected shutdown success")
	}
}
