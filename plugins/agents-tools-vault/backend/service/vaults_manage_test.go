package service_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

const (
	gardenID     = "garden-vault"
	gardenName   = "Kitchen Garden"
	gardenFolder = "/vaults/garden-vault"
)

type vaultOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type vaultRemoveOutput struct {
	Path string `json:"path"`
}

// requireExactKeys fails the test unless the answer has exactly the keys.
func requireExactKeys(t *testing.T, structured []byte, want ...string) {
	t.Helper()
	var answer map[string]any
	decodeInto(t, structured, &answer)
	got := make([]string, 0, len(answer))
	for key := range answer {
		got = append(got, key)
	}
	slices.Sort(got)
	want = slices.Clone(want)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("answer %s has the keys %v, want %v", structured, got, want)
	}
}

// newTwoVaults is the geology vault, which the calls work on, beside a garden vault whose name
// differs from its identity and its folder.
func newTwoVaults(t *testing.T) (*hosttest.Vault, *hosttest.Vault, *tools.Registry) {
	t.Helper()
	geology, registry := newGeologyVault(t)
	garden := hosttest.New(gardenID)
	garden.File("Beds.md", "# Beds\nRaised beds of oak.")
	geology.Beside(garden)
	if _, err := garden.RenameVault(t.Context(), gardenID, gardenName); err != nil {
		t.Fatalf("name the garden vault: %v", err)
	}
	return geology, garden, registry
}

// getGardenAddresses is the three ways a call can name the garden vault: its name, its folder, its identity.
func getGardenAddresses() map[string]string {
	return map[string]string{"name": gardenName, "folder": gardenFolder, "id": gardenID}
}

// getVaultsByID is the vaults vault_list gives, by identity.
func getVaultsByID(t *testing.T, registry *tools.Registry) map[string]vaultEntry {
	t.Helper()
	var out vaultListOutput
	decodeInto(t, callTool(t, registry, geologyVault, "vault_list", `{}`), &out)
	byID := map[string]vaultEntry{}
	for _, one := range out.Vaults {
		byID[one.ID] = one
	}
	return byID
}

func TestVaultAddAddsTheFolderAsAVaultWithTheName(t *testing.T) {
	_, _, registry := newTwoVaults(t)

	var out vaultOutput
	answer := callTool(t, registry, geologyVault, "vault_add", `{"path":"/vaults/meadow","name":"Meadow"}`)
	requireExactKeys(t, answer, "id", "name", "path")
	decodeInto(t, answer, &out)
	if out.ID != "Meadow" || out.Name != "Meadow" || out.Path != "/vaults/meadow" {
		t.Errorf("result = %+v, want id Meadow, name Meadow, path /vaults/meadow", out)
	}
	if got, ok := getVaultsByID(t, registry)["Meadow"]; !ok || got.Path != "/vaults/meadow" {
		t.Errorf("vault_list gives %+v for Meadow, want it listed at /vaults/meadow", got)
	}
}

func TestVaultAddWithoutANameUsesTheFolderName(t *testing.T) {
	_, _, registry := newTwoVaults(t)

	var out vaultOutput
	answer := callTool(t, registry, geologyVault, "vault_add", `{"path":"/vaults/meadow"}`)
	requireExactKeys(t, answer, "id", "name", "path")
	decodeInto(t, answer, &out)
	if out.Name != "meadow" || out.Path != "/vaults/meadow" {
		t.Errorf("result = %+v, want name meadow, path /vaults/meadow", out)
	}
}

func TestVaultAddWithoutAPathIsRefused(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	requireRefusal(t, registry, geologyVault, "vault_add", `{}`)
	requireRefusal(t, registry, geologyVault, "vault_add", `{"name":"Meadow"}`)
	if got := getVaultsByID(t, registry); len(got) != 2 {
		t.Errorf("vault_list gives %d vaults, want the 2 there were: %v", len(got), got)
	}
}

func TestVaultAddOfAFolderInsideAVaultIsRefused(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	requireRefusal(t, registry, geologyVault, "vault_add", `{"path":"/vaults/garden-vault/Seeds"}`)
}

func TestVaultAddOfAnOverlappingFolderReportsTheHostReason(t *testing.T) {
	_, _, registry := newTwoVaults(t)

	text := getRefusalText(t, registry, geologyVault, "vault_add", `{"path":"/vaults/garden-vault/Seeds"}`)
	if reason := "a vault cannot be inside another vault"; !strings.Contains(text, reason) {
		t.Errorf("the refusal %q does not carry the reason %q", text, reason)
	}
}

func TestVaultRenameNamesTheVaultByItsNameFolderOrID(t *testing.T) {
	for kind, address := range getGardenAddresses() {
		t.Run(kind, func(t *testing.T) {
			_, _, registry := newTwoVaults(t)
			want := "Herb Garden " + kind

			var out vaultOutput
			decodeInto(t, callTool(t, registry, geologyVault, "vault_rename",
				fmt.Sprintf(`{"vault":%q,"name":%q}`, address, want)), &out)
			if out.ID != gardenID || out.Name != want || out.Path != gardenFolder {
				t.Errorf("result = %+v, want id %s, name %s, path %s", out, gardenID, want, gardenFolder)
			}
			if got := getVaultsByID(t, registry)[gardenID]; got.Name != want {
				t.Errorf("vault_list gives the name %q, want %q", got.Name, want)
			}
		})
	}
}

func TestVaultRenameOntoATakenNameOrOfAnUnknownVaultIsRefused(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	requireRefusal(t, registry, geologyVault, "vault_rename", fmt.Sprintf(`{"vault":%q,"name":%q}`, gardenID, geologyVault))
	requireRefusal(t, registry, geologyVault, "vault_rename", `{"vault":"Desert","name":"Dunes"}`)
	if got := getVaultsByID(t, registry)[gardenID]; got.Name != gardenName {
		t.Errorf("vault_list gives the name %q, want %q kept", got.Name, gardenName)
	}
}

func TestVaultRemoveByNameFolderOrIDDropsTheVault(t *testing.T) {
	for kind, address := range getGardenAddresses() {
		t.Run(kind, func(t *testing.T) {
			_, _, registry := newTwoVaults(t)

			var out vaultRemoveOutput
			answer := callTool(t, registry, geologyVault, "vault_remove", fmt.Sprintf(`{"vault":%q}`, address))
			decodeInto(t, answer, &out)
			requireExactKeys(t, answer, "path")
			if out.Path != gardenFolder {
				t.Errorf("result = %+v, want path %s", out, gardenFolder)
			}
			vaults := getVaultsByID(t, registry)
			if _, ok := vaults[gardenID]; ok {
				t.Errorf("vault_list still gives %s: %v", gardenID, vaults)
			}
			if _, ok := vaults[geologyVault]; !ok {
				t.Errorf("vault_list no longer gives %s: %v", geologyVault, vaults)
			}
		})
	}
}

func TestVaultRemoveOfTheVaultOfTheCallIsRefused(t *testing.T) {
	geology, _, registry := newTwoVaults(t)
	folder := geology.Vault().Path
	for _, address := range []string{geologyVault, folder} {
		requireRefusal(t, registry, geologyVault, "vault_remove", fmt.Sprintf(`{"vault":%q}`, address))
	}
	if _, ok := getVaultsByID(t, registry)[geologyVault]; !ok {
		t.Errorf("vault_list no longer gives %s", geologyVault)
	}
}

func TestVaultOpenOpensTheVaultByItsNameFolderOrID(t *testing.T) {
	for kind, address := range getGardenAddresses() {
		t.Run(kind, func(t *testing.T) {
			geology, _, registry := newTwoVaults(t)

			var out vaultOutput
			answer := callTool(t, registry, geologyVault, "vault_open", fmt.Sprintf(`{"vault":%q}`, address))
			decodeInto(t, answer, &out)
			requireExactKeys(t, answer, "id", "name", "path")
			if out.ID != gardenID || out.Name != gardenName || out.Path != gardenFolder {
				t.Errorf("result = %+v, want id %s, name %s, path %s", out, gardenID, gardenName, gardenFolder)
			}
			opened, ok := geology.GetOpenedVault()
			if !ok || string(opened.ID) != gardenID {
				t.Errorf("opened vault = %+v, %v, want %s", opened, ok, gardenID)
			}
		})
	}
}

func TestVaultOpenWithUnsavedChangesInTheWindowReturnsTheHostError(t *testing.T) {
	geology, _, registry := newTwoVaults(t)
	geology.SetUnsavedChanges(true)

	text := getRefusalText(t, registry, geologyVault, "vault_open", fmt.Sprintf(`{"vault":%q}`, gardenID))

	if !strings.Contains(text, "unsaved changes") {
		t.Errorf("the vault_open error %q does not say the window has unsaved changes", text)
	}
	if opened, ok := geology.GetOpenedVault(); ok {
		t.Errorf("the vault %s was opened while the window had unsaved changes", opened.ID)
	}
}

func TestVaultOpenOfTheVaultOfTheCallIsRefused(t *testing.T) {
	geology, _, registry := newTwoVaults(t)
	requireRefusal(t, registry, geologyVault, "vault_open", fmt.Sprintf(`{"vault":%q}`, geologyVault))
	if opened, ok := geology.GetOpenedVault(); ok {
		t.Errorf("the vault %s was opened", opened.ID)
	}
}

func TestVaultOpenOfAnUnknownVaultIsRefused(t *testing.T) {
	geology, _, registry := newTwoVaults(t)
	requireRefusal(t, registry, geologyVault, "vault_open", `{"vault":"Desert"}`)
	if opened, ok := geology.GetOpenedVault(); ok {
		t.Errorf("the vault %s was opened", opened.ID)
	}
}

func TestVaultListToolsWithoutAVaultListAreNotListed(t *testing.T) {
	registry, err := service.New(hosttest.New(geologyVault)).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	infos, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]bool{}
	for _, info := range infos.Tools {
		names[info.GetName()] = true
	}
	for _, name := range []string{"vault_add", "vault_rename", "vault_remove", "vault_open"} {
		if names[name] {
			t.Errorf("the tool %s is listed without a vault list", name)
		}
	}
	if !names["vault_list"] {
		t.Error("the tool vault_list is not listed without a vault list")
	}
}

func TestVaultAddOfAFolderInsideAVaultAnswersThePathAndTheVault(t *testing.T) {
	_, _, registry := newTwoVaults(t)

	text := getRefusalText(t, registry, geologyVault, "vault_add", `{"path":"/vaults/garden-vault/Seeds"}`)

	if want := "vault_add: /vaults/garden-vault/Seeds is inside the vault Kitchen Garden; a vault cannot be inside another vault"; text != want {
		t.Errorf("the refusal is %q, want %q", text, want)
	}
}

func TestVaultAddOfAFolderContainingAVaultAnswersThePathAndTheVault(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	callTool(t, registry, geologyVault, "vault_add", `{"path":"/fields/meadow","name":"Meadow"}`)

	text := getRefusalText(t, registry, geologyVault, "vault_add", `{"path":"/fields"}`)

	if want := "vault_add: /fields contains the vault Meadow; a vault cannot contain another vault"; text != want {
		t.Errorf("the refusal is %q, want %q", text, want)
	}
}

func TestVaultOpenAndRemoveOfTheCurrentVaultAnswerItIsTheCurrentVault(t *testing.T) {
	geology, _, registry := newTwoVaults(t)
	for _, address := range []string{geologyVault, geology.Vault().Path} {
		for _, name := range []string{"vault_open", "vault_remove"} {
			text := getRefusalText(t, registry, geologyVault, name, fmt.Sprintf(`{"vault":%q}`, address))
			if want := name + ": " + address + " is the current vault"; text != want {
				t.Errorf("the refusal is %q, want %q", text, want)
			}
		}
	}
}

// getFailingCalls is a call of each vault and note tool whose arguments make it fail: an unknown
// vault, a missing note, an empty path and the like.
func getFailingCalls() []struct{ name, arguments string } {
	return []struct{ name, arguments string }{
		{"vault_add", `{"path":""}`},
		{"vault_add", `{"path":"/vaults/garden-vault/Seeds"}`},
		{"vault_rename", `{"vault":"Desert","name":"Dunes"}`},
		{"vault_rename", `{"vault":"garden-vault","name":"geology-vault"}`},
		{"vault_rename", `{"vault":"garden-vault","name":""}`},
		{"vault_remove", `{"vault":"Desert"}`},
		{"vault_remove", `{"vault":""}`},
		{"vault_remove", `{"vault":"geology-vault"}`},
		{"vault_open", `{"vault":"Desert"}`},
		{"vault_open", `{"vault":"geology-vault"}`},
		{"vault_problem_list", `{"checks":["spelling"]}`},
		{"note_read", `{"path":"Rocks/Obsidian.md"}`},
		{"note_read", `{"path":""}`},
		{"note_write", `{"path":"","content":"Pumice floats."}`},
		{"note_heading_list", `{"path":"Rocks/Obsidian.md"}`},
		{"note_list", `{"limit":-1}`},
		{"note_resolve", `{"name":""}`},
		{"note_rename", `{"path":"Rocks/Obsidian.md","title":"Glass"}`},
		{"note_rename", `{"path":"","title":"Glass"}`},
		{"note_rename", `{"path":"Rocks/Basalt.md","title":"Granite"}`},
		{"note_move", `{"paths":[],"folder":"Archive"}`},
		{"note_remove", `{"paths":[]}`},
	}
}

func TestEveryVaultAndNoteToolErrorStartsWithTheToolNameOnceAndNamesNoInnerCall(t *testing.T) {
	inner := []string{
		"host:", "add vault", "rename vault", "remove vault", "open vault", "list vaults",
		"move notes", "remove notes", "was not changed", "this call works on",
	}
	for _, c := range getFailingCalls() {
		t.Run(c.name+" "+c.arguments, func(t *testing.T) {
			_, _, registry := newTwoVaults(t)

			text := getRefusalText(t, registry, geologyVault, c.name, c.arguments)

			if !strings.HasPrefix(text, c.name+": ") {
				t.Errorf("the error %q does not start with %q", text, c.name+": ")
			}
			if count := strings.Count(text, c.name); count != 1 {
				t.Errorf("the error %q names %s %d times, want once", text, c.name, count)
			}
			for _, phrase := range inner {
				if strings.Contains(text, phrase) {
					t.Errorf("the error %q contains %q", text, phrase)
				}
			}
		})
	}
}
