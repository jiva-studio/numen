package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

func TestEmbeddingServerGetModel(t *testing.T) {
	srv := newEmbeddingServer()
	msg, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.GetName() != "Xenova/multilingual-e5-small" {
		t.Errorf("expected the default model, got %s", msg.GetName())
	}
	if msg.GetDimensions() != 384 {
		t.Errorf("expected 384 dimensions, got %d", msg.GetDimensions())
	}
	if msg.GetMaxTokens() != 512 {
		t.Errorf("expected 512 max tokens, got %d", msg.GetMaxTokens())
	}
	if msg.GetPooling() != "mean" {
		t.Errorf("expected mean pooling, got %s", msg.GetPooling())
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
	if msg.GetName() != "Xenova/bge-m3" {
		t.Errorf("expected Xenova/bge-m3, got %s", msg.GetName())
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
		wantDims int32
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
		if info.GetDimensions() != tc.wantDims {
			t.Errorf("%s: got %d dimensions, want %d", tc.model, info.GetDimensions(), tc.wantDims)
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
	if rpcerror.GetCode(err) != rpcerror.FailedPrecondition || !strings.Contains(err.Error(), "owner/model") || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("GetModelInfo error = %v, want FailedPrecondition that names the model and the profile", err)
	}
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"text"}); rpcerror.GetCode(err) != rpcerror.FailedPrecondition {
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
	t.Setenv(envMockML, "true")
	srv := newEmbeddingServer()

	model, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if model.GetFrom() != "mock" {
		t.Errorf("the model is from %q, want mock", model.GetFrom())
	}

	vectors, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"boat", "ferry"})
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

func TestUpdateConfigAppliesProfileOfSelectedModel(t *testing.T) {
	srv := newEmbeddingServer()
	steps := []struct {
		values      map[string]string
		name        string
		dimensions  int32
		maxTokens   int32
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
		if info.GetName() != step.name || info.GetDimensions() != step.dimensions || info.GetMaxTokens() != step.maxTokens ||
			info.GetPooling() != step.pooling || info.GetFloorHint() != step.floorHint {
			t.Errorf("%s: got %s/%d/%d/%s/%v, want %s/%d/%d/%s/%v", step.description,
				info.GetName(), info.GetDimensions(), info.GetMaxTokens(), info.GetPooling(), info.GetFloorHint(),
				step.name, step.dimensions, step.maxTokens, step.pooling, step.floorHint)
		}
	}
}

func TestTheEmbedderOpensWithTheProfileOfTheLocalModel(t *testing.T) {
	srv := newEmbeddingServer()
	var openedInfo engine.ModelInfo
	srv.open = func(_ context.Context, info engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
		openedInfo = info
		return engine.MockEmbedder{}, nil
	}
	srv.updateConfig(map[string]string{"source": "local", "model": "Xenova/multilingual-e5-small"})
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	if openedInfo.Dimensions != 384 || openedInfo.MaxTokens != 512 || openedInfo.Pooling != "mean" {
		t.Fatalf("opened with %+v, want 384 dimensions, 512 tokens, mean pooling", openedInfo)
	}
}

func TestTheMockModelReportsNoFloorHint(t *testing.T) {
	t.Setenv(envMockML, "true")
	srv := newEmbeddingServer()
	if srv.modelInfo.MinChunkLength != 40 {
		t.Fatalf("initial mock MinChunkLength = %d, want 40", srv.modelInfo.MinChunkLength)
	}
	srv.updateConfig(map[string]string{"source": "local", "model": "Xenova/multilingual-e5-small"})
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.GetName() != "mock-embedding" || info.GetFloorHint() != 0 {
		t.Fatalf("mock model = %s with hint %v", info.GetName(), info.GetFloorHint())
	}
	if info.GetMinChunkLength() != 40 {
		t.Fatalf("mock MinChunkLength = %d, want 40", info.GetMinChunkLength())
	}
}

func TestOpenMockModelHoldWaitAndContextCancel(t *testing.T) {
	t.Setenv(envMockML, "true")
	srv := newEmbeddingServer()
	if srv.profiles == nil {
		t.Fatal("mock server did not load profiles")
	}

	t.Setenv(envMockOpenHold, "0")
	if _, err := srv.openMockModel(t.Context(), engine.ModelInfo{}, engine.LocalModel{}); err != nil {
		t.Fatalf("openMockModel with 0ms hold failed: %v", err)
	}

	t.Setenv(envMockOpenHold, "bad")
	if _, err := srv.openMockModel(t.Context(), engine.ModelInfo{}, engine.LocalModel{}); err != nil {
		t.Fatalf("openMockModel with invalid hold failed: %v", err)
	}

	t.Setenv(envMockOpenHold, "1")
	ctx1, cancel1 := context.WithCancel(t.Context())
	cancel1()
	if _, err := srv.openMockModel(ctx1, engine.ModelInfo{}, engine.LocalModel{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("openMockModel with 1ms hold and canceled ctx = %v, want Canceled", err)
	}

	t.Setenv(envMockOpenHold, "200")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := srv.openMockModel(ctx, engine.ModelInfo{}, engine.LocalModel{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("openMockModel with canceled ctx = %v, want Canceled", err)
	}
}

func TestRefreshModelInfoPreservesDimensionsEnvAndConcurrency(t *testing.T) {
	srv := newEmbeddingServer()
	srv.modelInfo.Concurrency = 7
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "777")
	srv.updateConfig(map[string]string{"source": "local", "model": engine.DefaultModel})
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
	srv2.updateLocalModelLocked(engine.DefaultModel)
	if !errors.Is(srv2.localErr, wantErr) || srv2.localName != engine.DefaultModel || srv2.localKey != "error" {
		t.Fatalf("profilesErr state: err=%v, name=%q, key=%q", srv2.localErr, srv2.localName, srv2.localKey)
	}

	srv3 := newEmbeddingServer()
	srv3.updateLocalModelLocked("missing/model")
	if srv3.localErr == nil || srv3.localName != "missing/model" || srv3.localKey != "error" {
		t.Fatalf("missing model state: err=%v, name=%q, key=%q", srv3.localErr, srv3.localName, srv3.localKey)
	}

	closed := make(chan struct{})
	srv3.embedder = stubEmbedder{onClose: func() { close(closed) }}
	srv3.updateLocalModelLocked(engine.DefaultModel)
	<-closed
	if srv3.embedder != nil || srv3.localKey == "error" {
		t.Fatalf("updateLocalModelLocked did not close retired embedder or update key: %q", srv3.localKey)
	}
}

func TestGetProfilesDirAndEnvOverrides(t *testing.T) {
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", "/tmp/numen-data")
	if got := getProfilesDir(); got != "/tmp/numen-data/profiles" {
		t.Fatalf("getProfilesDir with NUMEN_PLUGIN_DATA_DIR = %q", got)
	}
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", "")
	if got := getProfilesDir(); got != "" {
		t.Fatalf("getProfilesDir without NUMEN_PLUGIN_DATA_DIR = %q, want empty", got)
	}

	var local engine.LocalModel
	t.Setenv("NUMEN_EMBEDDING_DIR", "/custom/dir")
	t.Setenv("NUMEN_ONNX_RUNTIME", "/custom/libonnx.so")
	t.Setenv("NUMEN_ONNX_THREADS", "6")
	applyModelOverrides(&engine.ModelInfo{}, &local)
	applyRuntimeEnv(&local)
	if local.Dir != "/custom/dir" || local.Runtime != "/custom/libonnx.so" || local.Threads != 6 {
		t.Fatalf("overrides = %+v", local)
	}

	local2 := engine.LocalModel{Threads: 4}
	t.Setenv("NUMEN_ONNX_RUNTIME", "")
	t.Setenv("NUMEN_ONNX_THREADS", "invalid")
	applyRuntimeEnv(&local2)
	if local2.Runtime != "" || local2.Threads != 4 {
		t.Fatalf("invalid threads changed local2 = %+v", local2)
	}
	t.Setenv("NUMEN_ONNX_THREADS", "0")
	applyRuntimeEnv(&local2)
	if local2.Threads != 4 {
		t.Fatalf("zero threads changed local2 = %+v", local2)
	}
	t.Setenv("NUMEN_ONNX_THREADS", "-2")
	applyRuntimeEnv(&local2)
	if local2.Threads != 4 {
		t.Fatalf("negative threads changed local2 = %+v", local2)
	}
	t.Setenv("NUMEN_ONNX_THREADS", "1")
	applyRuntimeEnv(&local2)
	if local2.Threads != 1 {
		t.Fatalf("1 thread = %d, want 1", local2.Threads)
	}
}

func TestParsePositiveIntAndApplyDimensionEnvBoundaries(t *testing.T) {
	if got := parsePositiveInt(""); got != 0 {
		t.Fatalf("parsePositiveInt empty = %d", got)
	}
	if got := parsePositiveInt("0"); got != 0 {
		t.Fatalf("parsePositiveInt 0 = %d", got)
	}
	if got := parsePositiveInt("-1"); got != 0 {
		t.Fatalf("parsePositiveInt -1 = %d", got)
	}
	if got := parsePositiveInt("1"); got != 1 {
		t.Fatalf("parsePositiveInt 1 = %d", got)
	}
	if got := parsePositiveInt("2147483647"); got != 2147483647 {
		t.Fatalf("parsePositiveInt MaxInt32 = %d", got)
	}
	if got := parsePositiveInt("2147483648"); got != 0 {
		t.Fatalf("parsePositiveInt MaxInt32+1 = %d", got)
	}

	info := engine.ModelInfo{Dimensions: 384, MaxTokens: 512}
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "1")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "1")
	applyDimensionEnv(&info)
	if info.Dimensions != 1 || info.MaxTokens != 1 {
		t.Fatalf("applyDimensionEnv with 1 = %+v", info)
	}
}

func TestGetModelInfoConcurrencyAndGetLocalErrLocked(t *testing.T) {
	srv := newEmbeddingServer()
	srv.modelInfo.Concurrency = 0
	info, err := srv.GetModelInfo(t.Context())
	if err != nil || info.GetConcurrency() != 3 {
		t.Fatalf("Concurrency(0) = %d, %v, want 3", info.GetConcurrency(), err)
	}

	srv.modelInfo.Concurrency = -1
	info, err = srv.GetModelInfo(t.Context())
	if err != nil || info.GetConcurrency() != 3 {
		t.Fatalf("Concurrency(-1) = %d, %v, want 3", info.GetConcurrency(), err)
	}

	srv.modelInfo.Concurrency = 1
	info, err = srv.GetModelInfo(t.Context())
	if err != nil || info.GetConcurrency() != 1 {
		t.Fatalf("Concurrency(1) = %d, %v, want 1", info.GetConcurrency(), err)
	}

	srv.modelInfo.Concurrency = 3
	info, err = srv.GetModelInfo(t.Context())
	if err != nil || info.GetConcurrency() != 2 {
		t.Fatalf("Concurrency(3) = %d, %v, want 2", info.GetConcurrency(), err)
	}

	srv.localErr = errors.New("broken local")
	srv.isMock = true
	if err := srv.getLocalErrLocked(); err != nil {
		t.Fatalf("mock getLocalErrLocked = %v, want nil", err)
	}
	srv.isMock = false
	srv.source = sourceConnection
	if err := srv.getLocalErrLocked(); err != nil {
		t.Fatalf("connection getLocalErrLocked = %v, want nil", err)
	}
	srv.source = sourceLocal
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
	vecs, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, nil)
	if err != nil || vecs != nil {
		t.Fatalf("empty EmbedTexts = %v, %v", vecs, err)
	}

	srv2 := newEmbeddingServer()
	calls := 0
	srv2.open = func(_ context.Context, _ engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
		return funcEmbedder{
			embed: func(_ context.Context, _ []string) ([][]float32, error) {
				calls++
				if calls == 1 {
					srv2.mu.Lock()
					srv2.embedder = nil
					srv2.mu.Unlock()
					return nil, engine.ErrClosed
				}
				return [][]float32{{1, 2, 3}}, nil
			},
		}, nil
	}
	vecs, err = srv2.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"hi"})
	if err != nil || len(vecs) != 1 || calls != 2 {
		t.Fatalf("retry on ErrClosed = %v, %v, calls=%d", vecs, err, calls)
	}

	srv3 := newEmbeddingServer()
	srv3.open = func(_ context.Context, _ engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
		return funcEmbedder{
			embed: func(_ context.Context, _ []string) ([][]float32, error) {
				return nil, errors.New("onnx crash")
			},
		}, nil
	}
	_, err = srv3.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"hi"})
	if rpcerror.GetCode(err) != rpcerror.Internal || !strings.Contains(err.Error(), "embed batch: onnx crash") {
		t.Fatalf("embed error = %v", err)
	}

	srv4 := newEmbeddingServer()
	srv4.open = func(_ context.Context, _ engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
		return funcEmbedder{
			embed: func(_ context.Context, _ []string) ([][]float32, error) {
				return [][]float32{{1}}, nil
			},
		}, nil
	}
	_, err = srv4.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"one", "two"})
	if rpcerror.GetCode(err) != rpcerror.Internal || !strings.Contains(err.Error(), "expected 2 vectors, got 1") {
		t.Fatalf("vector count mismatch err = %v", err)
	}
}

func TestMountRegistersCloseCallback(t *testing.T) {
	t.Setenv(envMockML, "true")
	s := server.New(server.WithToken("token"))
	if err := Mount(s); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("s.Close: %v", err)
	}
}
