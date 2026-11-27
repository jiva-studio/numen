package reranking

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/ports"
)

// loadedModel is an open scorer and the calls that use it.
type loadedModel struct {
	key    string
	scorer ports.Scorer
	calls  sync.WaitGroup
}

// modelLoad is a load in progress. err is set before done closes.
type modelLoad struct {
	key  string
	done chan struct{}
	err  error
}

// loader keeps one model loaded and runs one load at a time. A load runs past the call that started it,
// until it ends or the loader closes.
type loader struct {
	open   ports.OpenScorer
	report func(what string, err error)

	mu       sync.Mutex
	loaded   *loadedModel
	loading  *modelLoad
	isClosed bool
	stop     context.CancelFunc
}

func newLoader(open ports.OpenScorer, report func(what string, err error)) *loader {
	return &loader{open: open, report: report, mu: sync.Mutex{}, loaded: nil, loading: nil, isClosed: false, stop: nil}
}

func getKey(p profile.Profile) string {
	return strings.Join([]string{p.ID, p.OnnxFile, p.Revision, joinFiles(p.SHA256), joinFiles(p.Dependencies), fmt.Sprint(p.MaxTokens)}, "\x00")
}

func joinFiles(files map[string]string) string {
	entries := make([]string, 0, len(files))
	for _, name := range slices.Sorted(maps.Keys(files)) {
		entries = append(entries, name+"="+files[name])
	}
	return strings.Join(entries, ",")
}

func (l *loader) acquire(ctx context.Context, p profile.Profile) (*loadedModel, error) {
	// The caller passes the returned model to release.
	key := getKey(p)
	for {
		l.mu.Lock()
		if l.isClosed {
			l.mu.Unlock()
			return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
		}
		if l.loaded != nil && l.loaded.key == key {
			model := l.loaded
			model.calls.Add(1)
			l.mu.Unlock()
			return model, nil
		}
		load := l.loading
		if load == nil {
			load = l.startLoad(ctx, p, key)
		}
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-load.done:
		}
		if load.key == key && load.err != nil {
			return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
		}
	}
}

func (l *loader) release(model *loadedModel) {
	model.calls.Done()
}

func (l *loader) startLoad(ctx context.Context, p profile.Profile, key string) *modelLoad {
	// The caller holds l.mu.
	load := &modelLoad{key: key, done: make(chan struct{}), err: nil}
	previous := l.loaded
	l.loaded = nil
	l.loading = load
	loadCtx, stop := context.WithCancel(context.WithoutCancel(ctx))
	l.stop = stop
	go func() {
		defer stop()
		l.closeModel(previous)
		scorer, err := l.open(loadCtx, p)
		if err != nil {
			l.report(p.ID, err)
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		load.err = err
		if err == nil {
			l.loaded = &loadedModel{key: key, scorer: scorer, calls: sync.WaitGroup{}}
		}
		l.loading = nil
		close(load.done)
	}()
	return load
}

func (l *loader) closeModel(model *loadedModel) {
	if model == nil {
		return
	}
	model.calls.Wait()
	if err := model.scorer.Close(); err != nil {
		l.report("close the model", err)
	}
}

func (l *loader) close() error {
	l.mu.Lock()
	l.isClosed = true
	load := l.loading
	if l.stop != nil {
		l.stop()
	}
	l.mu.Unlock()
	if load != nil {
		<-load.done
	}
	l.mu.Lock()
	model := l.loaded
	l.loaded = nil
	l.mu.Unlock()
	if model == nil {
		return nil
	}
	model.calls.Wait()
	return model.scorer.Close()
}
