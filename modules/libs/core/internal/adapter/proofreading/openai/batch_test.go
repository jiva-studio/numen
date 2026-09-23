package openai_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading/openai"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// newBatchClient is a client whose queue is the fake service.
func newBatchClient(t *testing.T, batchURL string) *openai.Client {
	t.Helper()
	t.Setenv(proofreading.KeyEnvVar, theKey)
	cfg := proofreading.ServiceDefaults()
	// Nowhere: no test here asks a page at a time.
	cfg.BaseURL = "http://127.0.0.1:1/v1"
	cfg.BatchURL = batchURL
	cfg.Name = "test-model"
	c, err := openai.New(cfg, proofread.ScanInstruction)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// left is the run of pages the service receives.
type left struct {
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
	Requests []struct {
		CustomID string `json:"custom_id"`
		Body     sent   `json:"body"`
	} `json:"requests"`
}

// newRunServer answers a run with a batch id, keeping the raw body it was
// given.
func newRunServer(t *testing.T, id string, raw *[]byte) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	return server(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		*raw = body
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "validating"})
	})
}

// newBatchServer answers with a batch in one status, carrying whatever results
// are given for it.
func newBatchServer(t *testing.T, status string, results []map[string]any) *httptest.Server {
	t.Helper()
	return server(t, func(w http.ResponseWriter, r *http.Request) {
		out := map[string]any{"id": strings.TrimPrefix(r.URL.Path, "/"), "status": status}
		if results != nil {
			out["results"] = results
		}
		_ = json.NewEncoder(w).Encode(out)
	})
}

// newResult is one result in a completed batch, in the shape the service writes
// it.
func newResult(customID, text string) map[string]any {
	return map[string]any{
		"custom_id": customID,
		"response": map[string]any{
			"body": map[string]any{
				"choices": []map[string]any{
					{"message": map[string]string{"role": "assistant", "content": text}},
				},
			},
		},
	}
}

// The service stream-parses the run, so the fields arrive in the order it reads
// them.
func TestTheRunIsLeftWithItsFieldsInTheOrderTheServiceReadsThem(t *testing.T) {
	var raw []byte
	s := newRunServer(t, "batch_1", &raw)

	if _, err := newBatchClient(t, s.URL).Leave(t.Context(), []proofread.Batch{page(1, "a line")}); err != nil {
		t.Fatal(err)
	}

	body := string(raw)
	endpoint := strings.Index(body, `"endpoint"`)
	model := strings.Index(body, `"model"`)
	requests := strings.Index(body, `"requests"`)
	if endpoint < 0 || model < 0 || requests < 0 {
		t.Fatalf("a field is missing from the run: %s", body)
	}
	if !(endpoint < model && model < requests) {
		t.Errorf("the fields stand at endpoint %d, model %d, requests %d: %s",
			endpoint, model, requests, body)
	}
}

func TestEveryPageOfTheRunIsAskedWhatOnePageIsAskedOnItsOwn(t *testing.T) {
	pages := []proofread.Batch{page(7, "the frst line", "the second"), page(8, "another page")}

	var raw []byte
	s := newRunServer(t, "batch_1", &raw)

	if _, err := newBatchClient(t, s.URL).Leave(t.Context(), pages); err != nil {
		t.Fatal(err)
	}

	var run left
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	if run.Endpoint != "/v1/chat/completions" {
		t.Errorf("endpoint %q", run.Endpoint)
	}
	if run.Model != "test-model" {
		t.Errorf("model %q", run.Model)
	}
	if len(run.Requests) != len(pages) {
		t.Fatalf("got %d requests, want %d", len(run.Requests), len(pages))
	}
	for i, one := range run.Requests {
		if want := fmt.Sprint(pages[i].Number); one.CustomID != want {
			t.Errorf("request %d comes back under %q, want %q", i, one.CustomID, want)
		}
		if one.Body.Model != "test-model" {
			t.Errorf("request %d model %q", i, one.Body.Model)
		}
		if one.Body.Temperature != 0 {
			t.Errorf("request %d temperature %v", i, one.Body.Temperature)
		}
		if len(one.Body.Messages) != 2 {
			t.Fatalf("request %d has %d messages", i, len(one.Body.Messages))
		}
		if one.Body.Messages[0].Role != "system" || one.Body.Messages[0].Content != proofread.ScanInstruction {
			t.Errorf("request %d system message %q: %q", i,
				one.Body.Messages[0].Role, one.Body.Messages[0].Content)
		}
		if one.Body.Messages[1].Role != "user" || one.Body.Messages[1].Content != proofread.Ask(pages[i]) {
			t.Errorf("request %d user message %q: %q", i,
				one.Body.Messages[1].Role, one.Body.Messages[1].Content)
		}
	}
}

func TestLeaveAnswersWithTheNameTheServiceGave(t *testing.T) {
	var raw []byte
	s := newRunServer(t, "batch_1e9f", &raw)

	name, err := newBatchClient(t, s.URL).Leave(t.Context(), []proofread.Batch{page(1, "a line")})
	if err != nil {
		t.Fatal(err)
	}
	if name != "batch_1e9f" {
		t.Errorf("left under %q", name)
	}
}

func TestABatchStillWorkingIsNotThereYet(t *testing.T) {
	for _, status := range []string{"validating", "in_progress", "finalizing"} {
		t.Run(status, func(t *testing.T) {
			s := newBatchServer(t, status, nil)

			replies, ready, err := newBatchClient(t, s.URL).Collect(t.Context(), "batch_1")
			if err != nil {
				t.Fatal(err)
			}
			if ready {
				t.Errorf("a batch %s was taken as done", status)
			}
			if len(replies) != 0 {
				t.Errorf("a batch %s answered %v", status, replies)
			}
		})
	}
}

func TestACompletedBatchAnswersAboutEveryPageUnderItsOwnNumber(t *testing.T) {
	s := newBatchServer(t, "completed", []map[string]any{
		newResult("7", "700|the first line"),
		newResult("8", "800|the second line"),
	})

	replies, ready, err := newBatchClient(t, s.URL).Collect(t.Context(), "batch_1")
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("a completed batch is not there yet")
	}
	if len(replies) != 2 {
		t.Fatalf("got %d replies: %v", len(replies), replies)
	}
	if replies[7] != "700|the first line" {
		t.Errorf("page 7 came back as %q", replies[7])
	}
	if replies[8] != "800|the second line" {
		t.Errorf("page 8 came back as %q", replies[8])
	}
}

func TestABatchThatEndedNamesTheStatus(t *testing.T) {
	for _, status := range []string{"failed", "expired", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			s := newBatchServer(t, status, nil)

			replies, ready, err := newBatchClient(t, s.URL).Collect(t.Context(), "batch_1")
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), status) {
				t.Errorf("the error does not name the status: %v", err)
			}
			if ready {
				t.Error("a batch that ended was taken as done")
			}
			if replies != nil {
				t.Errorf("got replies alongside the error: %v", replies)
			}
		})
	}
}

func TestAResultNotAboutAPageOrSayingNothingIsLeftOut(t *testing.T) {
	s := newBatchServer(t, "completed", []map[string]any{
		newResult("7", "700|the first line"),
		newResult("the third one", "800|a line under no number"),
		newResult("9", "   "),
		{"custom_id": "10", "response": map[string]any{"body": map[string]any{}}},
	})

	replies, ready, err := newBatchClient(t, s.URL).Collect(t.Context(), "batch_1")
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("a completed batch is not there yet")
	}
	if len(replies) != 1 {
		t.Fatalf("got %d replies, want the one page answered: %v", len(replies), replies)
	}
	if replies[7] != "700|the first line" {
		t.Errorf("page 7 came back as %q", replies[7])
	}
}

func TestACompletedBatchWithNoResultsAnswersAboutNothing(t *testing.T) {
	s := newBatchServer(t, "completed", nil)

	replies, ready, err := newBatchClient(t, s.URL).Collect(t.Context(), "batch_1")
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("a completed batch is not there yet")
	}
	if len(replies) != 0 {
		t.Errorf("got %v", replies)
	}
}

func TestARefusedBatchNamesTheStatusAndNotTheKey(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "the service says no, and quotes %s back", r.Header.Get("Authorization"))
	})
	c := newBatchClient(t, s.URL)

	name, err := c.Leave(t.Context(), []proofread.Batch{page(1, "a line")})
	if err == nil {
		t.Fatal("no error out of Leave")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("the error does not name the status: %v", err)
	}
	if strings.Contains(err.Error(), theKey) {
		t.Error("the error out of Leave carries the key")
	}
	if name != "" {
		t.Errorf("left under %q alongside the error", name)
	}

	if _, _, err := c.Collect(t.Context(), "batch_1"); err == nil {
		t.Fatal("no error out of Collect")
	} else if strings.Contains(err.Error(), theKey) {
		t.Error("the error out of Collect carries the key")
	}
}

// A client with a queue is one of these, and the use case is handed it as one.
