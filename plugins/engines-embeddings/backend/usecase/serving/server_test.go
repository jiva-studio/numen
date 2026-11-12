package serving

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/mockembedder"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/usecase/loading"
)

func TestEmbeddingServerGetModel(t *testing.T) {
	srv := newEmbeddingServer()
	msg, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Name != "Xenova/multilingual-e5-small" {
		t.Errorf("expected the default model, got %s", msg.Name)
	}
	if msg.Dimensions != 384 {
		t.Errorf("expected 384 dimensions, got %d", msg.Dimensions)
	}
	if msg.MaxTokens != 512 {
		t.Errorf("expected 512 max tokens, got %d", msg.MaxTokens)
	}
	if msg.Pooling != "mean" {
		t.Errorf("expected mean pooling, got %s", msg.Pooling)
	}
}

func TestEmbeddingServerEnvOverrides(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "Xenova/bge-m3")
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "512")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "1024")
	t.Setenv("NUMEN_EMBEDDING_POOLING", "mean")

	srv := newEmbeddingServer()
	msg, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Name != "Xenova/bge-m3" {
		t.Errorf("expected Xenova/bge-m3, got %s", msg.Name)
	}
	if msg.Dimensions != 512 {
		t.Errorf("expected 512 dimensions, got %d", msg.Dimensions)
	}
	if msg.MaxTokens != 1024 {
		t.Errorf("expected 1024 max tokens, got %d", msg.MaxTokens)
	}
	if msg.Pooling != "mean" {
		t.Errorf("expected mean pooling, got %s", msg.Pooling)
	}
}

func TestLifecycleConfigure(t *testing.T) {
	emb := newEmbeddingServer()
	life := server.NewLifecycleServer(make(chan struct{}), server.WithLifecycleUpdateConfig(emb.updateConfig))

	success, err := life.ApplyConfig(t.Context(), map[string]string{
		"model": "Xenova/bge-m3",
	})
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}
	if !success {
		t.Error("configure success = false, want true")
	}
	if emb.modelInfo.Name != "Xenova/bge-m3" {
		t.Errorf("expected model Xenova/bge-m3, got %s", emb.modelInfo.Name)
	}
	if emb.modelInfo.Dimensions != 1024 {
		t.Errorf("expected 1024 dimensions for Xenova/bge-m3, got %d", emb.modelInfo.Dimensions)
	}
}

func TestModelDimensionResolutionAcrossLocalModels(t *testing.T) {
	emb := newEmbeddingServer()
	cases := []struct {
		model    string
		wantDims int
	}{
		{model: "Xenova/bge-m3", wantDims: 1024},
		{model: "Xenova/multilingual-e5-small", wantDims: 384},
	}

	for _, tc := range cases {
		emb.updateConfig(map[string]string{"source": "local", "model": tc.model})
		info, err := emb.GetModelInfo(t.Context())
		if err != nil {
			t.Fatalf("%s: %v", tc.model, err)
		}
		if info.Dimensions != tc.wantDims {
			t.Errorf("%s: got %d dimensions, want %d", tc.model, info.Dimensions, tc.wantDims)
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

func TestEmbeddingServerWithoutAProfileFailsAndNamesWhereToAddOne(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "owner/model")
	srv := newEmbeddingServer()
	_, err := srv.GetModelInfo(t.Context())
	if !isPrecondition(err) || !strings.Contains(err.Error(), "owner/model") || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("GetModelInfo error = %v, want FailedPrecondition that names the model and the profile", err)
	}
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"text"}); !isPrecondition(err) {
		t.Errorf("EmbedTexts error = %v, want FailedPrecondition", err)
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
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	srv := newEmbeddingServer()

	served, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if served.From != "mock" {
		t.Errorf("the model is from %q, want mock", served.From)
	}

	vectors, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"boat", "ferry"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 2 {
		t.Fatalf("got %d vectors for 2 texts", len(vectors))
	}
	if got := len(vectors[0]); got != int(served.Dimensions) {
		t.Errorf("got %d dimensions, want %d", got, served.Dimensions)
	}
}

func TestUpdateConfigAppliesProfileOfSelectedModel(t *testing.T) {
	srv := newEmbeddingServer()
	steps := []struct {
		values      map[string]string
		name        string
		dimensions  int
		maxTokens   int
		pooling     string
		floorHint   float64
		description string
	}{
		{map[string]string{"source": "local", "model": "Xenova/multilingual-e5-small"}, "Xenova/multilingual-e5-small", 384, 512, "mean", 0.81, "local e5"},
		{map[string]string{"source": "local", "model": "Xenova/bge-m3"}, "Xenova/bge-m3", 1024, 512, "cls", 0.30, "local bge-m3"},
		{map[string]string{"source": "local", "model": "Xenova/multilingual-e5-small"}, "Xenova/multilingual-e5-small", 384, 512, "mean", 0.81, "local e5 again"},
	}
	for _, step := range steps {
		srv.updateConfig(step.values)
		info, err := srv.GetModelInfo(t.Context())
		if err != nil {
			t.Fatalf("%s: %v", step.description, err)
		}
		if info.Name != step.name || info.Dimensions != step.dimensions || info.MaxTokens != step.maxTokens ||
			info.Pooling != step.pooling || info.FloorHint != step.floorHint {
			t.Errorf("%s: got %s/%d/%d/%s/%v, want %s/%d/%d/%s/%v", step.description,
				info.Name, info.Dimensions, info.MaxTokens, info.Pooling, info.FloorHint,
				step.name, step.dimensions, step.maxTokens, step.pooling, step.floorHint)
		}
	}
}

func TestTheEmbedderOpensWithTheProfileOfTheLocalModel(t *testing.T) {
	srv := newEmbeddingServer()
	var openedInfo model.ModelInfo
	srv.open = func(_ context.Context, info model.ModelInfo, _ model.LocalModel) (ports.Embedder, error) {
		openedInfo = info
		return mockembedder.Embedder{}, nil
	}
	srv.updateConfig(map[string]string{"source": "local", "model": "Xenova/multilingual-e5-small"})
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	if openedInfo.Dimensions != 384 || openedInfo.MaxTokens != 512 || openedInfo.Pooling != "mean" {
		t.Fatalf("opened with %+v, want 384 dimensions, 512 tokens, mean pooling", openedInfo)
	}
}

func TestTheMockModelReportsNoFloorHint(t *testing.T) {
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	srv := newEmbeddingServer()
	if srv.modelInfo.MinChunkLength != 40 {
		t.Fatalf("initial mock MinChunkLength = %d, want 40", srv.modelInfo.MinChunkLength)
	}
	srv.updateConfig(map[string]string{"source": "local", "model": "Xenova/multilingual-e5-small"})
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "mock-embedding" || info.FloorHint != 0 {
		t.Fatalf("mock model = %s with hint %v", info.Name, info.FloorHint)
	}
	if info.MinChunkLength != 40 {
		t.Fatalf("mock MinChunkLength = %d, want 40", info.MinChunkLength)
	}
}

func TestRefreshModelInfoPreservesDimensionsEnvAndConcurrency(t *testing.T) {
	srv := newEmbeddingServer()
	srv.modelInfo.Concurrency = 7
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "777")
	srv.updateConfig(map[string]string{"source": "local", "model": model.DefaultModel})
	if srv.modelInfo.Dimensions != 777 {
		t.Fatalf("Dimensions with env override = %d, want 777", srv.modelInfo.Dimensions)
	}
	if srv.modelInfo.Concurrency != 7 {
		t.Fatalf("Concurrency after refresh = %d, want 7", srv.modelInfo.Concurrency)
	}
}

func TestUpdateLocalModelHandlesMockProfilesErrAndClosingRetired(t *testing.T) {
	srv := newEmbeddingServer()
	srv.isMock = true
	srv.localName = "keep-mock"
	srv.updateLocalModelLocked("unknown-model")
	if srv.localName != "keep-mock" {
		t.Fatalf("mock updateLocalModelLocked changed localName to %q", srv.localName)
	}

	srv2 := newEmbeddingServer()
	wantErr := errors.New("profiles broken")
	srv2.profilesErr = wantErr
	srv2.updateLocalModelLocked(model.DefaultModel)
	if !errors.Is(srv2.localErr, wantErr) || srv2.localName != model.DefaultModel || srv2.localKey != "error" {
		t.Fatalf("profilesErr state: err=%v, name=%q, key=%q", srv2.localErr, srv2.localName, srv2.localKey)
	}

	srv3 := newEmbeddingServer()
	srv3.updateLocalModelLocked("missing/model")
	if srv3.localErr == nil || srv3.localName != "missing/model" || srv3.localKey != "error" {
		t.Fatalf("missing model state: err=%v, name=%q, key=%q", srv3.localErr, srv3.localName, srv3.localKey)
	}

	closed := make(chan struct{})
	srv3.open = func(context.Context, model.ModelInfo, model.LocalModel) (ports.Embedder, error) {
		return stubEmbedder{onClose: func() { close(closed) }}, nil
	}
	if _, err := srv3.getEmbedder(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv3.updateLocalModelLocked(model.DefaultModel)
	<-closed
	if srv3.loader.IsLoaded() || srv3.localKey == "error" {
		t.Fatalf("updateLocalModelLocked did not close retired embedder or update key: %q", srv3.localKey)
	}
}

func TestGetModelInfoConcurrencyAndGetLocalErrLocked(t *testing.T) {
	srv := newEmbeddingServer()
	srv.modelInfo.Concurrency = 0
	info, err := srv.GetModelInfo(t.Context())
	if err != nil || info.Concurrency != 3 {
		t.Fatalf("Concurrency(0) = %d, %v, want 3", info.Concurrency, err)
	}

	srv.modelInfo.Concurrency = -1
	info, err = srv.GetModelInfo(t.Context())
	if err != nil || info.Concurrency != 3 {
		t.Fatalf("Concurrency(-1) = %d, %v, want 3", info.Concurrency, err)
	}

	srv.modelInfo.Concurrency = 1
	info, err = srv.GetModelInfo(t.Context())
	if err != nil || info.Concurrency != 1 {
		t.Fatalf("Concurrency(1) = %d, %v, want 1", info.Concurrency, err)
	}

	srv.modelInfo.Concurrency = 3
	info, err = srv.GetModelInfo(t.Context())
	if err != nil || info.Concurrency != 2 {
		t.Fatalf("Concurrency(3) = %d, %v, want 2", info.Concurrency, err)
	}

	srv.localErr = errors.New("broken local")
	srv.isMock = true
	if err := srv.getLocalErrLocked(); err != nil {
		t.Fatalf("mock getLocalErrLocked = %v, want nil", err)
	}
	srv.isMock = false
	srv.source = SourceConnection
	if err := srv.getLocalErrLocked(); err != nil {
		t.Fatalf("connection getLocalErrLocked = %v, want nil", err)
	}
	srv.source = SourceLocal
	if err := srv.getLocalErrLocked(); err == nil {
		t.Fatal("local getLocalErrLocked = nil, want error")
	}
}

type funcEmbedder struct {
	embed func(ctx context.Context, texts []string) ([][]float32, error)
}

func (f funcEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return f.embed(ctx, texts)
}

func (f funcEmbedder) Concurrency() int { return 2 }

func (f funcEmbedder) Close() error { return nil }

func TestEmbedTextsBranchesAndRetriesOnClosed(t *testing.T) {
	srv := newEmbeddingServer()
	vecs, err := srv.EmbedTexts(t.Context(), model.RoleDocument, nil)
	if err != nil || vecs != nil {
		t.Fatalf("empty EmbedTexts = %v, %v", vecs, err)
	}

	srv2 := newEmbeddingServer()
	calls := 0
	srv2.open = func(_ context.Context, _ model.ModelInfo, _ model.LocalModel) (ports.Embedder, error) {
		return funcEmbedder{
			embed: func(_ context.Context, _ []string) ([][]float32, error) {
				calls++
				if calls == 1 {
					srv2.mu.Lock()
					srv2.loader.ResetLocked()
					srv2.mu.Unlock()
					return nil, loading.ErrClosed
				}
				return [][]float32{{1, 2, 3}}, nil
			},
		}, nil
	}
	vecs, err = srv2.EmbedTexts(t.Context(), model.RoleDocument, []string{"hi"})
	if err != nil || len(vecs) != 1 || calls != 2 {
		t.Fatalf("retry on ErrClosed = %v, %v, calls=%d", vecs, err, calls)
	}

	srv3 := newEmbeddingServer()
	srv3.open = func(_ context.Context, _ model.ModelInfo, _ model.LocalModel) (ports.Embedder, error) {
		return funcEmbedder{
			embed: func(_ context.Context, _ []string) ([][]float32, error) {
				return nil, errors.New("onnx crash")
			},
		}, nil
	}
	_, err = srv3.EmbedTexts(t.Context(), model.RoleDocument, []string{"hi"})
	if isPrecondition(err) || !strings.Contains(err.Error(), "embed batch: onnx crash") {
		t.Fatalf("embed error = %v", err)
	}

	srv4 := newEmbeddingServer()
	srv4.open = func(_ context.Context, _ model.ModelInfo, _ model.LocalModel) (ports.Embedder, error) {
		return funcEmbedder{
			embed: func(_ context.Context, _ []string) ([][]float32, error) {
				return [][]float32{{1}}, nil
			},
		}, nil
	}
	_, err = srv4.EmbedTexts(t.Context(), model.RoleDocument, []string{"one", "two"})
	if isPrecondition(err) || !strings.Contains(err.Error(), "expected 2 vectors, got 1") {
		t.Fatalf("vector count mismatch err = %v", err)
	}
}
