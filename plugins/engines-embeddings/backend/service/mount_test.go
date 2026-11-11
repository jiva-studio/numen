package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// The handler of a host that links the plugin in embeds with the base URL and
// the key that its own Configure passes.
func TestTheLinkedHandlerEmbedsThroughTheServiceItsConfigureNames(t *testing.T) {
	var authorization string
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{0.5, 0.25}, "index": 0}}})
	}))
	defer remote.Close()
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	token := hostcall.WithBearer("linked-token")

	_, err = server.NewLifecycleClient(web.Client(), web.URL, token).Configure(t.Context(),
		map[string]string{"baseUrl": remote.URL, "provider": "custom"}, keysOf("remote-key"))
	if err != nil {
		t.Fatalf("Configure: %v", err)
	}
	vectors, err := embedding.NewClient(web.Client(), web.URL, token).EmbedTexts(t.Context(), embedding.RoleDocument, []string{"words"})
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}

	if authorization != "Bearer remote-key" {
		t.Errorf("the remote service was asked with %q", authorization)
	}
	if len(vectors) != 1 || len(vectors[0]) != 2 || vectors[0][0] != 0.5 {
		t.Errorf("vectors = %v", vectors)
	}
}

func TestMountRegistersCloseCallback(t *testing.T) {
	t.Setenv("NUMEN_MOCK_ML", "true")
	s := server.New(server.WithToken("token"))
	if err := Mount(s); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("s.Close: %v", err)
	}
}

func keysOf(values ...string) server.Secrets {
	list := make([]server.Secret, 0, len(values))
	for i, value := range values {
		list = append(list, server.Secret{ID: "s0000000" + string(rune('1'+i)), Name: "Key", Group: "openrouter", Value: value})
	}
	return server.Secrets{server.SecretAPIKey: list}
}
