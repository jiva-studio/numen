package client_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
)

const (
	idle  = 150 * time.Millisecond
	steps = 8
	pause = 100 * time.Millisecond
	chunk = 1024
)

type fakeTimer struct {
	owner    *fakeTimers
	at       time.Duration
	call     func()
	isActive bool
}

func (f *fakeTimer) Reset(d time.Duration) bool {
	f.owner.mu.Lock()
	defer f.owner.mu.Unlock()
	was := f.isActive
	f.at, f.isActive = f.owner.now+d, true
	return was
}

func (f *fakeTimer) Stop() bool {
	f.owner.mu.Lock()
	defer f.owner.mu.Unlock()
	was := f.isActive
	f.isActive = false
	return was
}

// fakeTimers starts timers that fire when Advance moves its time past them.
type fakeTimers struct {
	mu     sync.Mutex
	now    time.Duration
	timers []*fakeTimer
}

func (f *fakeTimers) StartTimer(d time.Duration, call func()) syncclient.Timer {
	f.mu.Lock()
	defer f.mu.Unlock()
	timer := &fakeTimer{owner: f, at: f.now + d, call: call, isActive: true}
	f.timers = append(f.timers, timer)
	return timer
}

func (f *fakeTimers) Advance(d time.Duration) {
	f.mu.Lock()
	f.now += d
	var due []func()
	for _, timer := range f.timers {
		if timer.isActive && timer.at <= f.now {
			timer.isActive = false
			due = append(due, timer.call)
		}
	}
	f.mu.Unlock()
	for _, call := range due {
		call()
	}
}

// serveByStep writes one chunk per signal of next and, after stallAfter chunks, waits until the request ends. The
// channel it returns is closed when the handler reaches that wait.
func serveByStep(t *testing.T, stallAfter int, next <-chan struct{}) (string, <-chan struct{}) {
	t.Helper()
	stalled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			if _, err := io.Copy(io.Discard, r.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		for i := range steps {
			if i == stallAfter {
				close(stalled)
				<-r.Context().Done()
				return
			}
			if _, err := w.Write(bytes.Repeat([]byte{1}, chunk)); err != nil {
				return
			}
			if err := http.NewResponseController(w).Flush(); err != nil {
				return
			}
			if next != nil {
				select {
				case <-next:
				case <-r.Context().Done():
					return
				}
			}
		}
	}))
	t.Cleanup(server.Close)
	return server.URL, stalled
}

func TestGetBlobRange_NoTotalTimeLimitsABodyThatKeepsArriving(t *testing.T) {
	t.Parallel()
	next := make(chan struct{})
	address, _ := serveByStep(t, -1, next)
	timers := &fakeTimers{}
	client := syncclient.NewBlobHTTPClientWithTimers(address, idle, timers)
	body, err := syncclient.GetBlobRange(t.Context(), client, address+"/blob", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer closeBody(t, body)

	for i := range steps {
		if _, err := io.ReadFull(body, make([]byte, chunk)); err != nil {
			t.Fatalf("read of chunk %d after %v of the clock: %v, want the chunk: each chunk came within the idle time %v", i, time.Duration(i)*pause, err, idle)
		}
		timers.Advance(pause)
		next <- struct{}{}
	}

	if rest, err := io.ReadAll(body); err != nil || len(rest) != 0 {
		t.Errorf("rest of the body = %d bytes and %v, want the end of the body", len(rest), err)
	}
}

func TestGetBlob_ABodyThatStopsArrivingEndsAfterTheIdleTime(t *testing.T) {
	t.Parallel()
	address, stalled := serveByStep(t, 3, nil)
	timers := &fakeTimers{}
	client := syncclient.NewBlobHTTPClientWithTimers(address, idle, timers)
	result := make(chan error, 1)
	go func() {
		_, err := syncclient.GetBlob(t.Context(), client, address+"/blob")
		result <- err
	}()
	<-stalled

	timers.Advance(idle)

	if err := <-result; !errors.Is(err, syncclient.ErrTransferStalled) {
		t.Fatalf("error = %v, want ErrTransferStalled", err)
	}
}

func TestGetBlobRange_ABodyThatStopsArrivingEndsTheReadOfItsBody(t *testing.T) {
	t.Parallel()
	address, stalled := serveByStep(t, 2, nil)
	timers := &fakeTimers{}
	client := syncclient.NewBlobHTTPClientWithTimers(address, idle, timers)
	body, err := syncclient.GetBlobRange(t.Context(), client, address+"/blob", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer closeBody(t, body)
	<-stalled

	timers.Advance(idle)
	_, err = io.ReadAll(body)

	if !errors.Is(err, syncclient.ErrTransferStalled) {
		t.Errorf("error = %v, want ErrTransferStalled", err)
	}
}

// trickle is a request body that takes pause of the clock to produce each chunk.
type trickle struct {
	left   int
	timers *fakeTimers
}

func (t *trickle) Read(p []byte) (int, error) {
	if t.left == 0 {
		return 0, io.EOF
	}
	t.timers.Advance(pause)
	t.left--
	return copy(p, bytes.Repeat([]byte{2}, chunk)), nil
}

func TestBlobClient_SendsAnUploadSlowerThanTheIdleTimeThatKeepsMoving(t *testing.T) {
	t.Parallel()
	address, _ := serveByStep(t, -1, nil)
	timers := &fakeTimers{}
	client := syncclient.NewBlobHTTPClientWithTimers(address, idle, timers)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPut, address+"/blob", io.NopCloser(&trickle{left: steps, timers: timers}))
	if err != nil {
		t.Fatal(err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("a slow upload that never stood still failed: %v", err)
	}
	defer closeBody(t, response.Body)

	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %s", response.Status)
	}
}

func closeBody(t *testing.T, body io.Closer) {
	t.Helper()
	if err := body.Close(); err != nil {
		t.Error(err)
	}
}
