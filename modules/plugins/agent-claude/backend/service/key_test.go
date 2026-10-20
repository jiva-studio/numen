package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestConfigureKeepsTheKeyTheHostSends(t *testing.T) {
	svc := &agentServer{}
	svc.applyConfig(map[string]string{"apiKey": "k1", "baseUrl": "http://127.0.0.1:1"})
	if key, base := svc.catalogue.credentials(); key != "k1" || base != "http://127.0.0.1:1" {
		t.Errorf("got %q %q", key, base)
	}
	svc.applyConfig(map[string]string{"apiKey": ""})
	if key, _ := svc.catalogue.credentials(); key != "" {
		t.Errorf("the key was kept after the host cleared it: %q", key)
	}
}

func TestTheConfiguredKeyReplacesTheHeldKeyInTheEnvironment(t *testing.T) {
	got := buildKeyEnvironment([]string{"A=1", "ANTHROPIC_API_KEY=old", "ANTHROPIC_BASE_URL=old", "CLAUDECODE=1"}, "new", "")
	want := []string{"A=1", "ANTHROPIC_API_KEY=new"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v", got)
	}
	held := buildKeyEnvironment([]string{"A=1", "ANTHROPIC_API_KEY=old", "CLAUDECODE=1"}, "", "")
	if strings.Join(held, ",") != "A=1,ANTHROPIC_API_KEY=old" {
		t.Errorf("with no key configured, got %v", held)
	}
}
