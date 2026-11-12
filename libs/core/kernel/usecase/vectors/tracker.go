package vectors

import (
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
)

// tracker reports the progress of one run in the order the changes happened,
// and knows which sources the run is working on.
//
// The producer reads ahead of the model, so the source it last read is not the
// source being embedded, and the source of the last stored batch is behind it.
// The sources being worked on are those of the oldest batches that left the
// producer and are not yet stored, as many batches as there are infer workers.
type tracker struct {
	onProgress func(vaultreading.EmbedResult)
	workers    int
	res        *vaultreading.EmbedResult
	mu         *sync.Mutex
	order      sync.Mutex

	flightMu sync.Mutex
	nextID   uint64
	flight   []flightBatch
}

// flightBatch is a batch that left the producer and is not stored yet.
type flightBatch struct {
	id      uint64
	sources []string
}

func newTracker(onProgress func(vaultreading.EmbedResult), workers int, res *vaultreading.EmbedResult, mu *sync.Mutex) *tracker {
	return &tracker{onProgress: onProgress, workers: workers, res: res, mu: mu}
}

func (t *tracker) enter(sources []string) uint64 {
	t.flightMu.Lock()
	defer t.flightMu.Unlock()
	t.nextID++
	t.flight = append(t.flight, flightBatch{id: t.nextID, sources: sources})
	return t.nextID
}

func (t *tracker) leave(id uint64) {
	t.flightMu.Lock()
	defer t.flightMu.Unlock()
	for i, one := range t.flight {
		if one.id == id {
			t.flight = append(t.flight[:i], t.flight[i+1:]...)
			return
		}
	}
}

func (t *tracker) isIdle() bool {
	t.flightMu.Lock()
	defer t.flightMu.Unlock()
	return len(t.flight) == 0
}

func (t *tracker) getActiveSources() (active []string) {
	t.flightMu.Lock()
	defer t.flightMu.Unlock()
	for _, one := range t.flight[:min(t.workers, len(t.flight))] {
		for _, source := range one.sources {
			if !slices.Contains(active, source) {
				active = append(active, source)
			}
		}
	}
	return active
}

func (t *tracker) publish(change func(res *vaultreading.EmbedResult)) {
	t.order.Lock()
	defer t.order.Unlock()

	t.mu.Lock()
	change(t.res)
	t.res.Active = t.getActiveSources()
	if len(t.res.Active) > 0 {
		t.res.CurrentSource = t.res.Active[0]
	}
	snapshot := *t.res
	t.mu.Unlock()
	if t.onProgress != nil {
		t.onProgress(snapshot)
	}
}

func (t *tracker) publishNext(source string) {
	t.publish(func(res *vaultreading.EmbedResult) {
		if t.isIdle() {
			res.CurrentSource = source
		}
	})
}

func (t *tracker) withdraw(id uint64) {
	t.leave(id)
	t.publish(func(res *vaultreading.EmbedResult) { res.CurrentSource = "" })
}
