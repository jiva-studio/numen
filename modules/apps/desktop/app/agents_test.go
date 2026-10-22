package app

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// and for no tools has nothing listening and no token kept beside the vault
// list.
func TestAnInstallationNamingNoAgentOpensNoPort(t *testing.T) {
	opened, cfg := windowOn(t)
	cfg.Agent = settings.Agent{Use: settings.UseProgram("")}
	addr := free(t)

	shut, err := serveAgents(t.Context(), cfg, opened, agentOptions{addr: addr}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = shut() })

	if isListening(addr) {
		t.Error("the tools are on a port")
	}
	if hasToken(t, cfg) {
		t.Error("a token was written")
	}
	if _, content := readAnnouncement(t, cfg); content {
		t.Error("an endpoint was announced")
	}
	if opened.State.Unreachable.Why() == "" {
		t.Error("expected error reason for missing agent")
	}
}

// TestTheAgentTheSettingsNameIsServedAndTakenAway. The endpoint answers while
// the window is up and the announcement points at it, and both go on the way
// out.
func TestTheAgentTheSettingsNameIsServedAndTakenAway(t *testing.T) {
	opened, cfg := windowOn(t)
	cfg.Agent = settings.Agent{}
	addr := free(t)

	shut, err := serveAgents(t.Context(), cfg, opened, agentOptions{addr: addr}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if !isListening(addr) {
		t.Error("no service responding at agent endpoint")
	}
	if !hasToken(t, cfg) {
		t.Error("no token was kept")
	}
	response, content := readAnnouncement(t, cfg)
	if !content {
		t.Fatal("nothing says where the endpoint is")
	}
	if want := "http://" + addr + "/mcp"; response.URL != want {
		t.Errorf("the announcement points at %q, and the endpoint is at %q", response.URL, want)
	}
	if response.Token == "" {
		t.Error("the announcement carries no token")
	}

	if err := shut(); err != nil {
		t.Fatal(err)
	}
	if isListening(addr) {
		t.Error("the port is still open")
	}
	if _, remaining := readAnnouncement(t, cfg); remaining {
		t.Error("expected announcement to be removed")
	}
}

// TestTheToolsAreServedToAnAgentAPersonRunsThemselves. The setting opens the
// endpoint on its own, and the panel is told there is no agent to ask.
func TestTheToolsAreServedToAnAgentAPersonRunsThemselves(t *testing.T) {
	opened, cfg := windowOn(t)
	cfg.Agent = settings.Agent{Use: settings.UseProgram(""), ShouldServeTools: true}
	addr := free(t)

	shut, err := serveAgents(t.Context(), cfg, opened, agentOptions{addr: addr}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if !isListening(addr) {
		t.Error("no service responding at agent endpoint")
	}
	response, content := readAnnouncement(t, cfg)
	if !content || response.Token == "" {
		t.Fatal("an agent has nothing to be configured from")
	}

	if err := shut(); err != nil {
		t.Fatal(err)
	}
	if isListening(addr) {
		t.Error("the port is still open")
	}
	if _, remaining := readAnnouncement(t, cfg); remaining {
		t.Error("expected announcement to be removed")
	}
}

// free is an address on this machine nothing holds. The listener is opened to
// find one the machine is not already using, and closed again.
func free(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

// isListening reports whether anything takes a connection at this address.
func isListening(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// readAnnouncement is what an agent would be configured from, and whether the
// file is there at all.
func readAnnouncement(t *testing.T, cfg container.Config) (agents.Discovery, bool) {
	t.Helper()

	path, err := agents.DiscoveryPath(cfg)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return agents.Discovery{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	var response agents.Discovery
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	return response, true
}

// hasToken is whether the token an agent presents has been written down.
func hasToken(t *testing.T, cfg container.Config) bool {
	t.Helper()

	path, err := agents.TokenPath(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(path)
	return err == nil
}
