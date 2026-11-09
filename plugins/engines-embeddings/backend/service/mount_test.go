package service

import (
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// The handler of a host that links the plugin in serves the local model it is configured for.
func TestTheLinkedHandlerServesTheModelItsConfigureNames(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	token := hostcall.WithBearer("linked-token")

	if _, err := server.NewLifecycleClient(web.Client(), web.URL, token).Configure(t.Context(),
		map[string]string{"source": "local", "model": "Xenova/bge-m3"}, nil); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	info, err := embedding.NewClient(web.Client(), web.URL, token).GetModelInfo(t.Context())
	if err != nil {
		t.Fatalf("GetModelInfo: %v", err)
	}

	if info.Name != "Xenova/bge-m3" || info.Dimensions != 1024 {
		t.Errorf("model = %s at %d, want Xenova/bge-m3 at 1024", info.Name, info.Dimensions)
	}
}

// The model is read while Configure changes it.
func TestTheModelIsReadWhileConfigureChangesIt(t *testing.T) {
	srv := newEmbeddingServer()
	isDone := make(chan struct{})
	go func() {
		defer close(isDone)
		for range 200 {
			srv.updateConfig(map[string]string{"model": "bge-m3"}, nil)
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
