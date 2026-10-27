package service

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
)

// The handler a host that links the plugin in serves lists Claude Code, with the token it was given.
func TestTheLinkedHandlerListsClaudeCode(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := agentrun.NewAgentServiceClient(web.Client(), web.URL)
	req := connect.NewRequest(&agentrun.GetAgentProgramsRequest{})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.GetAgentPrograms(t.Context(), req)
	if err != nil {
		t.Fatalf("GetAgentPrograms: %v", err)
	}
	if programs := resp.Msg.GetPrograms(); len(programs) != 1 || programs[0].GetName() != programName {
		t.Fatalf("programs = %v, want %s alone", programs, programName)
	}
}
