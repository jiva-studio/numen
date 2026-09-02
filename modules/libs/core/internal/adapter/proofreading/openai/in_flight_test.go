package openai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading/openai"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// crowd is the service, counting how many requests stand at once. It holds each
// one long enough for the others to arrive.
func crowd(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()
	var (
		mu   sync.Mutex
		now  int
		most int
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		now++
		most = max(most, now)
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		now--
		mu.Unlock()
		reply(w, "")
	})
	return s, func() int {
		mu.Lock()
		defer mu.Unlock()
		return most
	}
}

func asking(t *testing.T, baseURL string, inFlight int) *openai.Client {
	t.Helper()
	t.Setenv(proofreading.KeyEnvVar, theKey)
	cfg := proofreading.ServiceDefaults()
	cfg.BaseURL = baseURL
	cfg.Name = "test-model"
	cfg.InFlight = inFlight
	c, err := openai.New(cfg, proofread.ScanInstruction)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// A profile says how many batches the service is asked about at once.
func TestTheServiceIsAskedAboutAsManyBatchesAsTheProfileNames(t *testing.T) {
	for _, want := range []int{1, 2, 6} {
		s, most := crowd(t)
		var batches []proofread.Batch
		for at := range 8 {
			batches = append(batches, page(at, "a line"))
		}

		if _, err := asking(t, s.URL, want).Read(context.Background(), batches); err != nil {
			t.Fatal(err)
		}
		if got := most(); got != want {
			t.Errorf("%d asked for, %d stood at once", want, got)
		}
	}
}

// A profile naming none takes what the service is asked by default.
func TestAProfileNamingNoNumberTakesTheDefault(t *testing.T) {
	s, most := crowd(t)
	var batches []proofread.Batch
	for at := range 8 {
		batches = append(batches, page(at, "a line"))
	}

	if _, err := asking(t, s.URL, 0).Read(context.Background(), batches); err != nil {
		t.Fatal(err)
	}
	if got := most(); got != 4 {
		t.Errorf("%d stood at once", got)
	}
}
