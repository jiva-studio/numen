package service

import (
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
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
	client := agentrun.NewClient(web.Client(), web.URL)

	client.Header.Set("Authorization", "Bearer linked-token")
	if _, err := client.GetPrograms(t.Context(), nil); err != nil {
		t.Fatalf("call with the token: %v", err)
	}

	client.Header.Del("Authorization")
	_, err = client.GetPrograms(t.Context(), nil)
	if rpcerror.GetCode(err) != rpcerror.Unauthenticated {
		t.Fatalf("call without the token = %v, want unauthenticated", err)
	}
}
