package openai_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestAClientWithNoQueueSendsNothing(t *testing.T) {
	c := newBatchClient(t, "")
	var sending *url.Error

	_, err := c.Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	if err == nil || errors.As(err, &sending) {
		t.Errorf("leaving: %v", err)
	}
	_, ready, err := c.GetReplies(t.Context(), "batch_1")
	if err == nil || errors.As(err, &sending) {
		t.Errorf("collecting: %v", err)
	}
	if ready {
		t.Error("a batch with no queue was taken as done")
	}
}

func TestAQueueThatIsNoURLIsReported(t *testing.T) {
	c := newBatchClient(t, "://no-scheme")
	if _, err := c.Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Error("no error out of Submit")
	}
	_, ready, err := c.GetReplies(t.Context(), "batch_1")
	if err == nil {
		t.Error("no error out of GetReplies")
	}
	if ready {
		t.Error("a batch at no URL was taken as done")
	}
}

func TestAQueueThatCannotBeReachedIsReportedAsTheRequestThatFailed(t *testing.T) {
	s := server(t, func(http.ResponseWriter, *http.Request) {})
	gone := s.URL
	s.Close()
	c := newBatchClient(t, gone)
	var sending *url.Error

	if _, err := c.Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); !errors.As(err, &sending) {
		t.Errorf("leaving: %v", err)
	}
	_, ready, err := c.GetReplies(t.Context(), "batch_1")
	if !errors.As(err, &sending) {
		t.Errorf("collecting: %v", err)
	}
	if ready {
		t.Error("a batch at a queue out of reach was taken as done")
	}
}

func TestTheQueueIsAskedWithJSONAndTheKey(t *testing.T) {
	var (
		mu   sync.Mutex
		seen []http.Header
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Clone())
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "batch_1", "status": "validating"})
	})
	c := newBatchClient(t, s.URL)
	if _, err := c.Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.GetReplies(t.Context(), "batch_1"); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 {
		t.Fatalf("the queue was asked %d times", len(seen))
	}
	if got := seen[0].Get("Content-Type"); got != "application/json" {
		t.Errorf("leaving with content type %q", got)
	}
	for i, h := range seen {
		if h.Get("Authorization") != "Bearer "+theKey {
			t.Errorf("request %d does not carry the key", i)
		}
	}
}

// statusServer answers every request with the status and a batch named
// batch_1.
func statusServer(t *testing.T, status int) string {
	t.Helper()
	return server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "batch_1", "status": "validating"})
	}).URL
}

func TestEvery2xxLeavesTheRun(t *testing.T) {
	for _, status := range []int{200, 201, 202, 299} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			name, err := newBatchClient(t, statusServer(t, status)).Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
			if err != nil || name != "batch_1" {
				t.Errorf("left under %q: %v", name, err)
			}
		})
	}
}

func TestAStatusPast2xxDoesNotLeaveTheRun(t *testing.T) {
	for _, status := range []int{300, 400, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			name, err := newBatchClient(t, statusServer(t, status)).Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
			if err == nil || name != "" {
				t.Errorf("left under %q: %v", name, err)
			}
		})
	}
}

// A 1xx that ends the exchange is a refusal, and is reported by its status.
func TestASwitchingProtocolsAnswerDoesNotLeaveTheRun(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		conn, rw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n\r\n")
		_ = rw.Flush()
	})
	name, err := newBatchClient(t, s.URL).Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	if err == nil || name != "" {
		t.Fatalf("left under %q: %v", name, err)
	}
	if errors.Is(err, io.EOF) {
		t.Errorf("the answer was read as a batch: %v", err)
	}
}

func TestAMalformedAnswerToLeavingIsReportedAsSuch(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id": not json`)
	})
	_, err := newBatchClient(t, s.URL).Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	var malformed *json.SyntaxError
	if !errors.As(err, &malformed) {
		t.Fatalf("want a malformed answer, got %v", err)
	}
}

func TestAnAnswerNamingNoBatchDoesNotLeaveTheRun(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"validating"}`)
	})
	if _, err := newBatchClient(t, s.URL).Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")}); err == nil {
		t.Fatal("no error")
	}
}

func TestAMalformedAnswerToCollectingIsReportedAsSuch(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status": not json`)
	})
	_, ready, err := newBatchClient(t, s.URL).GetReplies(t.Context(), "batch_1")
	var malformed *json.SyntaxError
	if !errors.As(err, &malformed) {
		t.Fatalf("want a malformed answer, got %v", err)
	}
	if ready {
		t.Error("a malformed answer was taken as done")
	}
}

// A completed batch under a status that is not 200 is a refusal, and nothing it
// carries is taken.
func TestARefusalToCollectIsNotTakenAsDone(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "batch_1", "status": "completed",
			"results": []map[string]any{newResult("7", "700|the first line")},
		})
	})
	replies, ready, err := newBatchClient(t, s.URL).GetReplies(t.Context(), "batch_1")
	if err == nil {
		t.Error("no error")
	}
	if ready {
		t.Error("a refused batch was taken as done")
	}
	if replies != nil {
		t.Errorf("got replies alongside the error: %v", replies)
	}
}

func TestAResultNotAboutAPageDoesNotHideTheOnesAfterIt(t *testing.T) {
	s := newBatchServer(t, "completed", []map[string]any{
		newResult("the third one", "800|a line under no number"),
		{"custom_id": "10", "response": map[string]any{"body": map[string]any{}}},
		newResult("7", "700|the first line"),
	})

	replies, ready, err := newBatchClient(t, s.URL).GetReplies(t.Context(), "batch_1")
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("a completed batch is not there yet")
	}
	if len(replies) != 1 || replies[7] != "700|the first line" {
		t.Errorf("got %v", replies)
	}
}

func TestARefusalOfTheQueueCarriesTheFirst512BytesOfWhatItSaid(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, strings.Repeat("z", 1000))
	})
	c := newBatchClient(t, s.URL)

	_, err := c.Submit(t.Context(), []domain.ProofreadBatch{page(1, "a line")})
	if err == nil {
		t.Fatal("no error out of Submit")
	}
	if got := strings.Count(err.Error(), "z"); got != 512 {
		t.Errorf("leaving carries %d bytes of what the queue said", got)
	}
	_, _, err = c.GetReplies(t.Context(), "batch_1")
	if err == nil {
		t.Fatal("no error out of GetReplies")
	}
	if got := strings.Count(err.Error(), "z"); got != 512 {
		t.Errorf("collecting carries %d bytes of what the queue said", got)
	}
}
