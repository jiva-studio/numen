package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

func TestEmbeddingServerGetModel(t *testing.T) {
	srv := newEmbeddingServer()
	msg, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	msg, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	success, err := life.ApplyConfig(t.Context(), map[string]string{
		"model": "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}
	if !success {
		t.Error("configure success = false, want true")
	}
	if emb.modelInfo.Name != "text-embedding-3-small" {
		t.Errorf("expected model text-embedding-3-small, got %s", emb.modelInfo.Name)
	}
	if emb.modelInfo.Dimensions != 1536 {
		t.Errorf("expected 1536 dimensions for text-embedding-3-small, got %d", emb.modelInfo.Dimensions)
	}
}

func TestModelDimensionResolutionAcrossProviders(t *testing.T) {
	emb := newEmbeddingServer()
	cases := []struct {
		provider string
		model    string
		wantDims int32
	}{
		{provider: "local", model: "Xenova/bge-m3", wantDims: 1024},
		{provider: "local", model: "Xenova/multilingual-e5-small", wantDims: 384},
		{provider: "local", model: "openai/text-embedding-3-small", wantDims: 1024},
		{provider: "openrouter", model: "openai/text-embedding-3-small", wantDims: 1536},
		{provider: "openrouter", model: "Xenova/multilingual-e5-small", wantDims: 1536},
		{provider: "openrouter", model: "openai/text-embedding-3-large", wantDims: 3072},
		{provider: "openrouter", model: "baai/bge-m3", wantDims: 1024},
		{provider: "openrouter", model: "qwen/qwen3-embedding-8b", wantDims: 4096},
		{provider: "openai", model: "text-embedding-3-large", wantDims: 3072},
	}

	for _, tc := range cases {
		emb.updateConfig(map[string]string{
			"provider": tc.provider,
			"model":    tc.model,
		})
		info, err := emb.GetModelInfo(t.Context())
		if err != nil {
			t.Fatalf("%s/%s: %v", tc.provider, tc.model, err)
		}
		if info.GetDimensions() != tc.wantDims {
			t.Errorf("%s/%s: got %d dimensions, want %d", tc.provider, tc.model, info.GetDimensions(), tc.wantDims)
		}
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

	model, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if model.GetFrom() != "mock" {
		t.Errorf("the model is from %q, want mock", model.GetFrom())
	}

	vectors, err := srv.EmbedTexts(t.Context(), []string{"boat", "ferry"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 2 {
		t.Fatalf("got %d vectors for 2 texts", len(vectors))
	}
	if got := len(vectors[0]); got != int(model.GetDimensions()) {
		t.Errorf("got %d dimensions, want %d", got, model.GetDimensions())
	}
}

func TestIndexShareReservesOnePassForQuestions(t *testing.T) {
	for _, tc := range []struct{ passes, want int }{{1, 1}, {2, 1}, {4, 3}} {
		if got := getIndexShare(tc.passes); got != tc.want {
			t.Errorf("a model of %d passes gives the index %d, want %d", tc.passes, got, tc.want)
		}
	}
}

func TestRemoteProviderWithoutKeyDoesNotOpenLocalEmbedder(t *testing.T) {
	srv := newEmbeddingServer()
	hasOpenCall := false
	srv.open = func(_ context.Context, _ engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
		hasOpenCall = true
		return engine.MockEmbedder{}, nil
	}

	srv.updateConfig(map[string]string{
		"provider": "openrouter",
		"model":    "openai/text-embedding-3-small",
	})

	_, err := srv.EmbedTexts(t.Context(), []string{"hello"})
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("err = %v, want FailedPrecondition", err)
	}
	if hasOpenCall {
		t.Fatal("local ONNX embedder was opened while remote provider was selected without an API key")
	}
}

func TestLocalProviderWithRemoteModelNameUsesLocalDefault(t *testing.T) {
	srv := newEmbeddingServer()
	var activeInfo engine.ModelInfo
	var activeLocal engine.LocalModel
	srv.open = func(_ context.Context, info engine.ModelInfo, local engine.LocalModel) (textEmbedder, error) {
		activeInfo = info
		activeLocal = local
		return engine.MockEmbedder{}, nil
	}

	srv.updateConfig(map[string]string{
		"provider": "local",
		"model":    "openai/text-embedding-3-small",
	})

	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.GetName() != "bge-m3" || info.GetDimensions() != 1024 {
		t.Fatalf("GetModelInfo = %s@%d, want bge-m3@1024", info.GetName(), info.GetDimensions())
	}

	if _, err := srv.EmbedTexts(t.Context(), []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	if activeInfo.Name != "bge-m3" || activeInfo.Dimensions != 1024 || activeLocal.Name != "Xenova/bge-m3" {
		t.Fatalf("opened %+v / %+v, want bge-m3@1024 on Xenova/bge-m3", activeInfo, activeLocal)
	}
}
