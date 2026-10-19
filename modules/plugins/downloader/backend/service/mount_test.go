package service

import (
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves answers the download service with the token it
// was given, and refuses a call without it.
func TestTheLinkedHandlerAnswersTheDownloadService(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := numenv1connect.NewDownloadServiceClient(web.Client(), web.URL)

	refused := connect.NewRequest(&numenv1.CanHandleRequest{Url: "https://example.org/page"})
	if _, err := client.CanHandle(t.Context(), refused); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("CanHandle without the token = %v, want unauthenticated", err)
	}
	req := connect.NewRequest(&numenv1.CanHandleRequest{Url: "https://example.org/page"})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.CanHandle(t.Context(), req)
	if err != nil {
		t.Fatalf("CanHandle: %v", err)
	}
	if !resp.Msg.GetCanHandle() {
		t.Error("a web page is not handled")
	}
}
