package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// The handler a host that links the plugin in serves embeds with the base URL and the key its own
// Configure hands over.
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
	token := connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer linked-token")
			return next(ctx, req)
		}
	}))

	_, err = server.NewPluginLifecycleServiceClient(web.Client(), web.URL, token).Configure(t.Context(),
		connect.NewRequest(&server.ConfigureRequest{Values: map[string]string{"baseUrl": remote.URL, "apiKey": "remote-key"}}))
	if err != nil {
		t.Fatalf("Configure: %v", err)
	}
	resp, err := embedding.NewEmbeddingServiceClient(web.Client(), web.URL, token).EmbedBatch(t.Context(),
		connect.NewRequest(&embedding.EmbedBatchRequest{Texts: []string{"words"}}))
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}

	if authorization != "Bearer remote-key" {
		t.Errorf("the remote service was asked with %q", authorization)
	}
	vectors := resp.Msg.GetVectors()
	if len(vectors) != 1 || len(vectors[0].GetValues()) != 2 || vectors[0].GetValues()[0] != 0.5 {
		t.Errorf("vectors = %v", vectors)
	}
}

// The model is read while Configure changes it.
func TestTheModelIsReadWhileConfigureChangesIt(t *testing.T) {
	srv := newEmbeddingServer()
	isDone := make(chan struct{})
	go func() {
		defer close(isDone)
		for range 200 {
			srv.updateConfig(map[string]string{"model": "other-model"})
		}
	}()

	for range 200 {
		if _, err := srv.GetModelInfo(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	<-isDone
}

// A dimension or a token limit the model answer cannot hold is not taken from the environment.
func TestAnOverrideAboveWhatTheAnswerContainsIsIgnored(t *testing.T) {
	want := newEmbeddingServer().modelInfo
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "4294967296")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "4294967296")

	got := newEmbeddingServer().modelInfo

	if got.Dimensions != want.Dimensions || got.MaxTokens != want.MaxTokens {
		t.Fatalf("dimensions %d and tokens %d, want %d and %d", got.Dimensions, got.MaxTokens, want.Dimensions, want.MaxTokens)
	}
}
