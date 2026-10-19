package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves answers the book service with the token it
// was given.
func TestTheLinkedHandlerAnswersTheBookService(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := numenv1connect.NewBookServiceClient(web.Client(), web.URL)
	req := connect.NewRequest(&numenv1.ExtractBookTextRequest{Data: createPictureEpub(t)})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.ExtractBookText(t.Context(), req)
	if err != nil {
		t.Fatalf("ExtractBookText: %v", err)
	}
	if !strings.Contains(resp.Msg.GetText(), "Hello reader epub") {
		t.Errorf("the text is %q, want the chapter's words", resp.Msg.GetText())
	}
}
