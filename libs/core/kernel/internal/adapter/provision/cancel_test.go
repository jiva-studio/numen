package provision

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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
	req := newModelRequirement(server.URL+"/w.bin", data)
	seen := &statusLog{}
	started := make(chan struct{})
	var once sync.Once
	registry := NewObservedRegistry(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		seen.add(status)
		if status.BytesDownloaded > 0 {
			once.Do(func() { close(started) })
		}
	}, provisioner)
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
	reqs := []manifest.DependencyRequirement{
		{ID: "a", Kind: manifest.DependencyKindModel}, {ID: "b", Kind: manifest.DependencyKindModel},
	}
	progressed := make(chan struct{}, 2)
	registry := NewObservedRegistry(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		if status.BytesDownloaded > 0 {
			progressed <- struct{}{}
		}
	}, provisioner)
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

func TestAJoiningCallerMayCallTheRegistryFromItsProgressReplay(t *testing.T) {
	provisioner := &fakeProvisioner{kind: manifest.DependencyKindModel, gate: make(chan struct{})}
	req := manifest.DependencyRequirement{ID: "a", Kind: manifest.DependencyKindModel}
	progressed := make(chan struct{}, 1)
	registry := NewObservedRegistry(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		if status.BytesDownloaded > 0 {
			select {
			case progressed <- struct{}{}:
			default:
			}
		}
	}, provisioner)
	first := make(chan error, 1)
	go func() {
		_, err := registry.Provision(t.Context(), req, nil)
		first <- err
	}()
	<-progressed

	second := make(chan error, 1)
	go func() {
		_, err := registry.Provision(t.Context(), req, func(plugin.DependencyStatus) { registry.Cancel(req) })
		second <- err
	}()

	for _, answer := range []chan error{first, second} {
		if err := <-answer; !errors.Is(err, context.Canceled) {
			t.Errorf("a caller received %v, want the cancellation", err)
		}
	}
}

func TestAJoiningCallerReceivesTheReplayBeforeTheLaterStatuses(t *testing.T) {
	var group flightGroup
	release := make(chan struct{})
	reported := make(chan struct{})
	run := func(_ context.Context, report func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error) {
		report(plugin.DependencyStatus{Message: "1"})
		close(reported)
		<-release
		report(plugin.DependencyStatus{Message: "2"})
		return plugin.ResolvedDependency{}, nil
	}
	leader := make(chan error, 1)
	go func() {
		_, err := group.do(t.Context(), "k", nil, run)
		leader <- err
	}()
	<-reported

	var mu sync.Mutex
	var got []string
	replayed := make(chan struct{})
	joined := make(chan error, 1)
	go func() {
		_, err := group.do(t.Context(), "k", func(status plugin.DependencyStatus) {
			mu.Lock()
			got = append(got, status.Message)
			mu.Unlock()
			if status.Message == "1" {
				close(replayed)
			}
		}, run)
		joined <- err
	}()
	<-replayed
	close(release)
	if err := <-leader; err != nil {
		t.Errorf("the leading caller failed: %v", err)
	}
	if err := <-joined; err != nil {
		t.Errorf("the joining caller failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Errorf("the joining caller received %v, want 1 then 2", got)
	}
}

// resolveSignal reports every Resolve, which Registry.Provision calls just before it joins or starts a run.
type resolveSignal struct {
	*FileProvisioner
	resolved chan struct{}
}

func (s *resolveSignal) Resolve(req manifest.DependencyRequirement) (plugin.ResolvedDependency, plugin.DependencyState, error) {
	select {
	case s.resolved <- struct{}{}:
	default:
	}
	return s.FileProvisioner.Resolve(req)
}

func TestProvisionAfterCancelStartsANewDownloadInsteadOfJoiningTheCanceledOne(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 4000)
	half := len(data) / 2
	var requests atomic.Int32
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
		http.ServeContent(w, r, "w.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer server.Close()
	provisioner := &resolveSignal{
		FileProvisioner: NewFileProvisioner(Config{
			Dir: t.TempDir(), Discovery: discovery{dirs: []string{}}, Clock: &tickingClock{}, Client: server.Client(),
			Backoff: backoff{attempts: 3, first: time.Millisecond, sleep: func(context.Context, time.Duration) error { return nil }},
		}),
		resolved: make(chan struct{}, 1),
	}
	req := newModelRequirement(server.URL+"/w.bin", data)
	started := make(chan struct{})
	var startOnce sync.Once
	canceling := make(chan struct{})
	finish := make(chan struct{})
	var cancelOnce sync.Once
	registry := NewObservedRegistry(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		if status.BytesDownloaded > 0 {
			startOnce.Do(func() { close(started) })
		}
		if status.State == plugin.DependencyStateMissing {
			cancelOnce.Do(func() {
				close(canceling)
				<-finish
			})
		}
	}, provisioner)
	first := make(chan error, 1)
	go func() {
		_, err := registry.Provision(t.Context(), req, nil)
		first <- err
	}()
	<-started
	<-provisioner.resolved
	registry.Cancel(req)
	<-canceling

	type outcome struct {
		resolved plugin.ResolvedDependency
		err      error
	}
	second := make(chan outcome, 1)
	go func() {
		resolved, err := registry.Provision(t.Context(), req, nil)
		second <- outcome{resolved, err}
	}()
	<-provisioner.resolved
	for range 1000 {
		runtime.Gosched()
	}
	close(finish)

	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("the canceled caller received %v, want the cancellation", err)
	}
	got := <-second
	if got.err != nil {
		t.Fatalf("a Provision after Cancel received %v, want a new download", got.err)
	}
	body, err := os.ReadFile(got.resolved.PrimaryPath)
	if err != nil || !bytes.Equal(body, data) {
		t.Fatalf("the finished file differs from the data: %v", err)
	}
}

func TestACancelFromTheLastProgressReportStillCancelsTheRun(t *testing.T) {
	provisioner := &fakeProvisioner{kind: manifest.DependencyKindModel}
	req := manifest.DependencyRequirement{ID: "a", Kind: manifest.DependencyKindModel}
	seen := &statusLog{}
	var registry *Registry
	registry = NewObservedRegistry(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		seen.add(status)
		if status.BytesDownloaded == status.BytesTotal && status.BytesTotal > 0 {
			registry.Cancel(req)
		}
	}, provisioner)

	if _, err := registry.Provision(t.Context(), req, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("the caller received %v, want the cancellation", err)
	}
	states := seen.states()
	if last := states[len(states)-1]; last != plugin.DependencyStateMissing {
		t.Errorf("the last state = %q, want missing; all %v", last, states)
	}
}
