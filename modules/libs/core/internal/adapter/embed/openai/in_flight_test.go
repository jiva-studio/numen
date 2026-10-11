package openai_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/openai"
)

// peakRequests sends one batch per text through a client allowed inFlight
// requests at once, and is the most requests the service held at one moment.
func peakRequests(t *testing.T, inFlight, texts int) int {
	t.Helper()
	var (
		mu           sync.Mutex
		active, peak int
	)
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		peak = max(peak, active)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		answer(w, read(t, r), 4)
		mu.Lock()
		active--
		mu.Unlock()
	})

	t.Setenv(embed.KeyEnvVar, "test-key")
	cfg := embed.Defaults()
	cfg.Model.Dimensions = 4
	cfg, service := setServiceModel(t, cfg, func(at *embed.ServiceModel) {
		at.BaseURL, at.Name = s.URL, "test-embed"
		at.BatchCharacters = 4
	})
	c, err := openai.New(cfg.GetStoredModel(), service)
	if err != nil {
		t.Fatal(err)
	}
	c.Delay = 0
	c.InFlight = inFlight

	batch := make([]string, texts)
	for i := range batch {
		batch[i] = strings.Repeat("x", 4)
	}
	if _, err := c.Embed(t.Context(), batch); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	return peak
}

func TestAClientSendsNoMoreRequestsAtOnceThanItIsSetTo(t *testing.T) {
	if got := peakRequests(t, 2, 12); got != 2 {
		t.Errorf("the most requests at once was %d, want 2", got)
	}
}

func TestAClientSetToOneRequestSendsThemOneByOne(t *testing.T) {
	if got := peakRequests(t, 1, 6); got != 1 {
		t.Errorf("the most requests at once was %d, want 1", got)
	}
}

func TestAClientSetToNoLimitSendsSeveralRequestsAtOnce(t *testing.T) {
	if got := peakRequests(t, 0, 24); got < 2 {
		t.Errorf("the most requests at once was %d, want several", got)
	}
}
