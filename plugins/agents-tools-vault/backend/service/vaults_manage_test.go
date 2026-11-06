package service_test

import (
	"fmt"
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

type vaultAddOutput struct {
	Added  bool   `json:"added"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
}

type vaultRenameOutput struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
}

type vaultRemoveOutput struct {
	Removed bool   `json:"removed"`
	Folder  string `json:"folder"`
}

type vaultOpenOutput struct {
	Opening bool   `json:"opening"`
	Status  string `json:"status"`
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
func getVaultsByID(t *testing.T, registry *tools.Registry) map[string]listedVault {
	t.Helper()
	var out vaultListOutput
	decodeInto(t, callTool(t, registry, geologyVault, "vault_list", `{}`), &out)
	byID := map[string]listedVault{}
	for _, one := range out.Vaults {
		byID[one.ID] = one
	}
	return byID
}

func TestVaultAddAddsTheFolderAsAVaultWithTheName(t *testing.T) {
	_, _, registry := newTwoVaults(t)

	var out vaultAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "vault_add", `{"path":"/vaults/meadow","name":"Meadow"}`), &out)
	if !out.Added || out.ID != "Meadow" || out.Name != "Meadow" || out.Folder != "/vaults/meadow" {
		t.Errorf("result = %+v, want added, id Meadow, name Meadow, folder /vaults/meadow", out)
	}
	if got, ok := getVaultsByID(t, registry)["Meadow"]; !ok || got.Folder != "/vaults/meadow" {
		t.Errorf("vault_list gives %+v for Meadow, want it listed at /vaults/meadow", got)
	}
}

func TestVaultAddWithoutANameUsesTheFolderName(t *testing.T) {
	_, _, registry := newTwoVaults(t)

	var out vaultAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "vault_add", `{"path":"/vaults/meadow"}`), &out)
	if !out.Added || out.Name != "meadow" || out.Folder != "/vaults/meadow" {
		t.Errorf("result = %+v, want added, name meadow, folder /vaults/meadow", out)
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
	if reason := "the folder lies inside a vault on the list or holds one"; !strings.Contains(text, reason) {
		t.Errorf("the refusal %q does not carry the reason %q", text, reason)
	}
}

func TestVaultRenameNamesTheVaultByItsNameFolderOrID(t *testing.T) {
	for kind, address := range getGardenAddresses() {
		t.Run(kind, func(t *testing.T) {
			_, _, registry := newTwoVaults(t)
			want := "Herb Garden " + kind

			var out vaultRenameOutput
			decodeInto(t, callTool(t, registry, geologyVault, "vault_rename",
				fmt.Sprintf(`{"vault":%q,"name":%q}`, address, want)), &out)
			if out.ID != gardenID || out.Name != want || out.Folder != gardenFolder {
				t.Errorf("result = %+v, want id %s, name %s, folder %s", out, gardenID, want, gardenFolder)
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
			decodeInto(t, callTool(t, registry, geologyVault, "vault_remove", fmt.Sprintf(`{"vault":%q}`, address)), &out)
			if !out.Removed || out.Folder != gardenFolder {
				t.Errorf("result = %+v, want removed with folder %s", out, gardenFolder)
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

			var out vaultOpenOutput
			decodeInto(t, callTool(t, registry, geologyVault, "vault_open", fmt.Sprintf(`{"vault":%q}`, address)), &out)
			if !out.Opening || out.Status == "" {
				t.Errorf("result = %+v, want opening with a status", out)
			}
			opened, ok := geology.GetOpenedVault()
			if !ok || string(opened.ID) != gardenID {
				t.Errorf("opened vault = %+v, %v, want %s", opened, ok, gardenID)
			}
		})
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
