package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

// slowModel is a server whose model does not finish opening until release is closed, as a model
// that is still downloading.
func slowModel(t *testing.T) (srv *embeddingServer, started chan struct{}, release chan struct{}) {
	t.Helper()
	srv = newEmbeddingServer()
	started, release = make(chan struct{}), make(chan struct{})
	var once sync.Once
	srv.open = func(ctx context.Context, _ engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
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

// within fails the test when call does not return in time.
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
		if _, err := srv.GetModelInfo(t.Context()); err != nil {
			t.Error(err)
		}
	})
	within(t, "Configure", func() { srv.updateConfig(map[string]string{"provider": "local"}) })
}

func TestACancelledBatchDoesNotStopTheOpening(t *testing.T) {
	srv, started, release := slowModel(t)

	ctx, cancel := context.WithCancel(t.Context())
	ended := make(chan error, 1)
	go func() { _, err := srv.EmbedTexts(ctx, []string{"text"}); ended <- err }()
	<-started
	cancel()
	if err := <-ended; err == nil {
		t.Fatal("a cancelled batch returned vectors")
	}

	close(release)
	vectors, err := srv.EmbedTexts(t.Context(), []string{"text"})
	if err != nil || len(vectors) != 1 {
		t.Fatalf("the next batch: %v, %v", vectors, err)
	}
}

func TestAChangeOfTheModelEndsTheOpeningOfTheOldOne(t *testing.T) {
	srv := newEmbeddingServer()
	srv.isMock = false
	names := make(chan string, 2)
	srv.open = func(ctx context.Context, info engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
		names <- info.Name
		if info.Name == "first/model" {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return engine.MockEmbedder{}, nil
	}
	t.Cleanup(func() { closeServer(t, srv) })

	srv.updateConfig(map[string]string{"provider": "local", "model": "first/model"})
	go func() { embedText(t.Context(), t, srv) }()
	if got := <-names; got != "first/model" {
		t.Fatalf("opened %q first", got)
	}

	srv.updateConfig(map[string]string{"provider": "local", "model": "second/model"})
	vectors, err := srv.EmbedTexts(t.Context(), []string{"text"})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("the batch after the change: %v", err)
	}
	if err == nil && len(vectors) != 1 {
		t.Fatalf("vectors: %v", vectors)
	}
}

func closeServer(t *testing.T, srv *embeddingServer) {
	t.Helper()
	if err := srv.Close(); err != nil {
		t.Error(err)
	}
}

// embedText embeds one text and ignores the outcome: the caller is a batch that may be cancelled.
func embedText(ctx context.Context, t *testing.T, srv *embeddingServer) {
	t.Helper()
	if vectors, err := srv.EmbedTexts(ctx, []string{"text"}); err == nil && len(vectors) != 1 {
		t.Errorf("vectors: %v", vectors)
	}
}
