package mcp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// rows is the index as a vault is written to and taken out of it.
type rows struct {
	mu     sync.Mutex
	forgot []string
}

func (r *rows) Save(context.Context, domain.Vault) error { return nil }

func (r *rows) Register(context.Context, domain.Vault) error { return nil }

func (r *rows) Forget(_ context.Context, vaultID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.forgot = append(r.forgot, vaultID)
	return nil
}

func (r *rows) forgotten() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.forgot...)
}

// installation is the tools as an agent meets them, with a list of vaults
// behind them: two on it, and the window showing the first.
type installation struct {
	core     mcp.Core
	registry port.VaultRegistry
	rows     *rows
	first    domain.Vault
	second   domain.Vault
}

func onTheList(t *testing.T) *installation {
	t.Helper()

	registry := appstate.At(filepath.Join(t.TempDir(), "state", "vaults.json"))
	adding := usecase.Add{Identity: filesystem.Identity{}, Registry: registry, Now: time.Now}
	held := &rows{}

	f := &installation{
		registry: registry,
		rows:     held,
		first:    joined(t, adding, "one"),
		second:   joined(t, adding, "two"),
	}
	f.core = mcp.Core{
		Showing:    mcp.One(f.first, f.first.Path),
		Vaults:     registry,
		Renaming:   &usecase.Rename{Registry: registry, Index: held},
		Forgetting: &usecase.Forget{Registry: registry, Index: held},
	}
	return f
}

// joined makes a folder under a parent of its own and puts it on the list.
func joined(t *testing.T, add usecase.Add, name string) domain.Vault {
	t.Helper()

	v, err := add.Execute(folderNamed(t, name), name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// folderNamed makes a directory named name, under a parent of its own.
func folderNamed(t *testing.T, name string) string {
	t.Helper()

	at := filepath.Join(testsupport.TempDir(t), name)
	if err := os.MkdirAll(at, 0o755); err != nil {
		t.Fatal(err)
	}
	return at
}

// vaults is the list as an agent is answered with it.
type vaults struct {
	Vaults []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Folder  string `json:"folder"`
		Missing bool   `json:"missing"`
		Showing bool   `json:"showing"`
	} `json:"vaults"`
}

// offers reports whether a session is served a tool of that name.
func offers(t *testing.T, s *sdk.ClientSession, name string) bool {
	t.Helper()

	listed, err := s.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func TestTheListNamesTheVaultInFrontAndMarksAFolderThatIsGone(t *testing.T) {
	f := onTheList(t)
	if err := os.RemoveAll(f.second.Path); err != nil {
		t.Fatal(err)
	}
	session := connectedTo(t, f.core)

	list := call[vaults](t, session, "vault_list", struct{}{})
	if len(list.Vaults) != 2 {
		t.Fatalf("the list holds %v", list.Vaults)
	}
	for _, one := range list.Vaults {
		switch one.ID {
		case f.first.ID:
			if !one.Showing || one.Missing || one.Folder != f.first.Path {
				t.Errorf("the vault in front is answered as %+v", one)
			}
		case f.second.ID:
			if one.Showing || !one.Missing || one.Name != f.second.Name {
				t.Errorf("the vault whose folder is gone is answered as %+v", one)
			}
		default:
			t.Errorf("the list holds %+v", one)
		}
	}
}

// Which folders are vaults, and which of them the window shows, a person
// settles through the picker in front of them. An agent is served the list and
// the two names on it a person can ask it to change.
func TestTheVaultToolsAreTheListAndTheNamesOnIt(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	for _, name := range []string{"vault_add", "vault_open"} {
		if offers(t, session, name) {
			t.Errorf("%s is served to an agent", name)
		}
	}
	for _, name := range []string{"vault_list", "vault_rename", "vault_forget"} {
		if !offers(t, session, name) {
			t.Errorf("%s is missing from an installation that holds a list", name)
		}
	}
}

func TestRenamingToANameAnotherVaultHasSaysSo(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	why := failing(t, session, "vault_rename", map[string]any{
		"vault": f.second.ID, "name": f.first.Name,
	})
	if !strings.Contains(why, "already called") {
		t.Errorf("a taken name was refused with %q", why)
	}

	out := call[struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Folder string `json:"folder"`
	}](t, session, "vault_rename", map[string]any{"vault": f.second.ID, "name": "field notes"})
	if out.Name != "field notes" || out.Folder != f.second.Path {
		t.Errorf("the vault renamed is %+v", out)
	}
}

// The vault in front of the person stays: the use cases are not told which one
// that is, and the tools are.
func TestTheVaultTheWindowIsShowingIsNotForgotten(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	why := failing(t, session, "vault_forget", map[string]any{"vault": f.first.ID})
	if !strings.Contains(why, "showing") {
		t.Errorf("the vault in front was refused with %q", why)
	}
	if len(f.rows.forgotten()) != 0 {
		t.Errorf("the index was told to forget %v", f.rows.forgotten())
	}
}

// A vault forgotten leaves the list and the index, and its folder stays where
// it is.
func TestForgettingLeavesTheFolderWhereItIs(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	out := call[struct {
		Forgotten bool   `json:"forgotten"`
		Folder    string `json:"folder"`
	}](t, session, "vault_forget", map[string]any{"vault": f.second.ID})
	if !out.Forgotten || out.Folder != f.second.Path {
		t.Fatalf("the vault forgotten is %+v", out)
	}
	if _, err := os.Stat(f.second.Path); err != nil {
		t.Errorf("the folder went with the vault: %v", err)
	}
	if got := f.rows.forgotten(); len(got) != 1 || got[0] != f.second.ID {
		t.Errorf("the index was told to forget %v", got)
	}
	if _, found, err := f.registry.Find(f.second.ID); err != nil || found {
		t.Errorf("the vault is still on the list: %v", err)
	}
}

// An installation keeps a vault.
func TestTheLastVaultThisInstallationHasStays(t *testing.T) {
	f := onTheList(t)
	// The window is showing neither, so what is left is refused for being the
	// last one and not for being in front of anybody.
	f.core.Showing = mcp.One(domain.Vault{ID: "elsewhere"}, "")
	session := connectedTo(t, f.core)

	call[struct {
		Forgotten bool `json:"forgotten"`
	}](t, session, "vault_forget", map[string]any{"vault": f.second.ID})

	why := failing(t, session, "vault_forget", map[string]any{"vault": f.first.ID})
	if !strings.Contains(why, "only vault") {
		t.Errorf("the last vault was refused with %q", why)
	}
	held, err := f.registry.All()
	if err != nil || len(held) != 1 {
		t.Errorf("the list holds %v: %v", held, err)
	}
}

func TestForgettingAVaultTheListDoesNotHoldSaysSo(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	why := failing(t, session, "vault_forget", map[string]any{"vault": "nowhere"})
	if !strings.Contains(why, "no such vault") {
		t.Errorf("a vault nothing answers to was refused with %q", why)
	}
}

// A tool is served where what it works through is there, and a build without it
// serves nothing an agent could call and be refused by.
func TestAToolIsNotServedWithoutWhatItWorksThrough(t *testing.T) {
	for _, one := range []struct {
		tool    string
		without func(*mcp.Core)
	}{
		{"vault_list", func(c *mcp.Core) { c.Vaults = nil }},
		{"vault_rename", func(c *mcp.Core) { c.Renaming = nil }},
		{"vault_forget", func(c *mcp.Core) { c.Forgetting = nil }},
		{"vault_rename", func(c *mcp.Core) { c.Vaults = nil }},
		{"vault_forget", func(c *mcp.Core) { c.Vaults = nil }},
	} {
		t.Run(one.tool, func(t *testing.T) {
			f := onTheList(t)
			if !offers(t, connectedTo(t, f.core), one.tool) {
				t.Fatalf("%s is missing from an installation that holds a list", one.tool)
			}

			core := f.core
			one.without(&core)
			if offers(t, connectedTo(t, core), one.tool) {
				t.Errorf("%s is served with nothing behind it", one.tool)
			}
		})
	}
}
