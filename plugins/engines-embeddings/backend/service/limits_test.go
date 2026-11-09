package service

import (
	"context"
	"runtime"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

func TestTheProfileSetsThreadsBatchAndPriorityOnEveryMachine(t *testing.T) {
	table := []struct {
		profile performance.Profile
		cpus    int
		want    runLimits
	}{
		{performance.Eco, 2, runLimits{threads: 1, batchTexts: 2, nice: 19}},
		{performance.Eco, 16, runLimits{threads: 1, batchTexts: 2, nice: 19}},
		{performance.Balanced, 2, runLimits{threads: 1, batchTexts: 4, nice: 10}},
		{performance.Balanced, 4, runLimits{threads: 1, batchTexts: 4, nice: 10}},
		{performance.Balanced, 8, runLimits{threads: 2, batchTexts: 4, nice: 10}},
		{performance.Balanced, 16, runLimits{threads: 4, batchTexts: 4, nice: 10}},
		{performance.Maximum, 2, runLimits{threads: 1, batchTexts: 8, nice: 0}},
		{performance.Maximum, 4, runLimits{threads: 2, batchTexts: 8, nice: 0}},
		{performance.Maximum, 8, runLimits{threads: 4, batchTexts: 8, nice: 0}},
		{performance.Maximum, 16, runLimits{threads: 8, batchTexts: 8, nice: 0}},
	}
	for _, c := range table {
		if got := getRunLimits(c.profile, c.cpus); got != c.want {
			t.Errorf("%s on %d cores: %+v, want %+v", c.profile, c.cpus, got, c.want)
		}
	}
}

func TestDefaultProfileUsesAtMostQuarterOfCores(t *testing.T) {
	for _, cpus := range []int{4, 8, 12, 16, 32} {
		if got := getRunLimits(performance.Read(nil), cpus).threads; got*4 > cpus {
			t.Errorf("%d threads on %d cores", got, cpus)
		}
	}
}

// openRecord is what the server opened a model with, and the models it closed.
type openRecord struct {
	mu      sync.Mutex
	opened  []engine.LocalModel
	nices   []int
	closing int
}

type recordedEmbedder struct {
	engine.MockEmbedder
	record *openRecord
}

func (e recordedEmbedder) Close() error {
	e.record.mu.Lock()
	defer e.record.mu.Unlock()
	e.record.closing++
	return nil
}

func newRecordedServer(t *testing.T) (*embeddingServer, *openRecord) {
	t.Helper()
	srv := newEmbeddingServer()
	record := &openRecord{}
	srv.open = func(_ context.Context, _ engine.ModelInfo, local engine.LocalModel) (textEmbedder, error) {
		record.mu.Lock()
		defer record.mu.Unlock()
		record.opened = append(record.opened, local)
		return recordedEmbedder{record: record}, nil
	}
	srv.lowerNice = func(nice int) error {
		record.mu.Lock()
		defer record.mu.Unlock()
		record.nices = append(record.nices, nice)
		return nil
	}
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
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if got := record.opened[0]; got.Threads != 1 || got.BatchTexts != ecoBatchTexts {
		t.Errorf("opened with %d threads and batches of %d, want 1 and %d", got.Threads, got.BatchTexts, ecoBatchTexts)
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
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	first := record.opened[0]

	srv.updateConfig(map[string]string{performance.ConfigKey: "eco"}, nil)
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if record.openedCount() != 2 {
		t.Fatalf("%d models opened, want 2", record.openedCount())
	}
	second := record.opened[1]
	if second.Threads != 1 || second.BatchTexts != ecoBatchTexts || first.BatchTexts != engine.GetDefaultBatchTexts() {
		t.Errorf("opened %d threads and %d texts after %d texts, want 1 thread and %d texts", second.Threads, second.BatchTexts, first.BatchTexts, ecoBatchTexts)
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
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	srv.updateConfig(values, nil)
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
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
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
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

	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	srv.isProfileSet = false
	srv.applyProfileLocked(performance.Eco)
	if srv.embedder == nil {
		t.Fatal("embedder was retired when limits did not change")
	}

	canceled := false
	srv.opening = &modelOpening{
		ready:  make(chan struct{}),
		cancel: func() { canceled = true },
	}
	srv.isProfileSet = false
	srv.applyProfileLocked(performance.Eco)
	if canceled || srv.opening == nil {
		t.Fatal("opening was retired when limits did not change")
	}

	srv.applyProfileLocked(performance.Maximum)
	if !canceled || srv.opening != nil {
		t.Fatal("opening was not retired when limits changed")
	}

	runtime.GOMAXPROCS(4)
	srv.isMock = true
	srv.applyProfileLocked(performance.Eco)
	if got := runtime.GOMAXPROCS(0); got != 4 {
		t.Fatalf("mock profile changed GOMAXPROCS to %d, want 4", got)
	}
}
