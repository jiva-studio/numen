package openai_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading/openai"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// theKey is what the tests configure. No error and no message a test reads may
// contain it.
const theKey = "sk-proofread-secret"

// server is the service, faked. No test reaches a network.
func server(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return s
}

func client(t *testing.T, baseURL string) *openai.Client {
	t.Helper()
	t.Setenv(proofreading.KeyEnvVar, theKey)
	cfg := proofreading.ServiceDefaults()
	cfg.BaseURL = baseURL
	cfg.Name = "test-model"
	c, err := openai.New(cfg, proofread.ScanInstruction)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// sent is the body the service receives.
type sent struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	Messages    []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func read(t *testing.T, r *http.Request) sent {
	t.Helper()
	var in sent
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		t.Fatal(err)
	}
	return in
}

// reply writes one answer in the shape the service answers in.
func reply(w http.ResponseWriter, text string) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": text}}},
	})
}

func page(at int, lines ...string) proofread.Batch {
	p := proofread.Batch{Number: at}
	for i, text := range lines {
		p.Lines = append(p.Lines, proofread.Line{Number: at*100 + i, Text: text})
	}
	return p
}

func TestTheServiceIsAskedAboutOnePageWithTheInstruction(t *testing.T) {
	one := page(7, "the frst line", "the second")

	var (
		mu   sync.Mutex
		got  sent
		path string
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got, path = read(t, r), r.URL.Path
		mu.Unlock()
		reply(w, "700|the first line")
	})

	if _, err := client(t, s.URL).Proofread(t.Context(), []proofread.Batch{one}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if path != "/chat/completions" {
		t.Errorf("path %q", path)
	}
	if got.Model != "test-model" {
		t.Errorf("model %q", got.Model)
	}
	if got.Temperature != 0 {
		t.Errorf("temperature %v", got.Temperature)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("got %d messages", len(got.Messages))
	}
	if got.Messages[0].Role != "system" || got.Messages[0].Content != proofread.ScanInstruction {
		t.Errorf("system message %q: %q", got.Messages[0].Role, got.Messages[0].Content)
	}
	if got.Messages[1].Role != "user" || got.Messages[1].Content != proofread.Ask(one) {
		t.Errorf("user message %q: %q", got.Messages[1].Role, got.Messages[1].Content)
	}
}

func TestEveryPageComesBackUnderItsOwnNumber(t *testing.T) {
	var pages []proofread.Batch
	for at := 1; at <= 6; at++ {
		pages = append(pages, page(at, fmt.Sprintf("page %d as read", at)))
	}
	// The page nothing is answered about.
	pages[2] = page(3, "quiet")

	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		asked := read(t, r).Messages[1].Content
		if strings.Contains(asked, "quiet") {
			reply(w, "")
			return
		}
		reply(w, asked)
	})

	got, err := client(t, s.URL).Proofread(t.Context(), pages)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d answers: %v", len(got), got)
	}
	for _, p := range pages {
		if p.Number == 3 {
			if _, ok := got[3]; ok {
				t.Errorf("page 3 was answered: %q", got[3])
			}
			continue
		}
		if got[p.Number] != proofread.Ask(p) {
			t.Errorf("page %d came back as %q", p.Number, got[p.Number])
		}
	}
}

func TestARefusedRunNamesTheStatusAndNotTheKey(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s := server(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprintf(w, "the service says no, and quotes %s back", r.Header.Get("Authorization"))
			})

			got, err := client(t, s.URL).Proofread(t.Context(), []proofread.Batch{page(1, "a line")})
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Errorf("error does not name the status: %v", err)
			}
			if strings.Contains(err.Error(), theKey) {
				t.Error("the error carries the key")
			}
			if got != nil {
				t.Errorf("got answers alongside the error: %v", got)
			}
		})
	}
}

func TestACancelledContextStopsTheRun(t *testing.T) {
	arrived := make(chan struct{}, 1)
	release := make(chan struct{})
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case arrived <- struct{}{}:
		default:
		}
		<-release
	})
	// Closing release lets the handler return. It is registered after the
	// server, so it runs before the server is closed.
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-arrived
		cancel()
	}()

	_, err := client(t, s.URL).Proofread(ctx, []proofread.Batch{page(1, "a line"), page(2, "another")})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want a cancelled context, got %v", err)
	}
}

func TestAServiceWithoutAModelNameIsRefused(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, theKey)
	cfg := proofreading.ServiceDefaults()

	_, err := openai.New(cfg, proofread.ScanInstruction)
	if err == nil {
		t.Fatal("no error")
	}
	if !strings.Contains(err.Error(), "model name") {
		t.Errorf("error does not say what is missing: %v", err)
	}
	if strings.Contains(err.Error(), theKey) {
		t.Error("the error carries the key")
	}
}

func TestAServiceWithoutAKeyIsRefused(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "")
	cfg := proofreading.ServiceDefaults()
	cfg.Name = "test-model"

	_, err := openai.New(cfg, proofread.ScanInstruction)
	if !errors.Is(err, openai.ErrNoKey) {
		t.Fatalf("want a missing key, got %v", err)
	}
}
