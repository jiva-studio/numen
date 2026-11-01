package vectors

import (
	"slices"
	"sync"
)

// tracker reports the progress of one run in the order the changes happened,
// and knows which sources the run is working on.
//
// The producer reads ahead of the model, so the source it last read is not the
// source being embedded, and the source of the last stored batch is behind it.
// The sources being worked on are those of the oldest batches that left the
// producer and are not yet stored, as many batches as there are infer workers.
type tracker struct {
	u       Embed
	workers int
	res     *EmbedResult
	// mu guards res.
	mu *sync.Mutex
	// order serialises the reports, so a callback never receives an older
	// snapshot after a newer one.
	order sync.Mutex

	flightMu sync.Mutex
	nextID   uint64
	flight   []flightBatch
}

// flightBatch is a batch that left the producer and is not stored yet.
type flightBatch struct {
	id      uint64
	sources []string
}

func newTracker(u Embed, workers int, res *EmbedResult, mu *sync.Mutex) *tracker {
	return &tracker{u: u, workers: workers, res: res, mu: mu}
}

// enter lists a batch of the sources given as being worked on and returns its id.
func (t *tracker) enter(sources []string) uint64 {
	t.flightMu.Lock()
	defer t.flightMu.Unlock()
	t.nextID++
	t.flight = append(t.flight, flightBatch{id: t.nextID, sources: sources})
	return t.nextID
}

// leave takes a stored batch out of the list.
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

// isIdle reports whether no batch has left the producer without being stored.
func (t *tracker) isIdle() bool {
	t.flightMu.Lock()
	defer t.flightMu.Unlock()
	return len(t.flight) == 0
}

// getActive returns the distinct sources of the oldest batches that are being
// worked on, in the order they were sent.
func (t *tracker) getActive() (active []string) {
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

// publish applies change to the result and reports a snapshot of it. The
// snapshot names the sources being worked on, and the first of them is the
// current source; when no batch is waiting the current source is the one the
// change set.
func (t *tracker) publish(change func(res *EmbedResult)) {
	t.order.Lock()
	defer t.order.Unlock()

	t.mu.Lock()
	change(t.res)
	t.res.Active = t.getActive()
	if len(t.res.Active) > 0 {
		t.res.CurrentSource = t.res.Active[0]
	}
	snapshot := *t.res
	t.mu.Unlock()
	t.u.progress(snapshot)
}

// publishNext reports source as the current one when no batch is waiting: the
// first chunk read is then the next to be embedded. Once batches wait, the
// current source is that of the oldest.
func (t *tracker) publishNext(source string) {
	t.publish(func(res *EmbedResult) {
		if t.isIdle() {
			res.CurrentSource = source
		}
	})
}
