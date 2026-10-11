package openai_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/openai"
)

// newClient opens a client at the service the change describes, holding vectors
// of the given width.
func newClient(t *testing.T, dimensions int, change func(*embed.ServiceModel)) (*openai.Client, error) {
	t.Helper()
	t.Setenv(embed.KeyEnvVar, "test-key")
	cfg := embed.Defaults()
	cfg.Model.Dimensions = dimensions
	cfg, service := setServiceModel(t, cfg, func(at *embed.ServiceModel) {
		at.BaseURL, at.Name = "http://127.0.0.1:1", "test-embed"
		change(at)
	})
	return openai.New(cfg.GetStoredModel(), service)
}

func TestAClientIsRefusedWithoutWhatItNeeds(t *testing.T) {
	for _, one := range []struct {
		what       string
		dimensions int
		change     func(*embed.ServiceModel)
	}{
		{"no base URL", 4, func(at *embed.ServiceModel) { at.BaseURL = "" }},
		{"no model name", 4, func(at *embed.ServiceModel) { at.Name = "" }},
		{"no width", 0, func(*embed.ServiceModel) {}},
		{"a negative width", -1, func(*embed.ServiceModel) {}},
	} {
		t.Run(one.what, func(t *testing.T) {
			if _, err := newClient(t, one.dimensions, one.change); err == nil {
				t.Error("the client was opened")
			}
		})
	}
}

func TestAClientHoldingOneDimensionIsOpened(t *testing.T) {
	c, err := newClient(t, 1, func(*embed.ServiceModel) {})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Model().Dimensions; got != 1 {
		t.Errorf("the client holds %d dimensions", got)
	}
}

// A hung service is given up on, and connections are kept for the next request.
func TestAClientHoldsItsTimeoutsAndConnections(t *testing.T) {
	c, err := newClient(t, 4, func(*embed.ServiceModel) {})
	if err != nil {
		t.Fatal(err)
	}
	h := c.GetHTTPClient()
	if h.Timeout != 90*time.Second {
		t.Errorf("a request is given up after %v", h.Timeout)
	}
	tr, ok := h.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("the transport is a %T", h.Transport)
	}
	if tr.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("a handshake is given up after %v", tr.TLSHandshakeTimeout)
	}
	if tr.IdleConnTimeout != 90*time.Second {
		t.Errorf("an idle connection is kept for %v", tr.IdleConnTimeout)
	}
	if tr.ExpectContinueTimeout != time.Second {
		t.Errorf("a continue is waited for %v", tr.ExpectContinueTimeout)
	}
	if tr.MaxIdleConns != 100 || tr.MaxIdleConnsPerHost != 20 {
		t.Errorf("%d idle connections are kept, %d to one host", tr.MaxIdleConns, tr.MaxIdleConnsPerHost)
	}
}

func TestADefaultClientSendsFourRequestsAtOnce(t *testing.T) {
	if got := peakRequests(t, 0, 24); got != 4 {
		t.Errorf("the most requests at once was %d, want 4", got)
	}
}

func TestADefaultClientSendsARequestFiveTimesBeforeReportingIt(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	if _, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"}); err == nil {
		t.Fatal("want an error")
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 5 {
		t.Errorf("made %d calls, want 5", calls)
	}
}

func TestAClientSetToNoAttemptsSendsOnce(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c := client(t, s.URL, 4)
	c.Attempts = 0
	if _, err := c.Embed(t.Context(), []string{"one"}); err == nil {
		t.Fatal("want an error")
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("made %d calls, want 1", calls)
	}
}

func TestTheFirstRequestIsSentWithoutWaiting(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		answer(w, read(t, r), 4)
	})
	c := client(t, s.URL, 4)
	c.Delay = time.Hour
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if _, err := c.Embed(ctx, []string{"one"}); err != nil {
		t.Fatal(err)
	}
}

// The wait before each retry doubles the one before it.
func TestEachRetryWaitsTwiceAsLongAsTheOneBefore(t *testing.T) {
	const delay = 100 * time.Millisecond
	var (
		mu       sync.Mutex
		arrivals []time.Time
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		arrivals = append(arrivals, time.Now())
		n := len(arrivals)
		mu.Unlock()
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		answer(w, read(t, r), 4)
	})
	c := client(t, s.URL, 4)
	c.Attempts, c.Delay = 3, delay
	if _, err := c.Embed(t.Context(), []string{"one"}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(arrivals) != 3 {
		t.Fatalf("made %d calls, want 3", len(arrivals))
	}
	if gap := arrivals[1].Sub(arrivals[0]); gap < delay {
		t.Errorf("the first retry came %v after the first request, want at least %v", gap, delay)
	}
	if gap := arrivals[2].Sub(arrivals[1]); gap < 2*delay {
		t.Errorf("the second retry came %v after the first, want at least %v", gap, 2*delay)
	}
}

func TestRetryAfterReplacesTheClientsOwnDelay(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		answer(w, read(t, r), 4)
	})
	c := client(t, s.URL, 4)
	c.Delay = time.Hour
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if _, err := c.Embed(ctx, []string{"one"}); err != nil {
		t.Fatal(err)
	}
}

// A context that ends during the wait is reported as itself, and nothing more is
// sent.
func TestAContextEndingBeforeARetryEndsTheRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		cancel()
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c := client(t, s.URL, 4)
	c.Delay = time.Hour
	_, err := c.Embed(ctx, []string{"one"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want a cancelled context, got %v", err)
	}
	var sending *url.Error
	if errors.As(err, &sending) {
		t.Errorf("a request was sent after the context ended: %v", err)
	}
}

func TestAMalformedAnswerIsSentAgainAndReportedAsSuch(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		fmt.Fprint(w, `{"data":[`)
		fmt.Fprint(w, "not json")
	})
	c := client(t, s.URL, 4)
	c.Attempts = 2
	_, err := c.Embed(t.Context(), []string{"one"})
	var malformed *json.SyntaxError
	if !errors.As(err, &malformed) {
		t.Fatalf("want a malformed answer, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Errorf("made %d calls, want 2", calls)
	}
}

func TestABaseURLThatIsNoURLIsReported(t *testing.T) {
	c, err := newClient(t, 4, func(at *embed.ServiceModel) { at.BaseURL = "://no-scheme" })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Embed(t.Context(), []string{"one"}); err == nil {
		t.Fatal("want an error")
	}
}

func TestARequestIsSentAsJSON(t *testing.T) {
	var (
		mu          sync.Mutex
		contentType string
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		contentType = r.Header.Get("Content-Type")
		mu.Unlock()
		answer(w, read(t, r), 4)
	})
	if _, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if contentType != "application/json" {
		t.Errorf("content type %q", contentType)
	}
}

func TestAnErrorCarriesTheFirst4096BytesOfWhatTheServiceSaid(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, strings.Repeat("z", 5000))
	})
	_, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"})
	if err == nil {
		t.Fatal("want an error")
	}
	if got := strings.Count(err.Error(), "z"); got != 4096 {
		t.Errorf("the error carries %d bytes of what the service said", got)
	}
}

// A 400 is the batch being malformed, which is the batching rule's to fix.
func TestAMalformedBatchIsReportedAsTheBatchingRule(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	_, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"})
	if !errors.Is(err, openai.ErrRejected) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "batching rule") {
		t.Errorf("the error does not name the batching rule: %v", err)
	}
}

func TestAStatusBelow500IsNotRetried(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(499)
	})
	_, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"})
	if !errors.Is(err, openai.ErrRejected) {
		t.Fatalf("got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("made %d calls, want 1", calls)
	}
}

func TestAVectorOutsideTheAnswerIsRefused(t *testing.T) {
	for _, index := range []int{-1, 2} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			s := server(t, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"data":[
					{"index":0,"embedding":[1,0]},
					{"index":%d,"embedding":[0,1]}]}`, index)
			})
			got, err := client(t, s.URL, 2).Embed(t.Context(), []string{"first", "second"})
			if err == nil {
				t.Fatalf("got %v", got)
			}
		})
	}
}

func TestAVectorArrivingTwiceIsRefused(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"index":0,"embedding":[1,0]},
			{"index":0,"embedding":[0,1]}]}`)
	})
	_, err := client(t, s.URL, 2).Embed(t.Context(), []string{"first", "second"})
	if err == nil {
		t.Fatal("want an error")
	}
	if !strings.Contains(err.Error(), "twice") {
		t.Errorf("the error does not say the vector came twice: %v", err)
	}
}
