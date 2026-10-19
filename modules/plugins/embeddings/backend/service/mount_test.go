package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves embeds with the base URL and the key its own
// Configure hands over.
func TestTheLinkedHandlerEmbedsThroughTheServiceItsConfigureNames(t *testing.T) {
	var authorization string
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"embedding": []float32{0.5, 0.25}, "index": 0}}})
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

	_, err = numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL, token).Configure(t.Context(),
		connect.NewRequest(&numenv1.ConfigureRequest{Values: map[string]string{"baseUrl": remote.URL, "apiKey": "remote-key"}}))
	if err != nil {
		t.Fatalf("Configure: %v", err)
	}
	resp, err := numenv1connect.NewEmbeddingServiceClient(web.Client(), web.URL, token).EmbedText(t.Context(),
		connect.NewRequest(&numenv1.EmbedTextRequest{Text: "words"}))
	if err != nil {
		t.Fatalf("EmbedText: %v", err)
	}

	if authorization != "Bearer remote-key" {
		t.Errorf("the remote service was asked with %q", authorization)
	}
	if got := resp.Msg.GetVector(); len(got) != 2 || got[0] != 0.5 {
		t.Errorf("vector = %v", got)
	}
}
