//go:build !nomcp

package main

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
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// TestAnInstallationNamingNoAgentOpensNoPort. A person who asked for no agent
// and for no tools has nothing listening and no token kept beside the vault
// list.
func TestAnInstallationNamingNoAgentOpensNoPort(t *testing.T) {
	opened, cfg := windowOn(t)
	cfg.Agent = agent.Config{}
	addr := free(t)

	shut, err := serveAgents(t.Context(), cfg, opened, agentOptions{addr: addr}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = shut() })

	if listens(addr) {
		t.Error("the tools are on a port")
	}
	if hasToken(t, cfg) {
		t.Error("a token was written")
	}
	if _, written := readAnnouncement(t, cfg); written {
		t.Error("an endpoint was announced")
	}
	if opened.API.Unreachable.Why() == "" {
		t.Error("the panel was told nothing about why it has no agent")
	}
}

// TestTheAgentTheSettingsNameIsServedAndTakenAway. The endpoint answers while
// the window is up and the announcement points at it, and both go on the way
// out.
func TestTheAgentTheSettingsNameIsServedAndTakenAway(t *testing.T) {
	opened, cfg := windowOn(t)
	cfg.Agent = agent.Defaults()
	addr := free(t)

	shut, err := serveAgents(t.Context(), cfg, opened, agentOptions{addr: addr}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if !listens(addr) {
		t.Error("nothing answers where the agents are told to look")
	}
	if !hasToken(t, cfg) {
		t.Error("no token was kept")
	}
	said, written := readAnnouncement(t, cfg)
	if !written {
		t.Fatal("nothing says where the endpoint is")
	}
	if want := "http://" + addr + "/mcp"; said.URL != want {
		t.Errorf("the announcement points at %q, and the endpoint is at %q", said.URL, want)
	}
	if said.Token == "" {
		t.Error("the announcement carries no token")
	}
	if opened.API.Answering() == nil {
		t.Error("the panel has no agent to ask")
	}

	if err := shut(); err != nil {
		t.Fatal(err)
	}
	if listens(addr) {
		t.Error("the port is still open")
	}
	if _, left := readAnnouncement(t, cfg); left {
		t.Error("the announcement was left behind")
	}
}

// TestTheToolsAreServedToAnAgentAPersonRunsThemselves. The setting opens the
// endpoint on its own, and the panel is told there is no agent to ask.
func TestTheToolsAreServedToAnAgentAPersonRunsThemselves(t *testing.T) {
	opened, cfg := windowOn(t)
	cfg.Agent = agent.Config{ServeTools: true}
	addr := free(t)

	shut, err := serveAgents(t.Context(), cfg, opened, agentOptions{addr: addr}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if !listens(addr) {
		t.Error("nothing answers where the agents are told to look")
	}
	said, written := readAnnouncement(t, cfg)
	if !written || said.Token == "" {
		t.Fatal("an agent has nothing to be configured from")
	}
	if opened.API.Answering() != nil {
		t.Error("the panel was given an agent the settings do not name")
	}
	if opened.API.Unreachable.Why() == "" {
		t.Error("the panel was told nothing about why it has no agent")
	}

	if err := shut(); err != nil {
		t.Fatal(err)
	}
	if listens(addr) {
		t.Error("the port is still open")
	}
	if _, left := readAnnouncement(t, cfg); left {
		t.Error("the announcement was left behind")
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

// listens is whether anything takes a connection at this address.
func listens(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// readAnnouncement is what an agent would be configured from, and whether the
// file is there at all.
func readAnnouncement(t *testing.T, cfg container.Config) (agents.Announcement, bool) {
	t.Helper()

	path, err := agents.AnnouncementPath(cfg)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return agents.Announcement{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	var said agents.Announcement
	if err := json.Unmarshal(raw, &said); err != nil {
		t.Fatal(err)
	}
	return said, true
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
