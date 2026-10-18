package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// The key in the environment of the plugin is not a key: the models are asked
// for only with the one Configure hands over.
func TestTheModelsAreAskedForOnlyWithTheKeyConfigureHandsOver(t *testing.T) {
	t.Setenv(anthropicKeyVar, "env-key")
	var mu sync.Mutex
	var seen []string
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("x-api-key"))
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m1", "display_name": "Model One"}}})
	}))
	defer endpoint.Close()

	var catalogue modelCatalogue
	catalogue.update(map[string]string{"baseUrl": endpoint.URL})
	if got := catalogue.get(t.Context()); len(got) != 0 {
		t.Errorf("models = %v while Configure handed over no key", got)
	}

	catalogue.update(map[string]string{"apiKey": "host-key"})
	if got := catalogue.get(t.Context()); len(got) != 1 || got[0].Name != "m1" {
		t.Errorf("models = %v, want the one the endpoint lists", got)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 || seen[0] != "host-key" {
		t.Errorf("the endpoint was asked with %v, want only the key Configure handed over", seen)
	}
}
