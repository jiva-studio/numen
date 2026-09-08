package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Vault is one vault this installation holds, as an agent is told about it.
type Vault struct {
	ID     string `json:"id" jsonschema:"the identity this vault keeps wherever its folder moves to"`
	Name   string `json:"name" jsonschema:"what the person calls it"`
	Folder string `json:"folder" jsonschema:"where the vault is on this machine"`
	// Missing is a folder that is not there to be read. The vault stays on the
	// list until somebody forgets it.
	Missing bool `json:"missing,omitempty" jsonschema:"there is nothing at that folder now"`
	Showing bool `json:"showing,omitempty" jsonschema:"the vault the person is looking at, and the one every other tool works"`
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
	if core.Vaults.Registry == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
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
		held, err := vaults.NewKnownVaults(core.Vaults.Registry, core.Readers).
			Execute(core.shown().Vault.ID)
		if err != nil {
			return nil, out{}, err
		}
		list := make([]Vault, 0, len(held))
		for _, one := range held {
			list = append(list, knownOf(one))
		}
		return nil, out{Vaults: list}, nil
	})
}

func addVaultAdd(server *sdk.Server, core Core) {
	if core.Vaults.Add == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
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
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path,omitempty" jsonschema:"the folder to add, as an absolute path on this machine; left out, the person picks one"`
		Name string `json:"name,omitempty" jsonschema:"what to call it; the folder's own name by default, numbered when another vault has that name"`
	}) (*sdk.CallToolResult, struct {
		Added  bool   `json:"added"`
		ID     string `json:"id,omitempty"`
		Name   string `json:"name,omitempty"`
		Folder string `json:"folder,omitempty"`
		Why    string `json:"why,omitempty" jsonschema:"why nothing was added, for an answer that added nothing"`
	}, error) {
		type out = struct {
			Added  bool   `json:"added"`
			ID     string `json:"id,omitempty"`
			Name   string `json:"name,omitempty"`
			Folder string `json:"folder,omitempty"`
			Why    string `json:"why,omitempty" jsonschema:"why nothing was added, for an answer that added nothing"`
		}
		root := in.Path
		if root == "" {
			if core.Vaults.FolderDialog == nil {
				return nil, out{}, errors.New(
					"there is nobody here to pick a folder: name the one to add")
			}
			chosen, chose, err := core.Vaults.FolderDialog.Choose(ctx, "Choose a folder for a vault", "")
			if err != nil {
				return nil, out{}, err
			}
			if !chose {
				return nil, out{Why: "the person closed the picker and chose no folder"}, nil
			}
			root = chosen
		}
		if !filepath.IsAbs(root) {
			return nil, out{}, fmt.Errorf("%s is not an absolute path, and a vault is named by one", root)
		}
		if err := keepsNotes(root); err != nil {
			return nil, out{}, err
		}

		added, err := core.Vaults.Add.Execute(root, in.Name)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Added: true, ID: string(added.ID), Name: added.Name, Folder: added.Path}, nil
	})
}

func addVaultRename(server *sdk.Server, core Core) {
	if core.Vaults.Registry == nil || core.Vaults.Rename == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_rename",
		Title: "Rename a vault",
		Description: "What the person calls a vault. The folder keeps the name the " +
			"filesystem gives it and nothing on disk moves. A name another vault on the " +
			"list has is refused, so the names in `vault_list` name one vault each.",
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
		v, err := core.Vaults.found(in.Vault)
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
	if core.Vaults.Registry == nil || core.Vaults.Forget == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_forget",
		Title: "Forget a vault",
		Description: "Take a vault off the list and out of the index. Its folder stays " +
			"where it is with everything in it, and adding it again brings back the same " +
			"vault under the same identity. The vault the person is looking at is " +
			"refused, and so is the last vault an installation has. Nothing here deletes " +
			"a folder: taking a person's notes off their disk is theirs to ask for, in " +
			"front of them.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to forget, addressed by its name, its folder, or the identity vault_list gives it"`
	}) (*sdk.CallToolResult, struct {
		Forgotten bool   `json:"forgotten"`
		Folder    string `json:"folder" jsonschema:"where the folder still is; vault_add on it brings the vault back"`
	}, error) {
		type out = struct {
			Forgotten bool   `json:"forgotten"`
			Folder    string `json:"folder" jsonschema:"where the folder still is; vault_add on it brings the vault back"`
		}
		v, err := core.Vaults.found(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		// The use cases are not told which vault is in front of the person, and
		// this is.
		if v.ID == core.shown().Vault.ID {
			return nil, out{}, fmt.Errorf("%s is the vault the window is showing", v.Name)
		}
		if err := core.Vaults.Forget.Execute(ctx, v); err != nil {
			return nil, out{}, err
		}
		return nil, out{Forgotten: true, Folder: v.Path}, nil
	})
}

func addVaultOpen(server *sdk.Server, core Core) {
	if core.Vaults.Registry == nil || core.Vaults.Opens == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
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
		Opening bool   `json:"opening"`
		Doing   string `json:"doing" jsonschema:"what is happening now, in words to say back to the person"`
	}, error) {
		type out = struct {
			Opening bool   `json:"opening"`
			Doing   string `json:"doing" jsonschema:"what is happening now, in words to say back to the person"`
		}
		v, err := core.Vaults.found(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		if v.ID == core.shown().Vault.ID {
			return nil, out{}, fmt.Errorf("%s is the vault the window is already showing", v.Name)
		}

		// The swap stops the endpoint this call arrived on, and that waits for
		// the calls already taken. The answer goes back first, and the window
		// moves behind it.
		go func() { _ = core.Vaults.Opens(context.WithoutCancel(ctx), v) }()
		return nil, out{
			Opening: true,
			Doing: fmt.Sprintf(
				"the window is moving to %s, and this session ends with the vault it was serving",
				v.Name),
		}, nil
	})
}

// found is the vault a name, a folder or an identity reaches on the list.
func (v Vaults) found(nameOrPath string) (domain.Vault, error) {
	if nameOrPath == "" {
		return domain.Vault{}, errors.New("name the vault, as vault_list gives it")
	}
	return vaults.NewFind(v.Registry).Execute(nameOrPath)
}

// knownOf is one vault as an agent is told about it.
func knownOf(one vaults.KnownVault) Vault {
	return Vault{
		ID:      string(one.Vault.ID),
		Name:    one.Vault.Name,
		Folder:  one.Vault.Path,
		Missing: one.Missing,
		Showing: one.Current,
	}
}

// keepsNotes refuses a root that is not a folder somebody keeps notes in: a
// filesystem root, and the person's home directory itself.
func keepsNotes(root string) error {
	at := oneName(root)
	if filepath.Dir(at) == at {
		return fmt.Errorf(
			"%s is the root of this filesystem, and a vault is a folder somebody keeps notes in", at)
	}
	if home, err := os.UserHomeDir(); err == nil && oneName(home) == at {
		return fmt.Errorf(
			"%s is the person's home directory, and a vault is a folder inside it", at)
	}
	return nil
}

// oneName is one name for a folder, whichever route reached it.
func oneName(path string) string {
	clean := filepath.Clean(path)
	if real, err := filepath.EvalSymlinks(clean); err == nil {
		return real
	}
	return clean
}
