package openai_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/openai"
)

// server is the service, faked. No test reaches a network.
func server(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return s
}

// setServiceModel is the settings with the vault indexed at the service the
// changes describe, and that service: the two a client is opened with.
func setServiceModel(t *testing.T, cfg embed.Config, change func(*embed.ServiceModel)) (embed.Config, embed.ServiceModel) {
	t.Helper()
	cfg.Indexing.Use = embed.UseService
	service, ok := cfg.Indexing.Service()
	if !ok {
		t.Fatal("the vault is not indexed by a service")
	}
	change(&service)
	cfg.Indexing = cfg.Indexing.SetService(service)
	return cfg, service
}

func client(t *testing.T, baseURL string, dimensions int) *openai.Client {
	t.Helper()
	t.Setenv(embed.KeyEnvVar, "test-key")
	cfg := embed.Defaults()
	cfg.Model.Dimensions = dimensions
	cfg, service := setServiceModel(t, cfg, func(at *embed.ServiceModel) {
		at.BaseURL, at.Name = baseURL, "test-embed"
	})
	c, err := openai.New(cfg.GetStoredModel(), service)
	if err != nil {
		t.Fatal(err)
	}
	// No test waits for a real backoff.
	c.Delay = 0
	return c
}

type sent struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

func answer(w http.ResponseWriter, in sent, dimensions int) {
	type item struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	}
	out := struct {
		Data []item `json:"data"`
	}{}
	for i := range in.Input {
		v := make([]float32, dimensions)
		for d := range v {
			v[d] = float32(i+1) * float32(d+1)
		}
		out.Data = append(out.Data, item{Index: i, Embedding: v})
	}
	_ = json.NewEncoder(w).Encode(out)
}

func read(t *testing.T, r *http.Request) sent {
	t.Helper()
	var in sent
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		t.Fatal(err)
	}
	return in
}

func TestVectorsComeBackNormalisedAndInOrder(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/embeddings" {
			t.Errorf("path %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization %q", got)
		}
		answer(w, read(t, r), 4)
	})
	got, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one", "two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d vectors", len(got))
	}
	for i, v := range got {
		var sum float64
		for _, x := range v {
			sum += float64(x) * float64(x)
		}
		if math.Abs(sum-1) > 1e-6 {
			t.Errorf("vector %d has length %v", i, math.Sqrt(sum))
		}
	}
	// The second text was scaled twice as far, and normalising makes the two
	// vectors equal; only their order distinguishes them from a shuffle.
	if got[0][0] >= got[0][1] {
		t.Errorf("dimensions of vector 0 are not increasing: %v", got[0])
	}
}

func TestVectorsOutOfOrderAreRestoredByIndex(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"index":1,"embedding":[0,1]},
			{"index":0,"embedding":[1,0]}]}`)
	})
	got, err := client(t, s.URL, 2).Embed(t.Context(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0][0] != 1 || got[1][1] != 1 {
		t.Errorf("got %v", got)
	}
}

func TestOverloadIsWaitedOutAndRetried(t *testing.T) {
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":{"message":"rate limit"}}`)
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"error":{"message":"engine overloaded"}}`)
		default:
			answer(w, read(t, r), 4)
		}
	})
	if _, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"}); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Errorf("made %d calls, want 3", calls)
	}
}

func TestRetryAfterIsHonoured(t *testing.T) {
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		answer(w, read(t, r), 4)
	})
	if _, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("made %d calls, want 2", calls)
	}
}

func TestOverloadThatNeverClearsIsReported(t *testing.T) {
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	})
	c := client(t, s.URL, 4)
	c.Attempts = 3
	_, err := c.Embed(t.Context(), []string{"one"})
	if err == nil {
		t.Fatal("want an error")
	}
	if calls != 3 {
		t.Errorf("made %d calls, want 3", calls)
	}
	if !strings.Contains(err.Error(), "3 attempts") {
		t.Errorf("error does not say how often it tried: %v", err)
	}
}

func TestARejectedRequestIsNotRetried(t *testing.T) {
	// 400 means the request itself is wrong, and the error is final.
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"too many tokens in the request"}}`)
	})
	_, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"})
	if !errors.Is(err, openai.ErrRejected) {
		t.Fatalf("got %v", err)
	}
	if calls != 1 {
		t.Errorf("made %d calls, want 1", calls)
	}
}

func TestAnUnauthorisedKeyIsNotRetried(t *testing.T) {
	var calls int
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"})
	if !errors.Is(err, openai.ErrRejected) {
		t.Fatalf("got %v", err)
	}
	if calls != 1 {
		t.Errorf("made %d calls, want 1", calls)
	}
}

func TestTheWrongWidthIsRefusedRatherThanStored(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		answer(w, read(t, r), 3)
	})
	_, err := client(t, s.URL, 4).Embed(t.Context(), []string{"one"})
	if err == nil || !strings.Contains(err.Error(), "3 dimensions") {
		t.Fatalf("got %v", err)
	}
}

func TestABatchIsCutByCharacters(t *testing.T) {
	var batches [][]string
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		in := read(t, r)
		batches = append(batches, in.Input)
		answer(w, in, 4)
	})
	t.Setenv(embed.KeyEnvVar, "test-key")
	cfg := embed.Defaults()
	cfg.Model.Dimensions = 4
	// A verse in Devanagari: the same number of texts, far more tokens.
	verse := strings.Repeat("धर्मक्षेत्रे कुरुक्षेत्रे ", 10)
	cfg, service := setServiceModel(t, cfg, func(at *embed.ServiceModel) {
		at.BaseURL, at.Name = s.URL, "test-embed"
		at.BatchCharacters = len([]rune(verse)) * 2
	})
	c, err := openai.New(cfg.GetStoredModel(), service)
	if err != nil {
		t.Fatal(err)
	}
	c.Delay = 0

	got, err := c.Embed(t.Context(), []string{verse, verse, verse, verse, verse})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d vectors", len(got))
	}
	if len(batches) != 3 {
		t.Fatalf("sent %d requests, want 3: %v", len(batches), sizes(batches))
	}
	if want := []int{2, 2, 1}; !slices.Equal(sizes(batches), want) {
		t.Errorf("request sizes %v, want %v", sizes(batches), want)
	}
}

func TestNoTextsIsNoRequest(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("a request was sent for no texts")
	})
	got, err := client(t, s.URL, 4).Embed(t.Context(), nil)
	if err != nil || got != nil {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestAServiceWithoutAKeyIsRefusedBeforeAnyRequest(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "")
	cfg, service := setServiceModel(t, embed.Defaults(), func(at *embed.ServiceModel) {
		at.BaseURL = "http://127.0.0.1:1"
	})
	if _, err := openai.New(cfg.GetStoredModel(), service); !errors.Is(err, openai.ErrNoKey) {
		t.Fatalf("got %v", err)
	}
}

// A gateway that quotes the request back quotes the key back. The error a
// person reads carries what the service said and not the key.
func TestTheKeyIsNotInAnError(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusInternalServerError,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s := server(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprintf(w, `{"error":{"message":"rejected","headers":{"Authorization":%q}}}`,
					r.Header.Get("Authorization"))
			})
			c := client(t, s.URL, 4)
			c.Attempts = 1
			_, err := c.Embed(t.Context(), []string{"one"})
			if err == nil {
				t.Fatal("the request was answered")
			}
			if strings.Contains(err.Error(), "test-key") {
				t.Errorf("the key is in %q", err.Error())
			}
			if !strings.Contains(err.Error(), "rejected") {
				t.Errorf("what the service said is not in %q", err.Error())
			}
		})
	}
}

func TestTheKeyIsNotInWhatTheConfigurationPrints(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-secret")
	_, service := setServiceModel(t, embed.Defaults(), func(*embed.ServiceModel) {})
	if printed := fmt.Sprintf("%v", service); strings.Contains(printed, "sk-secret") {
		t.Errorf("the key is in %q", printed)
	}
}

func sizes(batches [][]string) []int {
	out := make([]int, len(batches))
	for i, b := range batches {
		out[i] = len(b)
	}
	return out
}
