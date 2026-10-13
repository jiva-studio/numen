package openai_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading/openai"
)

// waits records each wait a client is asked for. The jitter draws the ceiling
// it is given, so a wait is the ceiling.
type waits struct {
	mu   sync.Mutex
	took []time.Duration
}

func (w *waits) sleep(ctx context.Context, d time.Duration) error {
	w.mu.Lock()
	w.took = append(w.took, d)
	w.mu.Unlock()
	return ctx.Err()
}

func (w *waits) all() []time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]time.Duration(nil), w.took...)
}

func retrying(t *testing.T, baseURL string, now time.Time) (*openai.Client, *waits) {
	return retryingWith(t, baseURL, now, func(*proofreading.Profile) {})
}

func retryingWith(t *testing.T, baseURL string, now time.Time, set func(*proofreading.Profile)) (*openai.Client, *waits) {
	t.Helper()
	w := &waits{}
	c := clientWith(t, baseURL, set)
	c.SetClock(w.sleep, func() time.Time { return now }, func(d time.Duration) time.Duration { return d })
	return c, w
}

// failing answers the nth request with the nth status, and every request past
// the list with the last status. A status of 200 is a good reply.
func failing(t *testing.T, header http.Header, statuses ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		status := statuses[min(int(calls.Add(1)), len(statuses))-1]
		if status == http.StatusOK {
			reply(w, "1|fixed")
			return
		}
		for k, v := range header {
			w.Header()[k] = v
		}
		w.WriteHeader(status)
		fmt.Fprint(w, "no")
	})
	return s, &calls
}

func TestAThrottledPageIsSentAgainAndSucceeds(t *testing.T) {
	s, calls := failing(t, nil, http.StatusTooManyRequests, http.StatusOK)
	c, w := retrying(t, s.URL, time.Now())

	got, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")})
	if err != nil {
		t.Fatal(err)
	}
	if got[1] != "1|fixed" {
		t.Errorf("got %v", got)
	}
	if calls.Load() != 2 {
		t.Errorf("the service was asked %d times", calls.Load())
	}
	if len(w.all()) != 1 {
		t.Errorf("waited %v", w.all())
	}
}

func TestAServerErrorIsSentAgainAndSucceeds(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusRequestTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s, calls := failing(t, nil, status, status, http.StatusOK)
			c, _ := retrying(t, s.URL, time.Now())

			got, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")})
			if err != nil {
				t.Fatal(err)
			}
			if got[1] != "1|fixed" || calls.Load() != 3 {
				t.Errorf("got %v after %d requests", got, calls.Load())
			}
		})
	}
}

func TestTheWaitDoublesFromTheBaseUpToTheCeiling(t *testing.T) {
	s, _ := failing(t, nil, http.StatusServiceUnavailable)
	c, w := retrying(t, s.URL, time.Now())

	if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	base := proofreading.DefaultDelayMS * time.Millisecond
	want := []time.Duration{base, 2 * base, 4 * base}
	got := w.all()
	if len(got) != len(want) {
		t.Fatalf("waited %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("wait %d was %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRetryAfterInSecondsSetsTheWait(t *testing.T) {
	s, _ := failing(t, http.Header{"Retry-After": {"7"}}, http.StatusTooManyRequests, http.StatusOK)
	c, w := retrying(t, s.URL, time.Now())

	if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err != nil {
		t.Fatal(err)
	}
	if got := w.all(); len(got) != 1 || got[0] != 7*time.Second {
		t.Errorf("waited %v", got)
	}
}

func TestRetryAfterAsADateSetsTheWait(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	date := now.Add(12 * time.Second).Format(http.TimeFormat)
	s, _ := failing(t, http.Header{"Retry-After": {date}}, http.StatusServiceUnavailable, http.StatusOK)
	c, w := retrying(t, s.URL, now)

	if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err != nil {
		t.Fatal(err)
	}
	if got := w.all(); len(got) != 1 || got[0] != 12*time.Second {
		t.Errorf("waited %v", got)
	}
}

func TestRetryAfterBeyondTheCeilingIsCutToIt(t *testing.T) {
	s, _ := failing(t, http.Header{"Retry-After": {"3600"}}, http.StatusTooManyRequests, http.StatusOK)
	c, w := retrying(t, s.URL, time.Now())

	if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err != nil {
		t.Fatal(err)
	}
	if got := w.all(); len(got) != 1 || got[0] != proofreading.DefaultMaxDelayMS*time.Millisecond {
		t.Errorf("waited %v", got)
	}
}

func TestExhaustedAttemptsReturnTheLastError(t *testing.T) {
	s, calls := failing(t, nil, http.StatusTooManyRequests, http.StatusTooManyRequests,
		http.StatusTooManyRequests, http.StatusInternalServerError)
	c, _ := retrying(t, s.URL, time.Now())

	got, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")})
	if err == nil {
		t.Fatal("no error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("the error is not the last one: %v", err)
	}
	if calls.Load() != proofreading.DefaultAttempts {
		t.Errorf("the service was asked %d times", calls.Load())
	}
	if got != nil {
		t.Errorf("got answers alongside the error: %v", got)
	}
}

func TestAnUnretryableStatusFailsAtOnce(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s, calls := failing(t, nil, status, http.StatusOK)
			c, w := retrying(t, s.URL, time.Now())

			if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err == nil {
				t.Fatal("no error")
			}
			if calls.Load() != 1 || len(w.all()) != 0 {
				t.Errorf("asked %d times, waited %v", calls.Load(), w.all())
			}
		})
	}
}

func TestACancelledContextEndsTheWaitBetweenAttempts(t *testing.T) {
	s, calls := failing(t, nil, http.StatusTooManyRequests)
	c := client(t, s.URL)
	waiting := make(chan struct{})
	c.SetClock(func(ctx context.Context, _ time.Duration) error {
		close(waiting)
		<-ctx.Done()
		return ctx.Err()
	}, time.Now, func(d time.Duration) time.Duration { return d })

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-waiting
		cancel()
	}()

	done := make(chan error, 1)
	go func() {
		_, err := c.Proofread(ctx, []domain.ProofreadPrompt{page(1, "a line")})
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want a cancelled context, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the run did not end when the context did")
	}
	if calls.Load() != 1 {
		t.Errorf("the service was asked %d times", calls.Load())
	}
}

// connections is a server that counts the connections it accepts.
func connections(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var opened atomic.Int32
	s := httptest.NewUnstartedServer(handler)
	s.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}
	}
	s.Start()
	t.Cleanup(s.Close)
	return s, &opened
}

func TestTheConnectionIsReusedAfterASuccess(t *testing.T) {
	s, opened := connections(t, func(w http.ResponseWriter, r *http.Request) {
		reply(w, "1|fixed")
		// Bytes after the JSON value are left behind by a decoder.
		fmt.Fprint(w, strings.Repeat(" ", 4096))
	})
	c, _ := retrying(t, s.URL, time.Now())

	for range 3 {
		if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err != nil {
			t.Fatal(err)
		}
	}
	if opened.Load() != 1 {
		t.Errorf("%d connections were opened", opened.Load())
	}
}

func TestTheConnectionIsReusedAfterAnError(t *testing.T) {
	s, opened := connections(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, strings.Repeat("no ", 2048))
	})
	c, _ := retrying(t, s.URL, time.Now())

	for range 3 {
		if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err == nil {
			t.Fatal("no error")
		}
	}
	if opened.Load() != 1 {
		t.Errorf("%d connections were opened", opened.Load())
	}
}

func TestTheProfileSetsAttemptsDelayAndCeiling(t *testing.T) {
	s, calls := failing(t, http.Header{"Retry-After": {"3600"}}, http.StatusTooManyRequests)
	c, w := retryingWith(t, s.URL, time.Now(), func(p *proofreading.Profile) {
		p.Attempts, p.DelayMS, p.MaxDelayMS = 3, 100, 1000
	})

	if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	if calls.Load() != 3 {
		t.Errorf("the service was asked %d times", calls.Load())
	}
	for _, d := range w.all() {
		if d != time.Second {
			t.Errorf("waited %v, want the ceiling of one second", d)
		}
	}

	s, _ = failing(t, nil, http.StatusServiceUnavailable)
	c, w = retryingWith(t, s.URL, time.Now(), func(p *proofreading.Profile) { p.DelayMS = 100 })
	if _, err := c.Proofread(t.Context(), []domain.ProofreadPrompt{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	if got := w.all(); len(got) == 0 || got[0] != 100*time.Millisecond {
		t.Errorf("waited %v", got)
	}
}
