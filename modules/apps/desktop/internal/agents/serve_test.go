package agents

import (
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// makeInstallation is an installation whose own state is this test's alone.
func makeInstallation(t *testing.T) container.Config {
	t.Helper()
	return container.Config{
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
}

// authority is the host and port an endpoint is reached at.
func authority(t *testing.T, endpoint string) string {
	t.Helper()

	response, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	return response.Host
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

// TestTokenCreation_Cached. The line an agent is configured with is
// written down, so it still names this installation the next time it is asked
// for.
func TestTokenCreation_Cached(t *testing.T) {
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

// TestWindowToken_NotRetained. A window nobody configures an agent
// against is reached for as long as it is open, and writes down nothing another
// window would read.
func TestWindowToken_NotRetained(t *testing.T) {
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

	remaining, err := os.ReadDir(filepath.Dir(cfg.RegistryPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range remaining {
		t.Errorf("%s was written where the application keeps its own state", entry.Name())
	}
}

// TestAnEphemeralPortIsNamedByWhatItBoundTo. A window served on a port the
// machine picks returns the port it was given, which is what an agent is
// told to reach.
func TestAnEphemeralPortIsNamedByWhatItBoundTo(t *testing.T) {
	cfg := makeInstallation(t)
	cfg.Agent = settings.Agent{Use: settings.UseProgram(""), ShouldServeTools: true}

	served, err := Serve(t.Context(), Options{
		Config: cfg, Token: "secret", Out: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = served.Close() })

	pos := authority(t, served.URL)
	if _, port, _ := net.SplitHostPort(pos); port == "0" || port == "" {
		t.Fatalf("the endpoint is announced at %q", served.URL)
	}
	if !isListening(pos) {
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
		Config: cfg, Token: "secret", Out: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = served.Close() })

	remaining, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range remaining {
		t.Errorf("%s was written where the application keeps its own state", entry.Name())
	}
}
