//go:build !nomcp

package agents

import (
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// makeInstallation is an installation whose own state is this test's alone.
func makeInstallation(t *testing.T) container.Config {
	t.Helper()
	return container.Config{
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
}

// vault is a Core answering about one vault and holding nothing.
func vault(t *testing.T) mcp.Core {
	t.Helper()
	return mcp.Core{Showing: mcp.ShowOneVault(domain.Vault{ID: "one", Name: "one"}, t.TempDir())}
}

// authority is the host and port an endpoint is reached at.
func authority(t *testing.T, endpoint string) string {
	t.Helper()

	said, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	return said.Host
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

// TestTheTokenIsCreatedOnceAndKept. The line an agent is configured with is
// written down, so it still names this installation the next time it is asked
// for.
func TestTheTokenIsCreatedOnceAndKept(t *testing.T) {
	cfg := makeInstallation(t)

	token, err := GetToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("nothing was created")
	}
	again, err := GetToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if again != token {
		t.Errorf("the token was %q and is now %q", token, again)
	}
}

// TestATokenCreatedForAWindowIsKeptNowhere. A window nobody configures an agent
// against is reached for as long as it is open, and writes down nothing another
// window would read.
func TestATokenCreatedForAWindowIsKeptNowhere(t *testing.T) {
	cfg := makeInstallation(t)

	token, err := CreateToken()
	if err != nil {
		t.Fatal(err)
	}
	again, err := CreateToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || again == token {
		t.Errorf("the token was created as %q and again as %q", token, again)
	}

	left, err := os.ReadDir(filepath.Dir(cfg.RegistryPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range left {
		t.Errorf("%s was written where the application keeps its own state", entry.Name())
	}
}

// TestAnEphemeralPortIsNamedByWhatItBoundTo. A window served on a port the
// machine picks hands back the port it was given, which is what an agent is
// told to reach.
func TestAnEphemeralPortIsNamedByWhatItBoundTo(t *testing.T) {
	cfg := makeInstallation(t)
	cfg.Agent = agent.Config{ShouldServeTools: true}

	served, err := Serve(t.Context(), Options{
		Config: cfg, Core: vault(t), Token: "secret", Out: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = served.Close() })

	at := authority(t, served.URL)
	if _, port, _ := net.SplitHostPort(at); port == "0" || port == "" {
		t.Fatalf("the endpoint is announced at %q", served.URL)
	}
	if !listens(at) {
		t.Errorf("nothing answers at %q", served.URL)
	}
}

// TestAWindowThatDoesNotAnnounceWritesNothing. The announcement names one
// window's vault, and a second window rewriting it points a person's own agent
// at whichever started last.
func TestAWindowThatDoesNotAnnounceWritesNothing(t *testing.T) {
	cfg := makeInstallation(t)
	cfg.Agent = agent.Config{ShouldServeTools: true}
	state := filepath.Dir(cfg.RegistryPath)

	served, err := Serve(t.Context(), Options{
		Config: cfg, Core: vault(t), Token: "secret", Out: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = served.Close() })

	left, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range left {
		t.Errorf("%s was written where the application keeps its own state", entry.Name())
	}
}

// TestTheAgentsGoBeforeTheEndpoint. An agent still answering goes on writing to
// the vault, so it is stopped while the tools it writes through are still
// there.
func TestTheAgentsGoBeforeTheEndpoint(t *testing.T) {
	cfg := makeInstallation(t)
	cfg.Agent = agent.Defaults()
	cfg.Agent.Claude.Command = []string{"/bin/sh", "-c", "exit 0"}

	served, err := Serve(t.Context(), Options{
		Config: cfg, Core: vault(t), Token: "secret", Root: t.TempDir(), Out: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if served.Agent == nil {
		t.Fatal("the settings name an agent and none was started")
	}

	// A request left half-written holds the endpoint open, so the close is
	// still in front of it while the agent is asked for work.
	held, err := net.Dial("tcp", authority(t, served.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := held.Write([]byte("GET /mcp HTTP/1.1\r\nHost: numen\r\n")); err != nil {
		t.Fatal(err)
	}

	shut := make(chan error, 1)
	go func() { shut <- served.Close() }()

	// The endpoint stops taking connections as it begins closing, which is
	// where the agents are already gone.
	waiting := time.Now()
	for listens(authority(t, served.URL)) {
		if time.Since(waiting) > 5*time.Second {
			t.Fatal("the endpoint never began closing")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := served.Agent.Take(t.Context(), port.Task{Question: "anything"}); err == nil {
		t.Error("the agent took work while the endpoint was closing")
	}

	held.Close()
	if err := <-shut; err != nil {
		t.Fatal(err)
	}
}

// The panel's child is told which tools it may call, one by one. The endpoint
// in front of it serves the whole surface for an agent a person configured
// themselves, and a name absent from the allowance is refused under the mode
// this runs in.
func TestThePanelsChildIsAllowedTheToolsOfItsSurfaceByName(t *testing.T) {
	cfg := makeInstallation(t)
	cfg.Agent = agent.Defaults()

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	written := filepath.Join(dir, "argv")
	body := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + written + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg.Agent.Claude.Command = []string{script}

	core := vault(t)
	served, err := Serve(t.Context(), Options{
		Config: cfg, Core: core, Token: "secret", Root: dir, Out: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = served.Close() })
	if served.Agent == nil {
		t.Fatal("the settings name an agent and none was started")
	}

	work, err := served.Agent.Take(t.Context(), port.Task{Question: "what is here?"})
	if err != nil {
		t.Fatal(err)
	}
	for range work.Steps() {
	}
	raw, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")

	allowed := strings.Split(getFlagValue(t, argv, "--allowedTools"), ",")
	if slices.Contains(allowed, claudecode.Tool("*")) {
		t.Fatalf("the allowance is %q", allowed)
	}
	words, err := mcp.Vocabulary(t.Context(), core)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 {
		t.Fatal("the surface serves no tool, and an allowance naming none would pass")
	}
	for name := range words {
		if !slices.Contains(allowed, claudecode.Tool(name)) {
			t.Errorf("%s is served and is not in the allowance %q", name, allowed)
		}
	}
	// The search stands beside them, and nothing else the command line brings.
	if !slices.Contains(allowed, "WebSearch") {
		t.Errorf("the allowance is %q", allowed)
	}
	if len(allowed) != len(words)+1 {
		t.Errorf("the allowance is %q, and the surface serves %d tools", allowed, len(words))
	}
}

// getFlagValue is what one flag on a command line was given.
func getFlagValue(t *testing.T, argv []string, flag string) string {
	t.Helper()
	for i := len(argv) - 2; i >= 0; i-- {
		if argv[i] == flag {
			return argv[i+1]
		}
	}
	t.Fatalf("nothing names %s: %q", flag, argv)
	return ""
}
