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

func (r *rows) Save(context.Context, domain.VaultID) error { return nil }

func (r *rows) Register(context.Context, domain.VaultID) error { return nil }

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

// folders is the person picking a folder, as a test answers for them. The real
// picker is this machine's own and needs a window.
type folders struct {
	mu    sync.Mutex
	pick  string
	chose bool
	title string
}

func (f *folders) Choose(_ context.Context, title, _ string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.title = title
	return f.pick, f.chose, nil
}

// answers is what the picker hands back to the call that put it up.
func (f *folders) answers(path string, chose bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pick, f.chose = path, chose
}

// asked is what the person was told the picker was for.
func (f *folders) asked() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.title
}

// installation is the tools as an agent meets them, with a list of vaults
// behind them: two on it, and the window showing the first.
type installation struct {
	core     mcp.Core
	registry port.VaultRegistry
	rows     *rows
	picker   *folders
	first    domain.Vault
	second   domain.Vault

	// swapped carries every vault the window was asked to show. The endpoint a
	// call arrives on is stopped by the swap, so the tool answers first and the
	// window moves behind it.
	swapped chan domain.Vault
}

func onTheList(t *testing.T) *installation {
	t.Helper()

	registry := appstate.At(filepath.Join(t.TempDir(), "state", "vaults.json"))
	adding := usecase.Add{Identity: filesystem.VaultIdentity{}, Registry: registry, Now: time.Now}
	held := &rows{}

	f := &installation{
		registry: registry,
		rows:     held,
		picker:   &folders{},
		first:    joined(t, adding, "one"),
		second:   joined(t, adding, "two"),
		swapped:  make(chan domain.Vault, 1),
	}
	f.core = mcp.Core{
		Showing: mcp.One(f.first, f.first.Path),
		Vaults: mcp.Vaults{
			Registry: registry,
			Picker:   f.picker,
			Add:      &adding,
			Rename:   &usecase.Rename{Registry: registry, Index: held},
			Forget:   &usecase.Forget{Registry: registry, Index: held},
			Opens: func(_ context.Context, v domain.Vault) error {
				f.swapped <- v
				return nil
			},
		},
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

// joining is what an agent is answered with by vault_add.
type joining struct {
	Added  bool   `json:"added"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
	Why    string `json:"why"`
}

// describing is what a tool tells an agent about itself, which is as much the
// interface as its name is.
func describing(t *testing.T, s *sdk.ClientSession, name string) string {
	t.Helper()

	listed, err := s.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name == name {
			return tool.Description
		}
	}
	t.Fatalf("%s is missing", name)
	return ""
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
		case string(f.first.ID):
			if !one.Showing || one.Missing || one.Folder != f.first.Path {
				t.Errorf("the vault in front is answered as %+v", one)
			}
		case string(f.second.ID):
			if one.Showing || !one.Missing || one.Name != f.second.Name {
				t.Errorf("the vault whose folder is gone is answered as %+v", one)
			}
		default:
			t.Errorf("the list holds %+v", one)
		}
	}
}

// A vault added with no path is the person's own choice, made in front of them.
func TestAddingWithNoPathAsksThePersonAndAddsWhatTheyChose(t *testing.T) {
	f := onTheList(t)
	chosen := folderNamed(t, "three")
	f.picker.answers(chosen, true)
	session := connectedTo(t, f.core)

	out := call[joining](t, session, "vault_add", struct{}{})
	if !out.Added || out.Folder != chosen || out.Name != "three" {
		t.Fatalf("the vault added is %+v", out)
	}
	if f.picker.asked() == "" {
		t.Error("the person was shown a picker with nothing on it")
	}
	if _, found, err := f.registry.Find(out.ID); err != nil || !found {
		t.Errorf("the vault added is not on the list: %v", err)
	}
}

// The folder gains a file, and the tool says which one before it is called.
func TestAddingSaysTheIdentityIsWrittenIntoTheFolder(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	said := describing(t, session, "vault_add")
	for _, rule := range []string{"identity", filesystem.DefaultServiceDir} {
		if !strings.Contains(said, rule) {
			t.Errorf("vault_add says nothing about %q:\n%s", rule, said)
		}
	}

	chosen := folderNamed(t, "four")
	call[joining](t, session, "vault_add", map[string]any{"path": chosen})
	if _, err := os.Stat(filepath.Join(chosen, filesystem.DefaultServiceDir)); err != nil {
		t.Errorf("the folder holds no identity: %v", err)
	}
}

// A person who closed the picker chose nothing, and that is an answer.
func TestAPickerThePersonClosedAddsNothingAndSaysSo(t *testing.T) {
	f := onTheList(t)
	f.picker.answers("", false)
	session := connectedTo(t, f.core)

	out := call[joining](t, session, "vault_add", struct{}{})
	if out.Added || out.ID != "" {
		t.Fatalf("something was added: %+v", out)
	}
	if !strings.Contains(out.Why, "picker") {
		t.Errorf("the answer says %q", out.Why)
	}
	held, err := f.registry.All()
	if err != nil || len(held) != 2 {
		t.Errorf("the list holds %v: %v", held, err)
	}
}

// A vault is a folder somebody keeps notes in, and the two folders holding
// everything a person has are neither of them that.
func TestAFilesystemRootAndAHomeDirectoryAreRefused(t *testing.T) {
	home := folderNamed(t, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	f := onTheList(t)
	session := connectedTo(t, f.core)

	// The root of the volume the home directory is on, which is the whole of a
	// path on a machine that names no volume.
	top := filepath.VolumeName(home) + string(filepath.Separator)

	for root, said := range map[string]string{
		top:  "root of this filesystem",
		home: "home directory",
	} {
		why := failing(t, session, "vault_add", map[string]any{"path": root})
		if !strings.Contains(why, said) {
			t.Errorf("%s was refused with %q", root, why)
		}
		if _, err := os.Stat(filepath.Join(root, filesystem.DefaultServiceDir)); err == nil {
			t.Errorf("%s was made into a vault before it was refused", root)
		}
	}
	held, err := f.registry.All()
	if err != nil || len(held) != 2 {
		t.Errorf("the list holds %v: %v", held, err)
	}
}

// A vault does not lie inside another, and the list is what says so.
func TestAFolderInsideAVaultOnTheListIsRefused(t *testing.T) {
	f := onTheList(t)
	inner := filepath.Join(f.first.Path, "projects", "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	session := connectedTo(t, f.core)

	why := failing(t, session, "vault_add", map[string]any{"path": inner})
	if !strings.Contains(why, "does not lie inside") {
		t.Errorf("a folder inside a vault was refused with %q", why)
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
	if got := f.rows.forgotten(); len(got) != 1 || got[0] != string(f.second.ID) {
		t.Errorf("the index was told to forget %v", got)
	}
	if _, found, err := f.registry.Find(string(f.second.ID)); err != nil || found {
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

func TestOpeningAVaultMovesTheWindowToIt(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	out := call[struct {
		Opening bool   `json:"opening"`
		Doing   string `json:"doing"`
	}](t, session, "vault_open", map[string]any{"vault": f.second.Name})
	if !out.Opening || !strings.Contains(out.Doing, f.second.Name) {
		t.Fatalf("the answer is %+v", out)
	}

	select {
	case shown := <-f.swapped:
		if shown.ID != f.second.ID {
			t.Errorf("the window was asked to show %+v", shown)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the window was never asked to show another vault")
	}
}

// The session ends with the vault it was serving, and the tool's own
// description is where an agent is told so.
func TestOpeningSaysThatTheSessionEnds(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	if said := describing(t, session, "vault_open"); !strings.Contains(said, "session ends") {
		t.Errorf("vault_open says %q", said)
	}
}

// The vault already in front of the person is not worth a session for.
func TestOpeningTheVaultInFrontIsRefused(t *testing.T) {
	f := onTheList(t)
	session := connectedTo(t, f.core)

	why := failing(t, session, "vault_open", map[string]any{"vault": string(f.first.ID)})
	if !strings.Contains(why, "already showing") {
		t.Errorf("the vault in front was refused with %q", why)
	}
	select {
	case shown := <-f.swapped:
		t.Errorf("the window was asked to show %+v", shown)
	default:
	}
}

// A tool is served where what it works through is there, and a build without it
// serves nothing an agent could call and be refused by.
func TestAToolIsNotServedWithoutWhatItWorksThrough(t *testing.T) {
	for _, one := range []struct {
		tool    string
		without func(*mcp.Core)
	}{
		{"vault_list", func(c *mcp.Core) { c.Vaults.Registry = nil }},
		{"vault_add", func(c *mcp.Core) { c.Vaults.Add = nil }},
		{"vault_rename", func(c *mcp.Core) { c.Vaults.Rename = nil }},
		{"vault_forget", func(c *mcp.Core) { c.Vaults.Forget = nil }},
		{"vault_open", func(c *mcp.Core) { c.Vaults.Opens = nil }},
		{"vault_rename", func(c *mcp.Core) { c.Vaults.Registry = nil }},
		{"vault_forget", func(c *mcp.Core) { c.Vaults.Registry = nil }},
		{"vault_open", func(c *mcp.Core) { c.Vaults.Registry = nil }},
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

// A vault is added by a person choosing a folder, and a build with nowhere to
// put a picker takes the folder named.
func TestAddingWithNoPathAndNoPickerSaysSo(t *testing.T) {
	f := onTheList(t)
	core := f.core
	core.Vaults.Picker = nil
	session := connectedTo(t, core)

	if !offers(t, session, "vault_add") {
		t.Fatal("vault_add is missing, and a folder can be named")
	}
	why := failing(t, session, "vault_add", struct{}{})
	if !strings.Contains(why, "pick a folder") {
		t.Errorf("a build with no picker refused with %q", why)
	}
}
