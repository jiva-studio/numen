package serving

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

// The key in the environment of the plugin is not a key: the remote service is
// queried only with the one Configure supplies.
func TestConfigure_UsesConfiguredKeyNotEnvironment(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("NUMEN_EMBEDDING_KEY", "env-key")
	var auth []string
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.Header.Get("Authorization"))
		encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{0.5, 0.25}, "index": 0}}})
	}))
	defer service.Close()

	emb := newEmbeddingServer()
	life := server.NewLifecycleServer(make(chan struct{}), server.WithLifecycleUpdateConfig(emb.updateConfig))
	configure := func(values map[string]string, secrets server.Secrets) {
		t.Helper()
		if _, err := life.ApplyConfig(t.Context(), values, secrets); err != nil {
			t.Fatal(err)
		}
	}

	configure(map[string]string{"provider": "custom", "baseUrl": service.URL}, nil)
	if key, _ := emb.getService(); key != "" {
		t.Fatalf("key = %q while Configure passed none", key)
	}

	configure(map[string]string{"provider": "custom", "baseUrl": service.URL}, keysOf("host-key"))
	vectors, err := emb.EmbedTexts(t.Context(), engine.RoleDocument, []string{"a fixture sentence"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 1 || len(vectors[0]) != 2 || vectors[0][0] != 0.5 {
		t.Errorf("vectors = %v, want the one the service answered", vectors)
	}
	if len(auth) != 1 || auth[0] != "Bearer host-key" {
		t.Errorf("the service was asked with %v, want only the key Configure passed", auth)
	}

	configure(map[string]string{"provider": "custom", "baseUrl": service.URL}, keysOf())
	if key, _ := emb.getService(); key != "" {
		t.Errorf("key = %q after Configure passed an empty list", key)
	}
}

// The plugin reads the first secret of the list the host delivers and ignores the others.
func TestTheFirstSecretOfTheListIsTheKeyOfTheRemoteService(t *testing.T) {
	emb := newEmbeddingServer()
	emb.updateConfig(map[string]string{"provider": "custom", "baseUrl": "https://embeddings.example.org/v1"},
		keysOf("first-key", "second-key"))
	if key, _ := emb.getService(); key != "first-key" {
		t.Errorf("key = %q, want the first secret", key)
	}
}

// A call that does not carry the declaration leaves the key as it was, and a call that carries it empty drops it.
func TestACallWithoutTheDeclarationKeepsTheKeyAndAnEmptyListDropsIt(t *testing.T) {
	emb := newEmbeddingServer()
	values := map[string]string{"provider": "custom", "baseUrl": "https://embeddings.example.org/v1"}
	emb.updateConfig(values, keysOf("held-key"))

	emb.updateConfig(values, server.Secrets{})
	if key, _ := emb.getService(); key != "held-key" {
		t.Errorf("key = %q after a call that does not carry the declaration, want held-key", key)
	}
	emb.updateConfig(values, keysOf())
	if key, _ := emb.getService(); key != "" {
		t.Errorf("key = %q after an empty list, want none", key)
	}
}

func TestProviderSetting(t *testing.T) {
	emb := newEmbeddingServer()
	emb.updateConfig(map[string]string{
		"baseUrl":  "https://custom.com",
		"provider": "local",
	}, keysOf("key"))
	if key, url := emb.getService(); key != "" || url != "" {
		t.Errorf("expected local provider to disable service, got %q, %q", key, url)
	}

	emb.updateConfig(map[string]string{
		"provider": "openrouter",
		"baseUrl":  "",
	}, nil)
	if _, url := emb.getService(); url != "https://openrouter.ai/api/v1" {
		t.Errorf("expected openrouter default URL, got %q", url)
	}

	emb.updateConfig(map[string]string{
		"provider": "openai",
		"baseUrl":  "",
	}, nil)
	if _, url := emb.getService(); url != "https://api.openai.com/v1" {
		t.Errorf("expected openai default URL, got %q", url)
	}
}

// The backend takes the settings of the provider chosen: the URL of a custom service only while it
// is chosen, the model of a remote service only while one is chosen, and the local model when neither is chosen.
func TestOnlyTheSettingsOfTheChosenProviderAreTaken(t *testing.T) {
	emb := newEmbeddingServer()
	local := emb.getModelName()
	emb.updateConfig(map[string]string{"provider": "custom", "baseUrl": "https://embeddings.example.org/v1", "model": "fixture-embed"}, keysOf("key"))
	if key, url := emb.getService(); key != "key" || url != "https://embeddings.example.org/v1" || emb.getModelName() != "fixture-embed" {
		t.Errorf("custom: %q, %q, %q", key, url, emb.getModelName())
	}

	emb.updateConfig(map[string]string{"provider": "openai", "model": "fixture-embed"}, nil)
	if _, url := emb.getService(); url != "https://api.openai.com/v1" {
		t.Errorf("openai asks %q, a URL of another provider", url)
	}

	emb.updateConfig(map[string]string{"provider": "local"}, nil)
	if key, url := emb.getService(); key != "" || url != "" || emb.getModelName() != local {
		t.Errorf("local: %q, %q, model %q, want the local model %q", key, url, emb.getModelName(), local)
	}
}
