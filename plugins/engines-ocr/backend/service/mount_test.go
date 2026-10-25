package service

import (
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves answers the recognition service with the token
// it was given, and refuses a call without it.
func TestTheLinkedHandlerAnswersTheRecognitionService(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := numenv1connect.NewRecognitionServiceClient(web.Client(), web.URL)

	_, err = client.RecognisePage(t.Context(), connect.NewRequest(&numenv1.RecognisePageRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("RecognisePage without the token = %v, want unauthenticated", err)
	}
	req := connect.NewRequest(&numenv1.RecognisePageRequest{})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.RecognisePage(t.Context(), req)
	if err != nil || len(resp.Msg.GetBlocks()) != 0 {
		t.Fatalf("RecognisePage of no image = %v, %v; want no blocks", resp, err)
	}
}
