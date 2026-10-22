package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Vault is one vault this installation holds, as an agent is told about it.
type Vault struct {
	ID     string `json:"id" jsonschema:"the identity this vault keeps wherever its folder moves to"`
	Name   string `json:"name" jsonschema:"what the person calls it"`
	Folder string `json:"folder" jsonschema:"where the vault is on this machine"`
	// IsMissing is a folder that is not there to be read. The vault stays on the
	// list until somebody forgets it.
	IsMissing bool `json:"missing,omitempty" jsonschema:"there is nothing at that folder now"`
	IsShowing bool `json:"showing,omitempty" jsonschema:"the vault the person is looking at, and the one every other tool works"`
}

// addVaultsTools gives an agent the list of vaults this installation holds and
// what a person does to it: add a folder, call one something else, take one off
// the list, and show another in the window.
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
	addVaultAdd(server, core)
	addVaultRename(server, core)
	addVaultForget(server, core)
	addVaultOpen(server, core)
}

func addVaultList(server *sdk.Server, core Core) {
	if core.Vaults.GetKnownVaults == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "vault_list",
		Title: "List vaults",
		Description: "Every vault this installation holds: what each is called, where its " +
			"folder is on this machine, and which one the person is looking at. Every " +
			"other tool works the vault marked `showing`, and no other. A vault whose " +
			"folder is gone is marked and stays on the list until somebody forgets it.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, struct {
		Vaults []Vault `json:"vaults"`
	}, error) {
		type out = struct {
			Vaults []Vault `json:"vaults"`
		}
		target, err := core.Vaults.GetKnownVaults.
			Execute(core.getCurrentVault().Vault.ID)
		if err != nil {
			return nil, out{}, err
		}
		list := make([]Vault, 0, len(target))
		for _, one := range target {
			list = append(list, newVault(one))
		}
		return nil, out{Vaults: list}, nil
	})
}

func addVaultAdd(server *sdk.Server, core Core) {
	if core.Vaults.Add == nil || core.Vaults.CheckRoot == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "vault_add",
		Title: "Add a vault",
		Description: "Put a folder on the list of vaults this installation holds. The " +
			"window goes on showing the vault it is showing, and `vault_open` is what " +
			"moves it. " +
			"Nothing already in the folder is moved or rewritten, and one file is " +
			"written into it: the vault's identity, under the application's own folder " +
			"there — `.numen/config.json`, unless this installation is configured to " +
			"another name. That identity is what the folder is recognised by wherever " +
			"it moves to. " +
			"Called with no path, this machine's own folder picker goes up in front of " +
			"the person and what they choose is added; a person who closes it has chosen " +
			"nothing, and the answer says so, which is not a failure and not worth a " +
			"second try. " +
			"Called with a path, the folder at that path becomes a vault: weigh that " +
			"before naming one, because this application reads every file under that " +
			"folder and an agent working the vault reads and writes every file under it. " +
			"Say which folder you are adding and why. A vault does not lie inside " +
			"another, and neither a filesystem root nor the person's home directory is a " +
			"folder somebody keeps notes in.",
	}, newHandler(func(ctx context.Context, in vaultAddInput) (vaultAddOutput, error) {
		return addVault(ctx, core, in)
	}))
}

type vaultAddInput struct {
	Path string `json:"path,omitempty" jsonschema:"the folder to add, as an absolute path on this machine; left out, the person picks one"`
	Name string `json:"name,omitempty" jsonschema:"what to call it; the folder's own name by default, numbered when another vault has that name"`
}

type vaultAddOutput struct {
	IsAdded bool   `json:"added"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Folder  string `json:"folder,omitempty"`
	Why     string `json:"why,omitempty" jsonschema:"why nothing was added, for an answer that added nothing"`
}

func addVault(ctx context.Context, core Core, in vaultAddInput) (vaultAddOutput, error) {
	if in.Path != "" {
		return addVaultAt(core, in.Path, in.Name)
	}
	if core.Vaults.PickFolder == nil {
		return vaultAddOutput{}, errNoDialog
	}
	picked, err := core.Vaults.PickFolder.Execute(ctx)
	if err != nil {
		return vaultAddOutput{}, err
	}
	if !picked.IsChosen {
		return vaultAddOutput{Why: "the person closed the picker and chose no folder"}, nil
	}
	return addVaultAt(core, picked.Path, in.Name)
}

var errNoDialog = errors.New("there is nobody here to pick a folder: name the one to add")

func addVaultAt(core Core, root, name string) (vaultAddOutput, error) {
	if err := core.Vaults.CheckRoot.Execute(root); err != nil {
		return vaultAddOutput{}, err
	}

	added, err := core.Vaults.Add.Execute(root, name)
	if err != nil {
		return vaultAddOutput{}, err
	}
	return vaultAddOutput{IsAdded: true, ID: string(added.ID), Name: added.Name, Folder: added.Path}, nil
}

func addVaultRename(server *sdk.Server, core Core) {
	if core.Vaults.Find == nil || core.Vaults.Rename == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "vault_rename",
		Title: "Rename a vault",
		Description: "What the person calls a vault. The folder keeps the name the " +
			"filesystem gives it and nothing on disk moves. A name another vault on the " +
			"list has fails, so the names in `vault_list` name one vault each.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to rename, addressed by its name, its folder, or the identity vault_list gives it"`
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
		v, err := core.Vaults.Find.Execute(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		renamed, err := core.Vaults.Rename.Execute(ctx, v, in.Name)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{ID: string(renamed.ID), Name: renamed.Name, Folder: renamed.Path}, nil
	})
}

func addVaultForget(server *sdk.Server, core Core) {
	if core.Vaults.Find == nil || core.Vaults.Unregister == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "vault_remove",
		Title: "Forget a vault",
		Description: "Take a vault off the list and out of the index. Its folder stays " +
			"where it is with everything in it, and adding it again brings back the same " +
			"vault under the same identity. The vault the person is looking at is " +
			"rejected, and so is the last vault an installation has. Nothing here deletes " +
			"a folder: taking a person's notes off their disk is theirs to ask for, in " +
			"front of them.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to forget, addressed by its name, its folder, or the identity vault_list gives it"`
	}) (*sdk.CallToolResult, struct {
		IsForgotten bool   `json:"forgotten"`
		Folder      string `json:"folder" jsonschema:"where the folder still is; vault_add on it brings the vault back"`
	}, error) {
		type out = struct {
			IsForgotten bool   `json:"forgotten"`
			Folder      string `json:"folder" jsonschema:"where the folder still is; vault_add on it brings the vault back"`
		}
		v, err := core.Vaults.Find.Execute(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		if core.isShown(v) {
			return nil, out{}, fmt.Errorf("%s is the vault the window is showing", v.Name)
		}
		if err := core.Vaults.Unregister.Execute(ctx, v); err != nil {
			return nil, out{}, err
		}
		return nil, out{IsForgotten: true, Folder: v.Path}, nil
	})
}

func addVaultOpen(server *sdk.Server, core Core) {
	if core.Vaults.Find == nil || core.Vaults.Opens == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "vault_open",
		Title: "Open a vault",
		Description: "Put another vault in front of the person, in the window they have " +
			"open. Nothing on disk moves and nothing is written into either vault; the " +
			"list of vaults records which one was opened last, and the window comes back " +
			"to it the next time the application starts. Your " +
			"session ends when the window turns: these tools are served for the vault " +
			"that is going and they go with it — the folder you were told about, the " +
			"notes you have looked up, this conversation. Whatever you were in the middle " +
			"of has to be asked again once the window is on the vault that arrived, and " +
			"the person is the one who asks. So call this when moving the person to " +
			"another vault is what was asked for, and never on the way to something else.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to show, addressed by its name, its folder, or the identity vault_list gives it"`
	}) (*sdk.CallToolResult, struct {
		IsOpening bool   `json:"opening"`
		Status    string `json:"status" jsonschema:"what is happening now, in words to say back to the person"`
	}, error) {
		type out = struct {
			IsOpening bool   `json:"opening"`
			Status    string `json:"status" jsonschema:"what is happening now, in words to say back to the person"`
		}
		v, err := core.Vaults.Find.Execute(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		if core.isShown(v) {
			return nil, out{}, fmt.Errorf("%s is the vault the window is already showing", v.Name)
		}

		// The swap stops the endpoint this call arrived on, and that waits for
		// the calls already taken. The answer goes back first, and the window
		// moves behind it, under a context this call's ending does not cancel.
		//
		// What comes back is reported where the swap is bound, so there is
		// nothing to carry back through an answer that has already gone.
		go func() { _ = core.Vaults.Opens(context.WithoutCancel(ctx), v) }()
		return nil, out{
			IsOpening: true,
			Status: fmt.Sprintf(
				"the window is moving to %s, and this session ends with the vault it was serving",
				v.Name),
		}, nil
	})
}

// newVault is one vault as an agent is told about it.
func newVault(one vaults.KnownVault) Vault {
	return Vault{
		ID:        string(one.Vault.ID),
		Name:      one.Vault.Name,
		Folder:    one.Vault.Path,
		IsMissing: one.IsMissing,
		IsShowing: one.IsCurrent,
	}
}

func (c Core) isShown(v domain.Vault) bool {
	return v.ID == c.getCurrentVault().Vault.ID
}
