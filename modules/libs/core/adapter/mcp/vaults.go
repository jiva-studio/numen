package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Known is one vault this installation holds, as an agent is told about it.
type Known struct {
	ID     string `json:"id" jsonschema:"the identity this vault keeps wherever its folder moves to"`
	Name   string `json:"name" jsonschema:"what the person calls it"`
	Folder string `json:"folder" jsonschema:"where the vault is on this machine"`
	// Missing is a folder that is not there to be read. The vault stays on the
	// list until somebody forgets it.
	Missing bool `json:"missing,omitempty" jsonschema:"there is nothing at that folder now"`
	Showing bool `json:"showing,omitempty" jsonschema:"the vault the person is looking at, and the one every other tool works"`
}

// addVaultsTools gives an agent the list of vaults this installation holds and
// what a person does to it: call one something else, and take one off the list.
//
// Which folders are vaults, and which of them the window shows, a person
// settles in front of the application through a folder picker.
//
// Each tool is added where what it works through is there. A build that holds
// no list, or that cannot change one, serves what is left.
func addVaultsTools(server *sdk.Server, core Core) {
	addVaultsReadingTools(server, core)
	addVaultsWritingTools(server, core)
}

func addVaultsReadingTools(server *sdk.Server, core Core) {
	addVaultList(server, core)
}

func addVaultsWritingTools(server *sdk.Server, core Core) {
	addVaultRename(server, core)
	addVaultForget(server, core)
}

func addVaultList(server *sdk.Server, core Core) {
	if core.Vaults == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_list",
		Title: "List the vaults this installation holds",
		Description: "Every vault this installation holds: what each is called, where its " +
			"folder is on this machine, and which one the person is looking at. Every " +
			"other tool works the vault marked `showing`, and no other. A vault whose " +
			"folder is gone is marked and stays on the list until somebody forgets it.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, struct {
		Vaults []Known `json:"vaults"`
	}, error) {
		type out = struct {
			Vaults []Known `json:"vaults"`
		}
		held, err := usecase.List{Registry: core.Vaults}.Execute()
		if err != nil {
			return nil, out{}, err
		}
		showing := core.shown().Vault.ID
		vaults := make([]Known, 0, len(held))
		for _, v := range held {
			vaults = append(vaults, knownOf(v, showing))
		}
		return nil, out{Vaults: vaults}, nil
	})
}

func addVaultRename(server *sdk.Server, core Core) {
	if core.Vaults == nil || core.Renaming == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_rename",
		Title: "Call a vault something else",
		Description: "What the person calls a vault. The folder keeps the name the " +
			"filesystem gives it and nothing on disk moves. A name another vault on the " +
			"list has is refused, so the names in `vault_list` name one vault each.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to rename: its name, its folder, or the identity vault_list gives it"`
		Name  string `json:"name" jsonschema:"what it is called from now on"`
	}) (*sdk.CallToolResult, struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Folder string `json:"folder"`
	}, error) {
		type out = struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Folder string `json:"folder"`
		}
		v, err := core.found(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		renamed, err := core.Renaming.Execute(ctx, v, in.Name)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{ID: renamed.ID, Name: renamed.Name, Folder: renamed.Path}, nil
	})
}

func addVaultForget(server *sdk.Server, core Core) {
	if core.Vaults == nil || core.Forgetting == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_forget",
		Title: "Take a vault off the list",
		Description: "Take a vault off the list and out of the index. Its folder stays " +
			"where it is with everything in it, and a person putting it back on the list " +
			"gets the same vault under the same identity. The vault the person is looking at is " +
			"refused, and so is the last vault an installation has. Nothing here deletes " +
			"a folder: taking a person's notes off their disk is theirs to ask for, in " +
			"front of them.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to forget: its name, its folder, or the identity vault_list gives it"`
	}) (*sdk.CallToolResult, struct {
		Forgotten bool   `json:"forgotten"`
		Folder    string `json:"folder" jsonschema:"where the folder still is, with everything in it"`
	}, error) {
		type out = struct {
			Forgotten bool   `json:"forgotten"`
			Folder    string `json:"folder" jsonschema:"where the folder still is, with everything in it"`
		}
		v, err := core.found(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		// The use cases are not told which vault is in front of the person, and
		// this is.
		if v.ID == core.shown().Vault.ID {
			return nil, out{}, fmt.Errorf("%s is the vault the window is showing", v.Name)
		}
		if err := core.Forgetting.Execute(ctx, v); err != nil {
			return nil, out{}, err
		}
		return nil, out{Forgotten: true, Folder: v.Path}, nil
	})
}

// found is the vault a name, a folder or an identity reaches on the list.
func (c Core) found(nameOrPath string) (domain.Vault, error) {
	if nameOrPath == "" {
		return domain.Vault{}, errors.New("name the vault, as vault_list gives it")
	}
	return usecase.Find{Registry: c.Vaults}.Execute(nameOrPath)
}

// knownOf is one vault as an agent is told about it. A folder that is not there
// to be found is marked, and the vault stays on the list.
func knownOf(v domain.Vault, showing string) Known {
	_, err := os.Stat(v.Path)
	return Known{
		ID:      v.ID,
		Name:    v.Name,
		Folder:  v.Path,
		Missing: err != nil,
		Showing: v.ID != "" && v.ID == showing,
	}
}
