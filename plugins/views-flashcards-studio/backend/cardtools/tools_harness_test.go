package cardtools_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service"
)

// session is the tools the plugin offers, called the way the host calls them.
type session struct {
	registry    *tools.Registry
	toolContext map[string]string
	vault       domain.VaultID
}

func newSession(t *testing.T, notes map[string]string) (*session, domain.Vault) {
	t.Helper()
	env := vaulttest.New(t, notes)
	registry, err := service.New(service.Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock()}).Tools()
	if err != nil {
		t.Fatal(err)
	}
	return &session{registry: registry, vault: env.Vault.ID}, env.Vault
}

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

func (s *session) list(t *testing.T) []*tools.ToolInfo {
	t.Helper()
	listed, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return listed.Tools
}

func call[T any](t *testing.T, s *session, name string, args any) T {
	t.Helper()
	res := s.invoke(t, name, args).GetResult()
	if res.GetIsError() {
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
	if !res.GetIsError() {
		t.Fatalf("%s: expected an error, got %s", name, res.GetStructured())
	}
	return res.GetText()
}

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
	registry, err := service.New(service.Config{Vaults: vaulttest.Vaults(installation.Host), Store: vaulttest.Folders(installation.Host), Now: vaulttest.NewClock()}).Tools()
	if err != nil {
		t.Fatal(err)
	}
	s := &session{registry: registry, vault: garden.ID}

	if read := readCards(t, s, map[string]any{"path": "Animals.md"}); len(read.Cards) != 2 || read.Cards[0].CardID != llama {
		t.Errorf("the call read %+v", read.Cards)
	}

	s.vault = "unknown-vault"
	if got := getToolError(t, s, "card_read", map[string]any{"path": "Animals.md"}); got != "card_read: the agent vault is not registered in this installation" {
		t.Errorf("a call naming an unknown vault answered %q", got)
	}

	s.vault = ""
	for toolName, args := range map[string]map[string]any{
		"card_read":         {"path": "Animals.md"},
		"card_model_list":   {},
		"card_model_read":   {"path": "Animal.md"},
		"card_section_add":  {"path": "Animals.md", "name": "S", "fingerprint": "1-1"},
		"card_model_edit":   {"path": "Animal.md", "fields": []string{"Name"}, "templates": []any{}, "fingerprint": "1-1"},
		"card_deck_create":  {"title": "D"},
		"card_model_create": {"title": "M", "fields": []string{"Name"}, "templates": []any{}},
		"card_field_rename": {"path": "Animal.md", "from": "Name", "to": "Species"},
	} {
		if got := getToolError(t, s, toolName, args); got != toolName+": the call names no vault" {
			t.Errorf("%s naming no vault answered %q", toolName, got)
		}
	}

	// Transport errors from GetVault and VaultNotes/FileAccess propagate through tools.
	fv := &errHostVault{Vault: installation.Host, getVaultErr: errors.New("transport down")}
	errReg, err := service.New(service.Config{Vaults: fv.vaults(), Store: vaulttest.Folders(installation.Host), Now: vaulttest.NewClock()}).Tools()
	if err != nil {
		t.Fatal(err)
	}
	errSession := &session{registry: errReg, vault: garden.ID}
	if got := getToolError(t, errSession, "card_read", map[string]any{"path": "Animals.md"}); got != "card_read: transport down" {
		t.Errorf("got %q, want card_read: transport down", got)
	}
	fv.getVaultErr = nil
	fv.opErr = errors.New("io down")
	for toolName, args := range map[string]map[string]any{
		"card_model_list": {},
		"card_model_read": {"path": "Animal.md"},
		"card_model_edit": {"path": "Animal.md", "fields": []string{"Name"}, "fingerprint": "1-1"},
	} {
		if got := getToolError(t, errSession, toolName, args); got == "" {
			t.Errorf("%s expected error on io down", toolName)
		}
	}
}

type errHostVault struct {
	*hosttest.Vault
	getVaultErr error
	opErr       error
}

func (e *errHostVault) With(id domain.VaultID) *errHostVault {
	return &errHostVault{Vault: e.Vault.With(id), getVaultErr: e.getVaultErr, opErr: e.opErr}
}

func (e *errHostVault) GetVault(ctx context.Context) (domain.Vault, error) {
	if e.getVaultErr != nil {
		return domain.Vault{}, e.getVaultErr
	}
	return e.Vault.GetVault(ctx)
}

func (e *errHostVault) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	if e.opErr != nil {
		return nil, e.opErr
	}
	return e.Vault.GetPathsOfType(ctx, noteType)
}

func (e *errHostVault) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	if e.opErr != nil {
		return domain.FileStat{}, e.opErr
	}
	return e.Vault.StatFile(ctx, path)
}

func (e *errHostVault) vaults() func(domain.VaultID) hostports.Vault {
	return func(id domain.VaultID) hostports.Vault {
		on := e
		if id != "" {
			on = e.With(id)
		}
		return hostports.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Cache: on, Selector: on}
	}
}
