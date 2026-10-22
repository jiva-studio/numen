package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// deck is a vault holding one stencil and one deck.
var deck = map[string]string{
	"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"decks/Words.md": "---\ntype: deck\n---\n" +
		"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
		"\n### Meaning\n\nCompost made of fallen leaves alone\n",
}

// makeWindow is this window as the binary builds it: an installation of its
// own, an index it may write, and two vaults to sit down to.
func makeWindow(
	t *testing.T,
) (container.Config, *container.Index, *container.OpenVaults, *window.API, []domain.Vault) {
	t.Helper()

	state := t.TempDir()
	cfg := container.Config{
		IndexPath:    filepath.Join(state, "index.db"),
		RegistryPath: filepath.Join(state, "vaults.json"),
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	target := []domain.Vault{
		{ID: "one", Name: "One", Path: vaultOf(t, cfg)},
		{ID: "two", Name: "Two", Path: vaultOf(t, cfg)},
	}
	vaults := &container.OpenVaults{
		Config: cfg, Index: db, Lifetime: t.Context(),
		RecordVault: func(domain.Vault) {}, Out: io.Discard,
	}
	t.Cleanup(vaults.CloseAndWait)
	return cfg, db, vaults, &window.API{}, target
}

// vaultOf is a folder holding one stencil and one deck, as a vault.
func vaultOf(t *testing.T, cfg container.Config) string {
	t.Helper()

	root := t.TempDir()
	if _, err := cfg.VaultIdentity().EnsureIdentity(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	for name, body := range deck {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// hasWrittenSecrets is whether this installation has written down a secret or
// an address. A window that serves the tools to the agent it starts itself
// writes neither.
func hasWrittenSecrets(t *testing.T, cfg container.Config) (token bool, announcement bool) {
	t.Helper()

	there := func(at func(container.Config) (string, error)) bool {
		path, err := at(cfg)
		if err != nil {
			t.Fatal(err)
		}
		_, err = os.Stat(path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
		return err == nil
	}
	return there(agents.TokenPath), there(agents.DiscoveryPath)
}

// An installation that names no agent lets nothing be asked, and says so.
func TestAnInstallationNamingNoAgentAsksNothingAboutACard(t *testing.T) {
	cfg, db, opening, api, _ := makeWindow(t)
	cfg.Agent = settings.Agent{Use: settings.UseProgram("")}

	notes := makeNotes(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, api, false, io.Discard)
	t.Cleanup(func() { _ = away() })

	if api.GetAgent() != nil {
		t.Error("a card can be asked about")
	}
	if api.Unreachable.Why() == "" {
		t.Error("expected error reason for missing agent")
	}
}

// The flag shuts it for one launch, whatever the settings name.
func TestTheFlagShutsTheAgentForOneLaunch(t *testing.T) {
	cfg, db, opening, api, _ := makeWindow(t)
	cfg.Agent = settings.Agent{}

	notes := makeNotes(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, api, true, io.Discard)
	t.Cleanup(func() { _ = away() })

	if api.GetAgent() != nil {
		t.Error("a card can be asked about")
	}
	if api.Opened != nil {
		t.Error("a session opening starts an agent")
	}
	if api.Unreachable.Why() != errNoAgentNamed {
		t.Errorf("got error message %q", api.Unreachable.Why())
	}
}

// The file an agent a person runs themselves is configured from names one
// window's vault. This window writes neither it nor a token: a second writer
// would point that agent at whichever window started last.
func TestTheReviewerWritesDownNoAddressAndNoToken(t *testing.T) {
	cfg, db, opening, api, vaults := makeWindow(t)
	cfg.Agent = settings.Agent{}

	notes := makeNotes(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, api, false, io.Discard)
	t.Cleanup(func() { _ = away() })

	api.Opened(t.Context(), vaults[0])

	token, announcement := hasWrittenSecrets(t, cfg)
	if token {
		t.Error("a token was written down")
	}
	if announcement {
		t.Error("an endpoint was announced")
	}
}

// The agent works the vault the person's session is on. A session on another
// vault starts it again there; one on the same vault leaves it where it is.
func TestTheAgentFollowsTheVaultTheSessionIsOn(t *testing.T) {
	cfg, db, opening, api, vaults := makeWindow(t)
	cfg.Agent = settings.Agent{}

	notes := makeNotes(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, api, false, io.Discard)
	t.Cleanup(func() { _ = away() })

	if api.GetAgent() != nil {
		t.Error("a card can be asked about before anybody has sat down")
	}

	api.Opened(t.Context(), vaults[0])
	first := api.GetAgent()
	if first == nil {
		t.Fatal("nothing answers about a card once a session is open")
	}

	api.Opened(t.Context(), vaults[0])
	if api.GetAgent() != first {
		t.Error("a session on the same vault again started the agent over")
	}

	api.Opened(t.Context(), vaults[1])
	if api.GetAgent() == first {
		t.Error("the agent stayed on the vault the person left")
	}
}

// A window that can reach an agent clears what an earlier answer said about why
// it could not.
func TestAReachableAgentClearsWhyItWasUnreachable(t *testing.T) {
	cfg, db, opening, api, _ := makeWindow(t)
	cfg.Agent = settings.Agent{}
	api.Unreachable.Store("an earlier reason")

	notes := makeNotes(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, api, false, io.Discard)
	t.Cleanup(func() { _ = away() })

	if why := api.Unreachable.Why(); why != "" {
		t.Errorf("got error message %q", why)
	}
}

// Taking the agents away stops the endpoint a session started, and the panel
// has nobody to ask.
func TestTakingTheAgentsAwayStopsTheEndpoint(t *testing.T) {
	cfg, db, opening, api, vaults := makeWindow(t)
	cfg.Agent = settings.Agent{}

	notes := makeNotes(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, api, false, io.Discard)
	t.Cleanup(func() { _ = away() })

	api.Opened(t.Context(), vaults[0])
	if api.GetAgent() == nil {
		t.Fatal("nothing answers about a card once a session is open")
	}

	if err := away(); err != nil {
		t.Fatal(err)
	}
	if api.GetAgent() != nil {
		t.Error("a card can still be asked about")
	}
}

// answering is a run that has nothing more to say.
type answering struct{}

func (answering) GetSteps() <-chan port.Step {
	steps := make(chan port.Step)
	close(steps)
	return steps
}

func (answering) Stop() error { return nil }

// listening is an agent that takes a question and ends it.
type listening struct{}

func (listening) StartRun(context.Context, port.Task) (port.Run, error) { return answering{}, nil }

func (listening) EndConversation(context.Context, string) error { return nil }

// The tools the review window serves answer from this window's own vault and notes, and the plugins
// are told what the person last asked about.
func TestTheReviewToolsAreWiredToTheWindow(t *testing.T) {
	cfg, db, opening, api, vaults := makeWindow(t)
	notes := makeNotes(cfg, db, opening)
	v := vaults[0]

	core := makeReviewCore(cfg, db, notes, api, v, v.Path, io.Discard)

	if core.Showing == nil {
		t.Fatal("the tools are not told which vault they work")
	}
	if got := core.Showing(); got.Vault.ID != v.ID || got.Root != v.Path {
		t.Errorf("the tools work %+v", got)
	}
	wired := map[string]bool{
		"problems":    !reflect.ValueOf(core.GetProblems).IsZero(),
		"summary":     !reflect.ValueOf(core.GetSummary).IsZero(),
		"asked":       !reflect.ValueOf(core.GetAsked).IsZero(),
		"read notes":  !reflect.ValueOf(core.Notes.Read).IsZero(),
		"titles":      !reflect.ValueOf(core.Notes.GetTitles).IsZero(),
		"sources":     !reflect.ValueOf(core.Sources.GetDocumentList).IsZero(),
		"named paths": !reflect.ValueOf(core.Notes.GetNamedPaths).IsZero(),
	}
	for name, ok := range wired {
		if !ok {
			t.Errorf("the tools have no %s", name)
		}
	}
	if reflect.ValueOf(core.Read).IsZero() {
		t.Error("the tools read no file")
	}
	if reflect.ValueOf(core.Notes.Search).IsZero() {
		t.Error("the tools search nothing")
	}

	api.SetAgent(listening{})
	route, handler := numenv1connect.NewAgentServiceHandler(api)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	client := numenv1connect.NewAgentServiceClient(server.Client(), server.URL)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{
		Prompt: "why", Focus: "decks/Words.md", Mark: "3f4g5h6j7k", Face: "Back",
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	for stream.Receive() {
	}

	request := core.GetAsked()
	want := map[string]string{"focus": "decks/Words.md", "mark": "3f4g5h6j7k", "face": "Back"}
	if !reflect.DeepEqual(request, want) {
		t.Errorf("the tools are told the person is on %v", request)
	}
}
