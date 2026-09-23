//go:build !nomcp

package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
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
) (container.Config, *container.Index, *openVaults, *window.API, []domain.Vault) {
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

	held := []domain.Vault{
		{ID: "one", Name: "One", Path: vaultOf(t, cfg)},
		{ID: "two", Name: "Two", Path: vaultOf(t, cfg)},
	}
	vaults := &openVaults{
		cfg: cfg, db: db, under: t.Context(),
		recordVault: func(domain.Vault) {}, out: io.Discard,
	}
	t.Cleanup(vaults.wait)
	return cfg, db, vaults, &window.API{}, held
}

// vaultOf is a folder holding one stencil and one deck, as a vault.
func vaultOf(t *testing.T, cfg container.Config) string {
	t.Helper()

	root := t.TempDir()
	if _, err := cfg.VaultIdentity().Ensure(root, time.Now()); err != nil {
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
	return there(agents.TokenPath), there(agents.AnnouncementPath)
}

// An installation that names no agent lets nothing be asked, and says so.
func TestAnInstallationNamingNoAgentAsksNothingAboutACard(t *testing.T) {
	cfg, db, opening, api, _ := makeWindow(t)
	cfg.Agent = agent.Config{}

	notes, cutting := makeNotesAndCards(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, cutting, api, false, io.Discard)
	t.Cleanup(func() { _ = away() })

	if api.GetAgent() != nil {
		t.Error("a card can be asked about")
	}
	if api.Unreachable.Why() == "" {
		t.Error("the page was told nothing about why it has no agent")
	}
}

// The flag shuts it for one launch, whatever the settings name.
func TestTheFlagShutsTheAgentForOneLaunch(t *testing.T) {
	cfg, db, opening, api, _ := makeWindow(t)
	cfg.Agent = agent.Defaults()

	notes, cutting := makeNotesAndCards(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, cutting, api, true, io.Discard)
	t.Cleanup(func() { _ = away() })

	if api.GetAgent() != nil {
		t.Error("a card can be asked about")
	}
}

// The file an agent a person runs themselves is configured from names one
// window's vault. This window writes neither it nor a token: a second writer
// would point that agent at whichever window started last.
func TestTheReviewerWritesDownNoAddressAndNoToken(t *testing.T) {
	cfg, db, opening, api, vaults := makeWindow(t)
	cfg.Agent = agent.Defaults()

	notes, cutting := makeNotesAndCards(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, cutting, api, false, io.Discard)
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
	cfg.Agent = agent.Defaults()

	notes, cutting := makeNotesAndCards(cfg, db, opening)
	away := serveAgents(t.Context(), cfg, db, notes, cutting, api, false, io.Discard)
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
