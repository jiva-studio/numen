package provision

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// tickingClock moves a second at every reading, so that every chunk of a download reports its progress.
type tickingClock struct{ now atomic.Int64 }

func (c *tickingClock) Now() time.Time { return time.Unix(c.now.Add(1), 0) }

type statusLog struct {
	mu       sync.Mutex
	statuses []plugin.DependencyStatus
}

func (l *statusLog) add(status plugin.DependencyStatus) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.statuses = append(l.statuses, status)
}

func (l *statusLog) states() []plugin.DependencyState {
	l.mu.Lock()
	defer l.mu.Unlock()
	states := make([]plugin.DependencyState, 0, len(l.statuses))
	for _, status := range l.statuses {
		states = append(states, status.State)
	}
	return states
}

func TestCancelStopsADownloadAndKeepsTheBytesForTheNextRun(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 4000)
	half := len(data) / 2
	var requests atomic.Int32
	var resumedFrom atomic.Value
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			if _, err := w.Write(data[:half]); err != nil {
				return
			}
			if err := http.NewResponseController(w).Flush(); err != nil {
				return
			}
			<-r.Context().Done()
			return
		}
		resumedFrom.Store(r.Header.Get("Range"))
		http.ServeContent(w, r, "w.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer server.Close()
	provisioner := NewFileProvisioner(Config{
		Dir: t.TempDir(), Discovery: discovery{dirs: []string{}}, Clock: &tickingClock{}, Client: server.Client(),
		Backoff: backoff{attempts: 3, first: time.Millisecond, sleep: func(context.Context, time.Duration) error { return nil }},
	})
	registry := NewRegistry(provisioner)
	req := newModelRequirement(server.URL+"/w.bin", data)
	seen := &statusLog{}
	started := make(chan struct{})
	var once sync.Once
	registry.SetObserver(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		seen.add(status)
		if status.BytesDownloaded > 0 {
			once.Do(func() { close(started) })
		}
	})
	answer := make(chan error, 1)
	go func() {
		_, err := registry.Provision(t.Context(), req, nil)
		answer <- err
	}()
	<-started

	if !registry.Cancel(req) {
		t.Fatal("Cancel found no download in progress")
	}
	if err := <-answer; !errors.Is(err, context.Canceled) {
		t.Fatalf("the caller received %v, want the cancellation", err)
	}
	states := seen.states()
	if last := states[len(states)-1]; last != plugin.DependencyStateMissing {
		t.Errorf("the last state = %q, want missing; all %v", last, states)
	}
	for _, state := range states {
		if state == plugin.DependencyStateFailed {
			t.Errorf("a cancellation was reported as a failure: %v", states)
		}
	}
	dir, err := provisioner.getDir(req)
	if err != nil {
		t.Fatal(err)
	}
	partial, err := filepath.Glob(filepath.Join(dir, tempPrefix+"*"))
	if err != nil || len(partial) != 1 {
		t.Fatalf("partial files = %v, %v, want one", partial, err)
	}
	info, statErr := os.Stat(partial[0])
	if statErr != nil || info.Size() == 0 || info.Size() > int64(half) {
		t.Fatalf("the partial file = %v, %v, want 1 to %d bytes", info, statErr, half)
	}
	if registry.Cancel(req) {
		t.Error("a second Cancel found a download in progress")
	}

	resolved, err := registry.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(resolved.PrimaryPath)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("the finished file differs from the data: %v", err)
	}
	if want := "bytes=" + strconv.FormatInt(info.Size(), 10) + "-"; resumedFrom.Load() != want {
		t.Errorf("the next run asked for %v, want %s", resumedFrom.Load(), want)
	}
}

func TestCancelAllStopsEveryRun(t *testing.T) {
	provisioner := &fakeProvisioner{kind: manifest.DependencyKindModel, gate: make(chan struct{})}
	registry := NewRegistry(provisioner)
	reqs := []manifest.DependencyRequirement{
		{ID: "a", Kind: manifest.DependencyKindModel}, {ID: "b", Kind: manifest.DependencyKindModel},
	}
	progressed := make(chan struct{}, 2)
	registry.SetObserver(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		if status.BytesDownloaded > 0 {
			progressed <- struct{}{}
		}
	})
	answers := make(chan error, len(reqs))
	for _, req := range reqs {
		go func() {
			_, err := registry.Provision(t.Context(), req, nil)
			answers <- err
		}()
	}
	<-progressed
	<-progressed

	registry.CancelAll()
	for range reqs {
		if err := <-answers; !errors.Is(err, context.Canceled) {
			t.Errorf("a caller received %v, want the cancellation", err)
		}
	}
}
