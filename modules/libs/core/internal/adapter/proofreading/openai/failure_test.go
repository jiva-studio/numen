package openai_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading/openai"
)

func TestAServiceWithoutAnInstructionIsRefused(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, theKey)
	cfg := proofreading.ServiceDefaults()
	cfg.Name = "test-model"

	if _, err := openai.New(cfg, ""); err == nil {
		t.Fatal("no error")
	}
}

func TestARequestTheServiceDoesNotAnswerIsGivenUpAfterTwoMinutes(t *testing.T) {
	if got := client(t, "http://127.0.0.1:1").GetHTTPClient().Timeout; got != 120*time.Second {
		t.Errorf("a request is given up after %v", got)
	}
}

func TestNoPagesIsNoRequestAndNoAnswer(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("a request was sent for no pages")
	})
	got, err := client(t, s.URL).Proofread(t.Context(), nil)
	if err != nil || got != nil {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestARequestCarriesJSONAndTheKey(t *testing.T) {
	var (
		mu                  sync.Mutex
		contentType, bearer string
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		contentType, bearer = r.Header.Get("Content-Type"), r.Header.Get("Authorization")
		mu.Unlock()
		reply(w, "1|fixed")
	})
	if _, err := client(t, s.URL).Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if contentType != "application/json" {
		t.Errorf("content type %q", contentType)
	}
	if bearer != "Bearer "+theKey {
		t.Error("the request does not carry the key")
	}
}

// One page failing ends the requests still standing for the others.
func TestAFailedPageEndsTheRequestsStillStanding(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(read(t, r).Messages[1].Content, "refused") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// The server notices a closed connection only once the body is read.
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	})
	pages := []domain.ProofreadBatch{page(1, "refused")}
	for at := 2; at <= 6; at++ {
		pages = append(pages, page(at, "held"))
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	_, err := newInFlightClient(t, s.URL, 2).Proofread(ctx, pages)
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("want the refused page, got %v", err)
	}
	if ctx.Err() != nil {
		t.Error("the run waited for the other pages until its context ended")
	}
}

func TestAContextEndingWithAFailureIsNotWaitedOn(t *testing.T) {
	s, _ := failing(t, nil, http.StatusTooManyRequests)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w := &waits{}
	c := client(t, s.URL)
	c.SetClock(w.sleep, func() time.Time {
		cancel()
		return time.Now()
	}, func(d time.Duration) time.Duration { return d })

	if _, err := c.Proofread(ctx, []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	if got := w.all(); len(got) != 0 {
		t.Errorf("waited %v after the context ended", got)
	}
}

func TestAWaitThatFailsEndsThePage(t *testing.T) {
	refused := errors.New("the wait was refused")
	s, calls := failing(t, nil, http.StatusTooManyRequests, http.StatusOK)
	c := client(t, s.URL)
	c.SetClock(func(context.Context, time.Duration) error { return refused },
		time.Now, func(d time.Duration) time.Duration { return d })

	_, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	if !errors.Is(err, refused) {
		t.Fatalf("want the wait's error, got %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("the service was asked %d times", calls.Load())
	}
}

func TestABaseURLThatIsNoURLFailsThePage(t *testing.T) {
	if _, err := client(t, "://no-scheme").Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
}

func TestARequestThatTimesOutIsSentAgain(t *testing.T) {
	var calls atomic.Int32
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			// The server notices a closed connection only once the body is read.
			_, _ = io.Copy(io.Discard, r.Body)
			<-r.Context().Done()
			return
		}
		reply(w, "1|fixed")
	})
	c, w := retrying(t, s.URL, time.Now())
	c.GetHTTPClient().Timeout = time.Second

	got, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	if err != nil {
		t.Fatal(err)
	}
	if got[1] != "1|fixed" {
		t.Errorf("got %v", got)
	}
	if len(w.all()) != 1 {
		t.Errorf("waited %v", w.all())
	}
}

func TestAServiceThatCannotBeReachedIsNotAskedAgain(t *testing.T) {
	s := server(t, func(http.ResponseWriter, *http.Request) {})
	gone := s.URL
	s.Close()
	c, w := retrying(t, gone, time.Now())

	if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	if got := w.all(); len(got) != 0 {
		t.Errorf("waited %v to ask again", got)
	}
}

func TestAnErrorCarriesTheFirst512BytesOfWhatTheServiceSaid(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, strings.Repeat("z", 1000))
	})
	_, err := client(t, s.URL).Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	if err == nil {
		t.Fatal("no error")
	}
	if got := strings.Count(err.Error(), "z"); got != 512 {
		t.Errorf("the error carries %d bytes of what the service said", got)
	}
}

func TestAMalformedAnswerFailsThePage(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[ not json`)
	})
	_, err := client(t, s.URL).Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	var malformed *json.SyntaxError
	if !errors.As(err, &malformed) {
		t.Fatalf("want a malformed answer, got %v", err)
	}
}

func TestAnAnswerWithNoChoiceLeavesThePageOut(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[]}`)
	})
	got, err := client(t, s.URL).Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

// drained is a service refusing every page with a body of the given length.
func drained(t *testing.T, length int) *atomic.Int32 {
	t.Helper()
	s, opened := connections(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(length))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, strings.Repeat("n", length))
	})
	c, _ := retrying(t, s.URL, time.Now())
	for range 2 {
		if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
			t.Fatal("no error")
		}
	}
	return opened
}

// A refusal is read to its end up to 64 KiB past what the error quotes, and the
// connection is kept; a longer one is closed.
func TestARefusalIsReadUpTo64KiBToKeepTheConnection(t *testing.T) {
	if got := drained(t, 512+64<<10).Load(); got != 1 {
		t.Errorf("a refusal of 64 KiB past the quote opened %d connections", got)
	}
	if got := drained(t, 512+64<<10+1).Load(); got != 2 {
		t.Errorf("a refusal of a byte more opened %d connections", got)
	}
}

func TestAProfileNamingNoAttemptsTakesTheDefault(t *testing.T) {
	s, calls := failing(t, nil, http.StatusServiceUnavailable)
	c, _ := retryingWith(t, s.URL, time.Now(), func(p *proofreading.Profile) { p.Attempts = 0 })
	if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	if calls.Load() != proofreading.DefaultAttempts {
		t.Errorf("the service was asked %d times", calls.Load())
	}
}

func TestAProfileNamingOneAttemptAsksOnce(t *testing.T) {
	s, calls := failing(t, nil, http.StatusServiceUnavailable)
	c, w := retryingWith(t, s.URL, time.Now(), func(p *proofreading.Profile) { p.Attempts = 1 })
	if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	if calls.Load() != 1 || len(w.all()) != 0 {
		t.Errorf("the service was asked %d times after waiting %v", calls.Load(), w.all())
	}
}

func TestTheFirstWaitIsTheProfilesDelayOrTheDefault(t *testing.T) {
	for _, one := range []struct {
		delayMS int
		want    time.Duration
	}{
		{0, proofreading.DefaultDelayMS * time.Millisecond},
		{1, time.Millisecond},
	} {
		t.Run(fmt.Sprint(one.delayMS), func(t *testing.T) {
			s, _ := failing(t, nil, http.StatusServiceUnavailable, http.StatusOK)
			c, w := retryingWith(t, s.URL, time.Now(), func(p *proofreading.Profile) { p.DelayMS = one.delayMS })
			if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err != nil {
				t.Fatal(err)
			}
			if got := w.all(); len(got) != 1 || got[0] != one.want {
				t.Errorf("waited %v, want %v", got, one.want)
			}
		})
	}
}

func TestTheLongestWaitIsTheProfilesCeilingOrTheDefault(t *testing.T) {
	for _, one := range []struct {
		maxDelayMS int
		want       time.Duration
	}{
		{0, proofreading.DefaultMaxDelayMS * time.Millisecond},
		{1, time.Millisecond},
	} {
		t.Run(fmt.Sprint(one.maxDelayMS), func(t *testing.T) {
			s, _ := failing(t, http.Header{"Retry-After": {"3600"}}, http.StatusTooManyRequests, http.StatusOK)
			c, w := retryingWith(t, s.URL, time.Now(), func(p *proofreading.Profile) { p.MaxDelayMS = one.maxDelayMS })
			if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err != nil {
				t.Fatal(err)
			}
			if got := w.all(); len(got) != 1 || got[0] != one.want {
				t.Errorf("waited %v, want %v", got, one.want)
			}
		})
	}
}

func TestAWaitOfOneNanosecondIsTakenAsAsked(t *testing.T) {
	due := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	s, _ := failing(t, http.Header{"Retry-After": {due.Format(http.TimeFormat)}}, http.StatusTooManyRequests, http.StatusOK)
	c, w := retrying(t, s.URL, due.Add(-time.Nanosecond))

	if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err != nil {
		t.Fatal(err)
	}
	if got := w.all(); len(got) != 1 || got[0] != time.Nanosecond {
		t.Errorf("waited %v", got)
	}
}

// However many times a page fails, no wait passes the ceiling or falls to
// nothing.
func TestALongRunOfFailuresWaitsAtTheCeiling(t *testing.T) {
	s, _ := failing(t, nil, http.StatusServiceUnavailable)
	c, w := retryingWith(t, s.URL, time.Now(), func(p *proofreading.Profile) { p.Attempts = 40 })

	if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	ceiling := proofreading.DefaultMaxDelayMS * time.Millisecond
	got := w.all()
	if len(got) != 39 {
		t.Fatalf("waited %d times", len(got))
	}
	for i, d := range got {
		if d <= 0 || d > ceiling {
			t.Errorf("wait %d was %v", i, d)
		}
	}
}

func TestAClientIsKnownByItsModel(t *testing.T) {
	if got := client(t, "http://127.0.0.1:1").GetName(); got != "test-model" {
		t.Errorf("known as %q", got)
	}
}

// A wait is drawn at random up to its ceiling.
func TestAWaitIsDrawnUpToItsCeiling(t *testing.T) {
	s, _ := failing(t, nil, http.StatusServiceUnavailable)
	w := &waits{}
	c := clientWith(t, s.URL, func(p *proofreading.Profile) { p.Attempts = 12 })
	c.SetSleep(w.sleep)

	if _, err := c.Proofread(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
	got := w.all()
	if len(got) != 11 {
		t.Fatalf("waited %d times", len(got))
	}
	var drawn bool
	for i, d := range got {
		if d < 0 || d > proofreading.DefaultMaxDelayMS*time.Millisecond {
			t.Errorf("wait %d was %v", i, d)
		}
		drawn = drawn || d > 0
	}
	if !drawn {
		t.Errorf("every wait was nothing: %v", got)
	}
}
