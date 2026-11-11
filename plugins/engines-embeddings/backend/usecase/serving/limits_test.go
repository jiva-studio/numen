package serving

import (
	"context"
	"runtime"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/mockembedder"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/usecase/loading"
)

// openRecord is what the server opened a model with, and the models it closed.
type openRecord struct {
	mu      sync.Mutex
	opened  []model.LocalModel
	nices   []int
	closing int
}

type recordedEmbedder struct {
	mockembedder.Embedder
	record *openRecord
}

func (e recordedEmbedder) Close() error {
	e.record.mu.Lock()
	defer e.record.mu.Unlock()
	e.record.closing++
	return nil
}

func newRecordedServer(t *testing.T) (*Server, *openRecord) {
	t.Helper()
	srv := newEmbeddingServer()
	record := &openRecord{}
	srv.open = func(_ context.Context, _ model.ModelInfo, local model.LocalModel) (loading.Embedder, error) {
		record.mu.Lock()
		defer record.mu.Unlock()
		record.opened = append(record.opened, local)
		return recordedEmbedder{record: record}, nil
	}
	srv.machine = niceRecorder{Process: process.Process{}, record: func(nice int) {
		record.mu.Lock()
		defer record.mu.Unlock()
		record.nices = append(record.nices, nice)
	}}
	return srv, record
}

func (r *openRecord) openedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.opened)
}

func TestTheModelOpensWithTheThreadsOfTheProfile(t *testing.T) {
	srv, record := newRecordedServer(t)
	srv.updateConfig(map[string]string{performance.ConfigKey: "eco"}, nil)
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if got := record.opened[0]; got.Threads != 1 || got.BatchTexts != limits.EcoBatchTexts {
		t.Errorf("opened with %d threads and batches of %d, want 1 and %d", got.Threads, got.BatchTexts, limits.EcoBatchTexts)
	}
	if last := record.nices[len(record.nices)-1]; last != 19 {
		t.Errorf("the niceness is %d, want 19", last)
	}
}

// The profile of a running plugin changes with the next batch: the model that holds the old
// threads closes, and the next batch opens one with the new threads.
func TestAChangeOfTheProfileReachesTheRunningModel(t *testing.T) {
	srv, record := newRecordedServer(t)
	srv.updateConfig(map[string]string{performance.ConfigKey: "maximum"}, nil)
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	first := record.opened[0]

	srv.updateConfig(map[string]string{performance.ConfigKey: "eco"}, nil)
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if record.openedCount() != 2 {
		t.Fatalf("%d models opened, want 2", record.openedCount())
	}
	second := record.opened[1]
	if second.Threads != 1 || second.BatchTexts != limits.EcoBatchTexts || first.BatchTexts != model.GetDefaultBatchTexts() {
		t.Errorf("opened %d threads and %d texts after %d texts, want 1 thread and %d texts", second.Threads, second.BatchTexts, first.BatchTexts, limits.EcoBatchTexts)
	}
	if record.nices[len(record.nices)-1] != 19 {
		t.Errorf("the niceness after the change is %v", record.nices)
	}
}

// The same profile in the next Configure call keeps the model.
func TestTheSameProfileKeepsTheRunningModel(t *testing.T) {
	srv, record := newRecordedServer(t)
	values := map[string]string{performance.ConfigKey: "eco"}
	srv.updateConfig(values, nil)
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(values, nil)
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if record.openedCount() != 1 {
		t.Errorf("%d models opened, want 1", record.openedCount())
	}
}

// A thread count the environment sets wins over the profile.
func TestTheThreadsOfTheEnvironmentWinOverTheProfile(t *testing.T) {
	t.Setenv("NUMEN_ONNX_THREADS", "3")
	srv, record := newRecordedServer(t)
	srv.updateConfig(map[string]string{performance.ConfigKey: "eco", "provider": "local"}, nil)
	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if got := record.opened[0].Threads; got != 3 {
		t.Errorf("opened with %d threads, want 3", got)
	}
}

func TestApplyProfileHandlesInitialDefaultMockAndOpeningRetirement(t *testing.T) {
	prevProcs := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(prevProcs) })

	srv, record := newRecordedServer(t)
	srv.updateConfig(map[string]string{performance.ConfigKey: "balanced"}, nil)
	if len(record.nices) != 1 || record.nices[0] != 10 {
		t.Fatalf("nices on first balanced config = %v, want [10]", record.nices)
	}

	srv.updateConfig(map[string]string{performance.ConfigKey: "balanced"}, nil)
	if len(record.nices) != 1 {
		t.Fatalf("nices on repeated balanced config = %v, want [10]", record.nices)
	}

	srv.updateConfig(map[string]string{performance.ConfigKey: "eco"}, nil)
	if len(record.nices) != 2 || record.nices[1] != 19 {
		t.Fatalf("nices on eco config = %v, want [10 19]", record.nices)
	}
	if got := runtime.GOMAXPROCS(0); got != 1 {
		t.Fatalf("GOMAXPROCS = %d, want 1", got)
	}

	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	srv.mu.Lock()
	srv.isProfileSet = false
	srv.applyProfileLocked(performance.Eco)
	isLoaded := srv.loader.IsLoaded()
	srv.mu.Unlock()
	if !isLoaded {
		t.Fatal("embedder was retired when limits did not change")
	}

	srv.mu.Lock()
	srv.loader.ResetLocked()
	srv.mu.Unlock()
	isOpening, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(release) })
	var once sync.Once
	srv.open = func(ctx context.Context, _ model.ModelInfo, _ model.LocalModel) (loading.Embedder, error) {
		isFirst := false
		once.Do(func() { isFirst = true; close(isOpening) })
		if isFirst {
			<-ctx.Done()
			close(canceled)
			return nil, ctx.Err()
		}
		<-release
		return recordedEmbedder{record: record}, nil
	}
	ended := make(chan error, 1)
	go func() { _, err := srv.getEmbedder(t.Context()); ended <- err }()
	<-isOpening

	srv.mu.Lock()
	srv.isProfileSet = false
	srv.applyProfileLocked(performance.Eco)
	isOpeningAfterSameLimits := srv.loader.IsOpening()
	srv.applyProfileLocked(performance.Maximum)
	isOpeningAfterNewLimits := srv.loader.IsOpening()
	srv.mu.Unlock()
	if !isOpeningAfterSameLimits {
		t.Fatal("opening was retired when limits did not change")
	}
	if isOpeningAfterNewLimits {
		t.Fatal("opening was not retired when limits changed")
	}
	<-canceled

	runtime.GOMAXPROCS(4)
	srv.mu.Lock()
	srv.isMock = true
	srv.applyProfileLocked(performance.Eco)
	srv.mu.Unlock()
	if got := runtime.GOMAXPROCS(0); got != 4 {
		t.Fatalf("mock profile changed GOMAXPROCS to %d, want 4", got)
	}
}

// niceRecorder is a process whose priority changes are recorded and not made.
type niceRecorder struct {
	process.Process
	record func(nice int)
}

func (n niceRecorder) LowerNice(nice int) error {
	n.record(nice)
	return nil
}
