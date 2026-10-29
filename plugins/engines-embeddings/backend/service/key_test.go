package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// The key in the environment of the plugin is not a key: the remote service is
// asked only with the one Configure hands over.
func TestConfigure_UsesConfiguredKeyNotEnvironment(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("NUMEN_EMBEDDING_KEY", "env-key")
	var auth []string
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"embedding": []float32{0.5, 0.25}, "index": 0}}})
	}))
	defer service.Close()

	emb := newEmbeddingServer()
	life := server.NewLifecycleServer(make(chan struct{}), server.WithLifecycleUpdateConfig(emb.updateConfig))
	configure := func(values map[string]string) {
		t.Helper()
		if _, err := life.ApplyConfig(t.Context(), values); err != nil {
			t.Fatal(err)
		}
	}

	configure(map[string]string{"provider": "custom", "baseUrl": service.URL})
	if key, _ := emb.getService(); key != "" {
		t.Fatalf("key = %q while Configure handed over none", key)
	}

	configure(map[string]string{"provider": "custom", "baseUrl": service.URL, server.SecretAPIKey: "host-key"})
	vectors, err := emb.EmbedTexts(t.Context(), []string{"a fixture sentence"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 1 || len(vectors[0]) != 2 || vectors[0][0] != 0.5 {
		t.Errorf("vectors = %v, want the one the service answered", vectors)
	}
	if len(auth) != 1 || auth[0] != "Bearer host-key" {
		t.Errorf("the service was asked with %v, want only the key Configure handed over", auth)
	}

	configure(map[string]string{"provider": "custom", "baseUrl": service.URL, server.SecretAPIKey: ""})
	if key, _ := emb.getService(); key != "" {
		t.Errorf("key = %q after Configure dropped it", key)
	}
}

func TestProviderSetting(t *testing.T) {
	emb := newEmbeddingServer()
	emb.updateConfig(map[string]string{
		server.SecretAPIKey: "key",
		"baseUrl":           "https://custom.com",
		"provider":          "local",
	})
	if key, url := emb.getService(); key != "" || url != "" {
		t.Errorf("expected local provider to disable service, got %q, %q", key, url)
	}

	emb.updateConfig(map[string]string{
		"provider": "openrouter",
		"baseUrl":  "",
	})
	if _, url := emb.getService(); url != "https://openrouter.ai/api/v1" {
		t.Errorf("expected openrouter default URL, got %q", url)
	}

	emb.updateConfig(map[string]string{
		"provider": "openai",
		"baseUrl":  "",
	})
	if _, url := emb.getService(); url != "https://api.openai.com/v1" {
		t.Errorf("expected openai default URL, got %q", url)
	}
}

// The backend takes the settings of the provider chosen: the URL of a custom service only while it
// is chosen, the model of a remote service only while one is chosen, and the local model otherwise.
func TestOnlyTheSettingsOfTheChosenProviderAreTaken(t *testing.T) {
	emb := newEmbeddingServer()
	local := emb.getModelName()
	emb.updateConfig(map[string]string{"provider": "custom", "baseUrl": "https://embeddings.example.org/v1", "model": "fixture-embed", server.SecretAPIKey: "key"})
	if key, url := emb.getService(); key != "key" || url != "https://embeddings.example.org/v1" || emb.getModelName() != "fixture-embed" {
		t.Errorf("custom: %q, %q, %q", key, url, emb.getModelName())
	}

	emb.updateConfig(map[string]string{"provider": "openai", "model": "fixture-embed"})
	if _, url := emb.getService(); url != "https://api.openai.com/v1" {
		t.Errorf("openai asks %q, a URL of another provider", url)
	}

	emb.updateConfig(map[string]string{"provider": "local"})
	if key, url := emb.getService(); key != "" || url != "" || emb.getModelName() != local {
		t.Errorf("local: %q, %q, model %q, want the local model %q", key, url, emb.getModelName(), local)
	}
}
