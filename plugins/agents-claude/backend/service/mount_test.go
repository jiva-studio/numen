package service

import (
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// The handler a host that links the plugin in serves lists Claude Code, with the token it was given.
func TestTheLinkedHandlerListsClaudeCode(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := agentrun.NewClient(web.Client(), web.URL)
	client.Header.Set("Authorization", "Bearer linked-token")
	programs, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatalf("GetPrograms: %v", err)
	}
	if len(programs) != 1 || programs[0].Name != programName {
		t.Fatalf("programs = %v, want %s alone", programs, programName)
	}
}
