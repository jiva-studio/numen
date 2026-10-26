package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func TestEmbeddingServerGetModel(t *testing.T) {
	srv := newEmbeddingServer()
	resp, err := srv.GetEmbeddingModel(context.Background(), connect.NewRequest(&numenv1.GetEmbeddingModelRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := resp.Msg
	if msg.GetName() != "bge-m3" {
		t.Errorf("expected name bge-m3, got %s", msg.GetName())
	}
	if msg.GetDimensions() != 1024 {
		t.Errorf("expected 1024 dimensions, got %d", msg.GetDimensions())
	}
	if msg.GetMaxTokens() != 8192 {
		t.Errorf("expected 8192 max tokens, got %d", msg.GetMaxTokens())
	}
	if msg.GetPooling() != "head" {
		t.Errorf("expected head pooling, got %s", msg.GetPooling())
	}
}

func TestEmbeddingServerEnvOverrides(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "custom-model")
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "512")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "1024")
	t.Setenv("NUMEN_EMBEDDING_POOLING", "mean")

	srv := newEmbeddingServer()
	resp, err := srv.GetEmbeddingModel(context.Background(), connect.NewRequest(&numenv1.GetEmbeddingModelRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := resp.Msg
	if msg.GetName() != "custom-model" {
		t.Errorf("expected custom-model, got %s", msg.GetName())
	}
	if msg.GetDimensions() != 512 {
		t.Errorf("expected 512 dimensions, got %d", msg.GetDimensions())
	}
	if msg.GetMaxTokens() != 1024 {
		t.Errorf("expected 1024 max tokens, got %d", msg.GetMaxTokens())
	}
	if msg.GetPooling() != "mean" {
		t.Errorf("expected mean pooling, got %s", msg.GetPooling())
	}
}

func TestLifecycleConfigure(t *testing.T) {
	emb := newEmbeddingServer()
	life := server.NewLifecycleServer(make(chan struct{}), server.WithLifecycleUpdateConfig(emb.updateConfig))

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		PluginId: "md.numen.engines.embeddings",
		Values: map[string]string{
			"model": "text-embedding-3-small",
		},
	})
	resp, err := life.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("configure success = false, want true")
	}
	if emb.modelInfo.Name != "text-embedding-3-small" {
		t.Errorf("expected model text-embedding-3-small, got %s", emb.modelInfo.Name)
	}
}

func TestEmbeddingServerCustomRepositoryIsNotPinned(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_REPO", "owner/repo")
	srv := newEmbeddingServer()
	if srv.localModel.Name != "owner/repo" || srv.localModel.Revision != "" || srv.localModel.Sums != nil {
		t.Errorf("localModel = %+v", srv.localModel)
	}
}

func TestEmbeddingServerCustomModelIsNotPinned(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "owner/model")
	srv := newEmbeddingServer()
	if srv.localModel.Name != "owner/model" || srv.localModel.Sums != nil {
		t.Errorf("localModel = %+v", srv.localModel)
	}
}

func TestEmbeddingServerDefaultModelIsPinned(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "bge-m3-int8")
	srv := newEmbeddingServer()
	if srv.localModel.Name != "Xenova/bge-m3" || srv.localModel.Sums == nil {
		t.Errorf("localModel = %+v", srv.localModel)
	}
}

func TestTheMockEmbedderServesWhereTheEnvironmentAsksForIt(t *testing.T) {
	t.Setenv(envMockML, "true")
	srv := newEmbeddingServer()

	model, err := srv.GetEmbeddingModel(t.Context(), connect.NewRequest(&numenv1.GetEmbeddingModelRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if model.Msg.GetFrom() != "mock" {
		t.Errorf("the model is from %q, want mock", model.Msg.GetFrom())
	}

	resp, err := srv.EmbedBatch(t.Context(), connect.NewRequest(&numenv1.EmbedBatchRequest{Texts: []string{"boat", "ferry"}}))
	if err != nil {
		t.Fatal(err)
	}
	vectors := resp.Msg.GetVectors()
	if len(vectors) != 2 {
		t.Fatalf("got %d vectors for 2 texts", len(vectors))
	}
	if got := len(vectors[0].GetValues()); got != int(model.Msg.GetDimensions()) {
		t.Errorf("got %d dimensions, want %d", got, model.Msg.GetDimensions())
	}
}

func TestIndexShareReservesOnePassForQuestions(t *testing.T) {
	for _, tc := range []struct{ passes, want int }{{1, 1}, {2, 1}, {4, 3}} {
		if got := getIndexShare(tc.passes); got != tc.want {
			t.Errorf("a model of %d passes gives the index %d, want %d", tc.passes, got, tc.want)
		}
	}
}
