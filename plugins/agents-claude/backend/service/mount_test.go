package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// The handler a host that links the plugin in serves lists Claude Code, with the token it was given.
func TestTheLinkedHandlerListsClaudeCode(t *testing.T) {
	script := writeScript(t, "claude", "exit 0\n")
	vars := map[string]string{
		"NUMEN_PLUGIN_TOKEN":   "linked-token",
		"NUMEN_CLAUDE_COMMAND": script,
		"PATH":                 t.TempDir(),
	}
	handler, err := NewHandler(vars)
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
	if len(programs) != 1 || programs[0].Name != "claude" {
		t.Fatalf("programs = %v, want claude alone", programs)
	}

	lifecycle := server.NewLifecycleClient(http.DefaultClient, web.URL, hostcall.WithBearer("linked-token"))
	if _, err := lifecycle.Configure(t.Context(), map[string]string{"executablePath": script}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	state, err := client.GetState(t.Context())
	if err != nil || state.Unreachable != "" {
		t.Fatalf("GetState after Configure = (%+v, %v), want reachable", state, err)
	}

	// Verify Mount also wires config updates and registry close on Shutdown.
	mountedHandler, err := server.NewLinkedHandler(vars, Mount)
	if err != nil {
		t.Fatalf("NewLinkedHandler(Mount): %v", err)
	}
	mountedWeb := httptest.NewServer(mountedHandler)
	defer mountedWeb.Close()
	mountedLifecycle := server.NewLifecycleClient(http.DefaultClient, mountedWeb.URL, hostcall.WithBearer("linked-token"))
	if _, err := mountedLifecycle.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
