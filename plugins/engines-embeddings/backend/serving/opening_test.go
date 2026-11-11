package serving

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
)

func slowModel(t *testing.T) (srv *Server, started chan struct{}, release chan struct{}) {
	t.Helper()
	srv = newEmbeddingServer()
	started, release = make(chan struct{}), make(chan struct{})
	var once sync.Once
	srv.open = func(ctx context.Context, _ engine.ModelInfo, _ engine.LocalModel) (loading.Embedder, error) {
		once.Do(func() { close(started) })
		select {
		case <-release:
			return engine.MockEmbedder{}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	t.Cleanup(func() { closeServer(t, srv) })
	return srv, started, release
}

func within(t *testing.T, what string, call func()) {
	t.Helper()
	finished := make(chan struct{})
	go func() { call(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s waited for the model to open", what)
	}
}

func TestACallOfTheHostDoesNotWaitForTheModelToOpen(t *testing.T) {
	srv, started, release := slowModel(t)
	defer close(release)

	go func() { embedText(t.Context(), t, srv) }()
	<-started

	within(t, "GetModelInfo", func() {
		if _, err := srv.GetModelInfo(); err != nil {
			t.Error(err)
		}
	})
	within(t, "Configure", func() { srv.updateConfig(map[string]string{"provider": "local"}, nil) })
}

func TestACancelledBatchDoesNotStopTheOpening(t *testing.T) {
	srv, started, release := slowModel(t)

	ctx, cancel := context.WithCancel(t.Context())
	ended := make(chan error, 1)
	go func() { _, err := srv.EmbedTexts(ctx, engine.RoleDocument, []string{"text"}); ended <- err }()
	<-started
	cancel()
	if err := <-ended; err == nil {
		t.Fatal("a canceled batch returned vectors")
	}

	close(release)
	vectors, err := srv.EmbedTexts(t.Context(), engine.RoleDocument, []string{"text"})
	if err != nil || len(vectors) != 1 {
		t.Fatalf("the next batch: %v, %v", vectors, err)
	}
}

func TestAChangeOfTheModelEndsTheOpeningOfTheOldOne(t *testing.T) {
	srv := newEmbeddingServer()
	srv.isMock = false
	names := make(chan string, 2)
	srv.open = func(ctx context.Context, info engine.ModelInfo, _ engine.LocalModel) (loading.Embedder, error) {
		names <- info.Name
		if info.Name == "Xenova/bge-m3" {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return engine.MockEmbedder{}, nil
	}
	t.Cleanup(func() { closeServer(t, srv) })

	srv.updateConfig(map[string]string{"provider": "local", "model": "Xenova/bge-m3"}, nil)
	go func() { embedText(t.Context(), t, srv) }()
	if got := <-names; got != "Xenova/bge-m3" {
		t.Fatalf("opened %q first", got)
	}

	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel}, nil)
	vectors, err := srv.EmbedTexts(t.Context(), engine.RoleDocument, []string{"text"})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("the batch after the change: %v", err)
	}
	if err == nil && len(vectors) != 1 {
		t.Fatalf("vectors: %v", vectors)
	}
}

func closeServer(t *testing.T, srv *Server) {
	t.Helper()
	if err := srv.Close(); err != nil {
		t.Error(err)
	}
}

func embedText(ctx context.Context, t *testing.T, srv *Server) {
	t.Helper()
	if vectors, err := srv.EmbedTexts(ctx, engine.RoleDocument, []string{"text"}); err == nil && len(vectors) != 1 {
		t.Errorf("vectors: %v", vectors)
	}
}

type stubEmbedder struct {
	engine.MockEmbedder
	concurrency int
	closeErr    error
	onClose     func()
}

func (s stubEmbedder) Concurrency() int { return s.concurrency }

func (s stubEmbedder) Close() error {
	if s.onClose != nil {
		s.onClose()
	}
	return s.closeErr
}

func TestStartOpeningUsesLocalInfoForNonMockAndModelInfoForMock(t *testing.T) {
	srv := newEmbeddingServer()
	srv.isMock = false
	srv.modelInfo = engine.ModelInfo{Name: "remote-override"}
	srv.localInfo = engine.ModelInfo{Name: "local-actual"}
	var gotInfo engine.ModelInfo
	srv.open = func(_ context.Context, info engine.ModelInfo, _ engine.LocalModel) (loading.Embedder, error) {
		gotInfo = info
		return stubEmbedder{concurrency: 6}, nil
	}
	emb, err := srv.getEmbedder(t.Context())
	if err != nil || emb == nil {
		t.Fatalf("getEmbedder = %v, %v", emb, err)
	}
	if gotInfo.Name != "local-actual" {
		t.Fatalf("non-mock opened %q, want local-actual", gotInfo.Name)
	}
	if srv.modelInfo.Concurrency != 6 {
		t.Fatalf("modelInfo.Concurrency = %d, want 6", srv.modelInfo.Concurrency)
	}

	mockSrv := newEmbeddingServer()
	mockSrv.isMock = true
	mockSrv.modelInfo = engine.ModelInfo{Name: "mock-model-info"}
	mockSrv.localInfo = engine.ModelInfo{Name: "unused-local"}
	mockSrv.open = func(_ context.Context, info engine.ModelInfo, _ engine.LocalModel) (loading.Embedder, error) {
		gotInfo = info
		return stubEmbedder{concurrency: 4}, nil
	}
	if _, err := mockSrv.getEmbedder(t.Context()); err != nil {
		t.Fatalf("mock getEmbedder: %v", err)
	}
	if gotInfo.Name != "mock-model-info" {
		t.Fatalf("mock opened %q, want mock-model-info", gotInfo.Name)
	}
}

func TestFinishOpeningFailure(t *testing.T) {
	srv := newEmbeddingServer()
	wantErr := errors.New("boom")
	srv.open = func(_ context.Context, _ engine.ModelInfo, _ engine.LocalModel) (loading.Embedder, error) {
		return stubEmbedder{}, wantErr
	}
	emb, err := srv.getEmbedder(t.Context())
	if !errors.Is(err, wantErr) || emb != nil {
		t.Fatalf("getEmbedder = %v, %v, want nil and %v", emb, err, wantErr)
	}
	if srv.loader.IsLoaded() {
		t.Fatal("the loader holds a model after the open failed")
	}
}

func TestFinishOpeningSupersededClosesRetiredModelAndRetries(t *testing.T) {
	srv := newEmbeddingServer()
	closedRetired := make(chan struct{})
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	calls := 0
	srv.open = func(_ context.Context, _ engine.ModelInfo, _ engine.LocalModel) (loading.Embedder, error) {
		calls++
		if calls == 1 {
			close(firstEntered)
			<-releaseFirst
			return stubEmbedder{
				concurrency: 3,
				onClose:     func() { close(closedRetired) },
			}, nil
		}
		return stubEmbedder{concurrency: 5}, nil
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := srv.getEmbedder(t.Context())
		errCh <- err
	}()
	<-firstEntered

	srv.mu.Lock()
	srv.loader.ResetLocked()
	srv.mu.Unlock()
	close(releaseFirst)

	if err := <-errCh; err != nil {
		t.Fatalf("getEmbedder after supersede = %v", err)
	}
	<-closedRetired
	if srv.modelInfo.Concurrency != 5 {
		t.Fatalf("Concurrency = %d, want 5 from second opening", srv.modelInfo.Concurrency)
	}
}
