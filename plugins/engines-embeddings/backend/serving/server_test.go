package serving

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/mockembedder"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/model"
)

func TestEmbeddingServerGetModel(t *testing.T) {
	srv := newEmbeddingServer()
	msg, err := srv.GetModelInfo()
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
	msg, err := srv.GetModelInfo()
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
		"model": "text-embedding-3-small",
	}, nil)
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
		wantDims int
	}{
		{provider: "local", model: "Xenova/bge-m3", wantDims: 1024},
		{provider: "local", model: "Xenova/multilingual-e5-small", wantDims: 384},
		{provider: "local", model: "openai/text-embedding-3-small", wantDims: 384},
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
		}, nil)
		info, err := emb.GetModelInfo()
		if err != nil {
			t.Fatalf("%s/%s: %v", tc.provider, tc.model, err)
		}
		if info.Dimensions != tc.wantDims {
			t.Errorf("%s/%s: got %d dimensions, want %d", tc.provider, tc.model, info.Dimensions, tc.wantDims)
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
	_, err := srv.GetModelInfo()
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
	t.Setenv("NUMEN_MOCK_ML", "true")
	srv := newEmbeddingServer()

	served, err := srv.GetModelInfo()
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

func TestRemoteProviderWithoutKeyDoesNotOpenLocalEmbedder(t *testing.T) {
	srv := newEmbeddingServer()
	hasOpenCall := false
	srv.open = func(_ context.Context, _ model.ModelInfo, _ model.LocalModel) (loading.Embedder, error) {
		hasOpenCall = true
		return mockembedder.Embedder{}, nil
	}

	srv.updateConfig(map[string]string{
		"provider": "openrouter",
		"model":    "openai/text-embedding-3-small",
	}, nil)

	_, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hello"})
	if !isPrecondition(err) {
		t.Fatalf("err = %v, want FailedPrecondition", err)
	}
	if hasOpenCall {
		t.Fatal("local ONNX embedder was opened while remote provider was selected without an API key")
	}
}

func TestLocalProviderWithRemoteModelNameUsesLocalDefault(t *testing.T) {
	srv := newEmbeddingServer()
	var activeInfo model.ModelInfo
	var activeLocal model.LocalModel
	srv.open = func(_ context.Context, info model.ModelInfo, local model.LocalModel) (loading.Embedder, error) {
		activeInfo = info
		activeLocal = local
		return mockembedder.Embedder{}, nil
	}

	srv.updateConfig(map[string]string{
		"provider": "local",
		"model":    "openai/text-embedding-3-small",
	}, nil)

	info, err := srv.GetModelInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != model.DefaultModel || info.Dimensions != 384 {
		t.Fatalf("GetModelInfo = %s@%d, want the default model at 384", info.Name, info.Dimensions)
	}

	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	if activeInfo.Name != model.DefaultModel || activeInfo.Dimensions != 384 || activeLocal.Name != model.DefaultModel {
		t.Fatalf("opened %+v / %+v, want the default model at 384", activeInfo, activeLocal)
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
		{map[string]string{"provider": "local", "model": "Xenova/multilingual-e5-small"}, "Xenova/multilingual-e5-small", 384, 512, "mean", 0.81, "local e5"},
		{map[string]string{"provider": "openai", "model": "text-embedding-ada-002"}, "text-embedding-ada-002", 1536, 8191, "cls", 0.75, "remote ada"},
		{map[string]string{"provider": "openrouter", "model": "qwen/qwen3-embedding-8b"}, "qwen/qwen3-embedding-8b", 4096, 32768, "cls", 0, "remote qwen"},
		{map[string]string{"provider": "local", "model": "Xenova/bge-m3"}, "Xenova/bge-m3", 1024, 512, "cls", 0.30, "local bge-m3"},
		{map[string]string{"provider": "local", "model": "Xenova/multilingual-e5-small"}, "Xenova/multilingual-e5-small", 384, 512, "mean", 0.81, "local e5 again"},
	}
	for _, step := range steps {
		srv.updateConfig(step.values, nil)
		info, err := srv.GetModelInfo()
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
	srv.open = func(_ context.Context, info model.ModelInfo, _ model.LocalModel) (loading.Embedder, error) {
		openedInfo = info
		return mockembedder.Embedder{}, nil
	}
	srv.updateConfig(map[string]string{"provider": "local", "model": "Xenova/multilingual-e5-small"}, nil)
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	if openedInfo.Dimensions != 384 || openedInfo.MaxTokens != 512 || openedInfo.Pooling != "mean" {
		t.Fatalf("opened with %+v, want 384 dimensions, 512 tokens, mean pooling", openedInfo)
	}
}

func TestTheMockModelReportsNoFloorHint(t *testing.T) {
	t.Setenv("NUMEN_MOCK_ML", "true")
	srv := newEmbeddingServer()
	if srv.modelInfo.MinChunkLength != 40 {
		t.Fatalf("initial mock MinChunkLength = %d, want 40", srv.modelInfo.MinChunkLength)
	}
	srv.updateConfig(map[string]string{"provider": "local", "model": "Xenova/multilingual-e5-small"}, nil)
	info, err := srv.GetModelInfo()
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

func TestUpdateConfigDefaultRemoteModelsAndProviderSelection(t *testing.T) {
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"provider": "openrouter"}, nil)
	if srv.remoteModel != "openai/text-embedding-3-small" || srv.modelInfo.Name != "openai/text-embedding-3-small" {
		t.Fatalf("openrouter default remoteModel = %q / %q, want %q", srv.remoteModel, srv.modelInfo.Name, "openai/text-embedding-3-small")
	}

	srv2 := newEmbeddingServer()
	srv2.updateConfig(map[string]string{"provider": "openai"}, nil)
	if srv2.remoteModel != "text-embedding-3-small" || srv2.modelInfo.Name != "text-embedding-3-small" {
		t.Fatalf("openai default remoteModel = %q / %q, want %q", srv2.remoteModel, srv2.modelInfo.Name, "text-embedding-3-small")
	}

	srv3 := newEmbeddingServer()
	srv3.updateConfig(map[string]string{"provider": "local"}, nil)
	if srv3.remoteModel != "" {
		t.Fatalf("local provider set remoteModel = %q, want empty", srv3.remoteModel)
	}

	srv4 := newEmbeddingServer()
	srv4.updateConfig(map[string]string{"model": "Xenova/bge-m3"}, nil)
	if srv4.remoteModel != "" || srv4.modelInfo.Name != "Xenova/bge-m3" {
		t.Fatalf("empty provider with local model: remote=%q, name=%q", srv4.remoteModel, srv4.modelInfo.Name)
	}

	srv5 := newEmbeddingServer()
	srv5.updateConfig(map[string]string{"provider": "openai", "model": "text-embedding-3-large"}, nil)
	srv5.updateConfig(map[string]string{"provider": "openai", "model": ""}, nil)
	if srv5.remoteModel != "text-embedding-3-large" {
		t.Fatalf("empty model overwrote remoteModel to %q", srv5.remoteModel)
	}
}

func TestRefreshModelInfoPreservesDimensionsEnvAndConcurrency(t *testing.T) {
	srv := newEmbeddingServer()
	srv.modelInfo.Concurrency = 7
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "777")
	srv.updateConfig(map[string]string{"provider": "openai", "model": "text-embedding-3-small"}, nil)
	if srv.modelInfo.Dimensions != 777 {
		t.Fatalf("remote Dimensions with env override = %d, want 777", srv.modelInfo.Dimensions)
	}
	if srv.modelInfo.Concurrency != 7 {
		t.Fatalf("remote Concurrency = %d, want 7", srv.modelInfo.Concurrency)
	}

	srv.updateConfig(map[string]string{"provider": "local", "model": model.DefaultModel}, nil)
	if srv.modelInfo.Concurrency != 7 {
		t.Fatalf("local Concurrency after refresh = %d, want 7", srv.modelInfo.Concurrency)
	}

	srv2 := newEmbeddingServer()
	srv2.provider = "custom"
	srv2.remoteModel = ""
	srv2.refreshModelInfoLocked()
	if srv2.modelInfo.Name != model.DefaultModel {
		t.Fatalf("refreshModelInfoLocked with empty remoteModel = %q, want %q", srv2.modelInfo.Name, model.DefaultModel)
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
	srv3.open = func(context.Context, model.ModelInfo, model.LocalModel) (loading.Embedder, error) {
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
	info, err := srv.GetModelInfo()
	if err != nil || info.Concurrency != 3 {
		t.Fatalf("Concurrency(0) = %d, %v, want 3", info.Concurrency, err)
	}

	srv.modelInfo.Concurrency = -1
	info, err = srv.GetModelInfo()
	if err != nil || info.Concurrency != 3 {
		t.Fatalf("Concurrency(-1) = %d, %v, want 3", info.Concurrency, err)
	}

	srv.modelInfo.Concurrency = 1
	info, err = srv.GetModelInfo()
	if err != nil || info.Concurrency != 1 {
		t.Fatalf("Concurrency(1) = %d, %v, want 1", info.Concurrency, err)
	}

	srv.modelInfo.Concurrency = 3
	info, err = srv.GetModelInfo()
	if err != nil || info.Concurrency != 2 {
		t.Fatalf("Concurrency(3) = %d, %v, want 2", info.Concurrency, err)
	}

	srv.localErr = errors.New("broken local")
	srv.isMock = true
	if err := srv.getLocalErrLocked(); err != nil {
		t.Fatalf("mock getLocalErrLocked = %v, want nil", err)
	}
	srv.isMock = false
	srv.provider = "openai"
	if err := srv.getLocalErrLocked(); err != nil {
		t.Fatalf("remote getLocalErrLocked = %v, want nil", err)
	}
	srv.provider = "local"
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

	srv.updateConfig(map[string]string{"provider": "custom", "baseUrl": ""}, keysOf("my-key"))
	_, err = srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hi"})
	if !isPrecondition(err) || !strings.Contains(err.Error(), "URL is not configured") {
		t.Fatalf("missing URL err = %v", err)
	}

	srv.updateConfig(map[string]string{"provider": "custom", "baseUrl": "https://example.com"}, keysOf())
	_, err = srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hi"})
	if !isPrecondition(err) || !strings.Contains(err.Error(), "API key is not configured") {
		t.Fatalf("missing key err = %v", err)
	}

	srv2 := newEmbeddingServer()
	calls := 0
	srv2.open = func(_ context.Context, _ model.ModelInfo, _ model.LocalModel) (loading.Embedder, error) {
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
	srv3.open = func(_ context.Context, _ model.ModelInfo, _ model.LocalModel) (loading.Embedder, error) {
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
	srv4.open = func(_ context.Context, _ model.ModelInfo, _ model.LocalModel) (loading.Embedder, error) {
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

func TestIsRemoteProviderAndGetService(t *testing.T) {
	srv := newEmbeddingServer()
	srv.provider = "openai"
	srv.apiKey = "oa-key"
	srv.isMock = true
	if srv.isRemoteProvider() {
		t.Fatal("mock server reported isRemoteProvider = true")
	}
	if k, u := srv.getService(); k != "" || u != "" {
		t.Fatalf("mock getService = %q, %q", k, u)
	}

	srv.isMock = false
	if !srv.isRemoteProvider() {
		t.Fatal("openai server reported isRemoteProvider = false")
	}
	if k, u := srv.getService(); k != "oa-key" || u != "https://api.openai.com/v1" {
		t.Fatalf("openai getService = %q, %q", k, u)
	}

	srv.provider = "openrouter"
	srv.apiKey = "or-key"
	if k, u := srv.getService(); k != "or-key" || u != "https://openrouter.ai/api/v1" {
		t.Fatalf("openrouter getService = %q, %q", k, u)
	}

	srv.provider = "custom"
	srv.apiKey = "cust-key"
	srv.baseURL = ""
	t.Setenv("NUMEN_EMBEDDING_URL", "https://env.example.com/v1")
	if k, u := srv.getService(); k != "cust-key" || u != "https://env.example.com/v1" {
		t.Fatalf("env getService = %q, %q", k, u)
	}

	srv.baseURL = "https://configured.example.com/v1"
	if k, u := srv.getService(); k != "cust-key" || u != "https://configured.example.com/v1" {
		t.Fatalf("configured getService = %q, %q", k, u)
	}
}

func TestEmbedRemoteTexts(t *testing.T) {
	var gotContentType string
	mode := "ok"
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		switch mode {
		case "status500":
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte("server exploded")); err != nil {
				t.Error(err)
			}
		case "badjson":
			if _, err := w.Write([]byte("{invalid json")); err != nil {
				t.Error(err)
			}
		case "emptydata":
			encodeJSON(t, w, map[string]any{"data": []any{}})
		case "emptyvec":
			encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{}, "index": 0}}})
		case "singlefloat":
			encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{0.42}, "index": 0}}})
		default:
			encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{0.1, 0.2, 0.3, 0.4, 0.5}, "index": 0}}})
		}
	}))
	defer remote.Close()

	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{
		"provider": "custom",
		"baseUrl":  remote.URL,
		"model":    "remote-test",
	}, keysOf("secret"))

	vecs, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hello"})
	if err != nil || len(vecs) != 1 || len(vecs[0]) != 5 {
		t.Fatalf("EmbedTexts remote = %v, %v", vecs, err)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotContentType)
	}
	if srv.modelInfo.Dimensions != 5 {
		t.Fatalf("Dimensions updated to %d, want 5", srv.modelInfo.Dimensions)
	}

	mode = "status500"
	_, err = srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"hello"})
	if isPrecondition(err) || !strings.Contains(err.Error(), "service text 0: status 500: server exploded") {
		t.Fatalf("status500 err = %v", err)
	}
}
