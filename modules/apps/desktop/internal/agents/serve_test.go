package agents

import (
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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

// isListening reports whether anything takes a connection at this address.
func isListening(addr string) bool {
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
	cfg.Agent = settings.Agent{Use: settings.UseProgram(""), ShouldServeTools: true}

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
	if !isListening(at) {
		t.Errorf("nothing answers at %q", served.URL)
	}
}

// TestWindowWithoutDiscoveryWritesNoFile. The announcement names one
// window's vault, and a second window rewriting it points a person's own agent
// at whichever started last.
func TestWindowWithoutDiscoveryWritesNoFile(t *testing.T) {
	cfg := makeInstallation(t)
	cfg.Agent = settings.Agent{Use: settings.UseProgram(""), ShouldServeTools: true}
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

	served, err := Serve(t.Context(), Options{
		Config: cfg, Core: vault(t), Token: "secret", Root: t.TempDir(), Out: io.Discard,
		Provider: &fakeProvider{},
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
	for isListening(authority(t, served.URL)) {
		if time.Since(waiting) > 5*time.Second {
			t.Fatal("the endpoint never began closing")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := served.Agent.StartRun(t.Context(), port.Task{Question: "anything"}); err == nil {
		t.Error("the agent took work while the endpoint was closing")
	}

	held.Close()
	if err := <-shut; err != nil {
		t.Fatal(err)
	}
}

// The panel's child is told which tools it may call, one by one, and where to reach them. The
// endpoint in front of it serves the whole surface for an agent a person configured themselves,
// and a name absent from the allowance is refused under the mode this runs in.
func TestThePanelsChildIsAllowedTheToolsOfItsSurfaceByName(t *testing.T) {
	cfg := makeInstallation(t)
	provider := &fakeProvider{}

	dir := t.TempDir()
	core := vault(t)
	served, err := Serve(t.Context(), Options{
		Config: cfg, Core: core, Token: "secret", Root: dir, Out: io.Discard, Provider: provider,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = served.Close() })
	if served.Agent == nil {
		t.Fatal("the settings name an agent and none was started")
	}

	work, err := served.Agent.StartRun(t.Context(), port.Task{Question: "what is here?", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	for range work.GetSteps() {
	}
	given := provider.getTasks()[0]

	words, err := mcp.Vocabulary(t.Context(), core)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 {
		t.Fatal("the surface serves no tool, and an allowance naming none would pass")
	}
	var surfaced []string
	for name := range words {
		surfaced = append(surfaced, name)
	}
	slices.Sort(surfaced)
	if !slices.Equal(given.Tools, surfaced) {
		t.Errorf("the allowance is %q, and the surface serves %q", given.Tools, surfaced)
	}
	if len(given.ToolDeclarations) != len(words) {
		t.Errorf("%d tools are declared, and the surface serves %d", len(given.ToolDeclarations), len(words))
	}
	if given.Endpoint.URL != served.URL || given.Endpoint.Token != "secret" || given.Root != dir {
		t.Errorf("the child is given %q at %+v", given.Root, given.Endpoint)
	}
}

// The agent is the router where the settings name one, and nothing where they name none.
func TestTheAgentIsTheRouterWhereTheSettingsNameOne(t *testing.T) {
	for name, agent := range map[string]settings.Agent{
		"left to the default": {},
		"named":               {Use: settings.UseProgram("claude")},
		"named none":          {Use: settings.UseProgram(""), ShouldServeTools: true},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := makeInstallation(t)
			cfg.Agent = agent
			served, err := Serve(t.Context(), Options{
				Config: cfg, Core: vault(t), Token: "secret", Out: io.Discard, Provider: &fakeProvider{},
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = served.Close() })
			if got, want := served.Agent != nil, agent.IsNamed(); got != want {
				t.Errorf("an agent is started: %v, and the settings name one: %v", got, want)
			}
		})
	}
}
