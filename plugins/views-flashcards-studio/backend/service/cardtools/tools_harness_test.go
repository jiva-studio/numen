package cardtools_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service"
)

// session is the tools the plugin offers, called the way the host calls them.
type session struct {
	registry *tools.Registry
	// toolContext is what the host tells each tool about the window the agent was started from.
	toolContext map[string]string
	// vault is the vault the host says each call works.
	vault domain.VaultID
}

// newSession is the tools over a vault of the notes given.
func newSession(t *testing.T, notes map[string]string) (*session, domain.Vault) {
	t.Helper()
	env := vaulttest.New(t, notes)
	registry, err := service.New(service.Config{Vault: env.Host, Store: env.Store}).Tools()
	if err != nil {
		t.Fatal(err)
	}
	return &session{registry: registry, vault: env.Vault.ID}, env.Vault
}

// invoke calls one tool and returns what it answered.
func (s *session) invoke(t *testing.T, name string, args any) *tools.CallResponse {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.registry.InvokeTool(t.Context(), tools.Call{
		Name: name, Arguments: raw, Context: s.toolContext, VaultID: string(s.vault),
	})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.GetError() != tools.ErrorCodeUnspecified {
		t.Fatalf("%s: rejected as %v", name, res.GetError())
	}
	return res
}

// list is the tools the plugin offers.
func (s *session) list(t *testing.T) []*tools.ToolInfo {
	t.Helper()
	listed, err := s.registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return listed
}

// call is a tool called and its answer read as T.
func call[T any](t *testing.T, s *session, name string, args any) T {
	t.Helper()
	res := s.invoke(t, name, args).GetResult()
	if res.GetIsRefusal() {
		t.Fatalf("%s: %s", name, res.GetText())
	}
	var out T
	if err := json.Unmarshal(res.GetStructured(), &out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return out
}

func getToolError(t *testing.T, s *session, name string, args any) string {
	t.Helper()
	res := s.invoke(t, name, args).GetResult()
	if !res.GetIsRefusal() {
		t.Fatalf("%s: expected an error, got %s", name, res.GetStructured())
	}
	return res.GetText()
}

// deckFingerprint is what a deck is at this moment, as card_read gives it.
func deckFingerprint(t *testing.T, s *session, path string) string {
	t.Helper()
	return call[struct {
		Fingerprint string `json:"fingerprint"`
	}](t, s, "card_read", map[string]any{"path": path}).Fingerprint
}

// A call works the vault the host names, though the host shows none of its own, and a call that
// names none is refused.
func TestACallWorksTheVaultTheHostNames(t *testing.T) {
	installation := vaulttest.NewInstallation()
	installation.Add(t, map[string]string{"Animals.md": "# Empty\n"})
	garden := installation.Add(t, vault())
	registry, err := service.New(service.Config{Vault: installation.Host, Store: installation.Host.Store()}).Tools()
	if err != nil {
		t.Fatal(err)
	}
	s := &session{registry: registry, vault: garden.ID}

	if read := readCards(t, s, map[string]any{"path": "Animals.md"}); len(read.Cards) != 2 || read.Cards[0].CardID != llama {
		t.Errorf("the call read %+v", read.Cards)
	}

	s.vault = ""
	if got := getToolError(t, s, "card_read", map[string]any{"path": "Animals.md"}); got != "the call names no vault" {
		t.Errorf("a call naming no vault answered %q", got)
	}
}
