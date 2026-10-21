package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The key in the environment of the plugin is not a key: the remote service is
// asked only with the one Configure hands over.
func TestTheRemoteServiceIsAskedOnlyWithTheKeyConfigurePassesOver(t *testing.T) {
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
		if _, err := life.Configure(t.Context(), connect.NewRequest(&numenv1.ConfigureRequest{Values: values})); err != nil {
			t.Fatal(err)
		}
	}

	configure(map[string]string{"baseUrl": service.URL})
	if key, _ := emb.getService(); key != "" {
		t.Fatalf("key = %q while Configure handed over none", key)
	}

	configure(map[string]string{server.SecretAPIKey: "host-key"})
	res, err := emb.EmbedBatch(t.Context(), connect.NewRequest(&numenv1.EmbedBatchRequest{Texts: []string{"a fixture sentence"}}))
	if err != nil {
		t.Fatal(err)
	}
	vectors := res.Msg.GetVectors()
	if len(vectors) != 1 || len(vectors[0].GetValues()) != 2 || vectors[0].GetValues()[0] != 0.5 {
		t.Errorf("vectors = %v, want the one the service answered", vectors)
	}
	if len(auth) != 1 || auth[0] != "Bearer host-key" {
		t.Errorf("the service was asked with %v, want only the key Configure handed over", auth)
	}

	configure(map[string]string{server.SecretAPIKey: ""})
	if key, _ := emb.getService(); key != "" {
		t.Errorf("key = %q after Configure dropped it", key)
	}
}
