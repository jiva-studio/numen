package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/engine"
)

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

func TestGetEngineSurvivesCallerCancellation(t *testing.T) {
	loaded := false
	factory := func(ctx context.Context, _ engine.Config) (engine.TranscriberPort, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		loaded = true
		return engine.NewMockTranscriber(), nil
	}
	ts := newTranscriptionServer(engine.Config{}, factory)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e, err := ts.getEngine(ctx)
	if err != nil {
		t.Fatalf("getEngine failed: %v", err)
	}
	if e == nil || !loaded {
		t.Fatal("expected engine to be loaded despite caller cancellation")
	}
}
