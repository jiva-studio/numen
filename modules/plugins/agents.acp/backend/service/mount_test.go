package service

import (
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves answers the agent service with the token it
// was given.
func TestTheLinkedHandlerAnswersTheAgentService(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := numenv1connect.NewAgentServiceClient(web.Client(), web.URL)

	req := connect.NewRequest(&numenv1.GetAgentProgramsRequest{})
	req.Header().Set("Authorization", "Bearer linked-token")
	if _, err := client.GetAgentPrograms(t.Context(), req); err != nil {
		t.Fatalf("call with the token: %v", err)
	}

	unauthReq := connect.NewRequest(&numenv1.GetAgentProgramsRequest{})
	_, err = client.GetAgentPrograms(t.Context(), unauthReq)
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("call without the token = %v, want unauthenticated", err)
	}
}
