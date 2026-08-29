package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
// what a person does to it: add a folder, call one something else, take one off
// the list, and show another in the window.
//
// Each tool is added where what it works through is there. A build that holds
// no list, or that cannot change one, serves what is left.
func addVaultsTools(server *sdk.Server, core Core) {
	addVaultList(server, core)
	addVaultAdd(server, core)
	addVaultRename(server, core)
	addVaultForget(server, core)
	addVaultOpen(server, core)
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

func addVaultAdd(server *sdk.Server, core Core) {
	if core.Adding == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_add",
		Title: "Make a folder into a vault",
		Description: "Put a folder on the list of vaults this installation holds. The " +
			"window goes on showing the vault it is showing, and `vault_open` is what " +
			"moves it. Nothing in the folder is moved or rewritten; what it gains is an " +
			"identity it keeps wherever it goes. " +
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
		Says   string `json:"says,omitempty" jsonschema:"why nothing was added, for an answer that added nothing"`
	}, error) {
		type out = struct {
			Added  bool   `json:"added"`
			ID     string `json:"id,omitempty"`
			Name   string `json:"name,omitempty"`
			Folder string `json:"folder,omitempty"`
			Says   string `json:"says,omitempty" jsonschema:"why nothing was added, for an answer that added nothing"`
		}
		root := in.Path
		if root == "" {
			if core.Choosing == nil {
				return nil, out{}, errors.New(
					"there is nobody here to pick a folder: name the one to add")
			}
			chosen, chose, err := core.Choosing.Choose(ctx, "Choose a folder for a vault", "")
			if err != nil {
				return nil, out{}, err
			}
			if !chose {
				return nil, out{Says: "the person closed the picker and chose no folder"}, nil
			}
			root = chosen
		}
		if !filepath.IsAbs(root) {
			return nil, out{}, fmt.Errorf("%s is not an absolute path, and a vault is named by one", root)
		}
		if err := keepsNotes(root); err != nil {
			return nil, out{}, err
		}

		added, err := core.Adding.Execute(root, in.Name)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Added: true, ID: added.ID, Name: added.Name, Folder: added.Path}, nil
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
			"where it is with everything in it, and adding it again brings back the same " +
			"vault under the same identity. The vault the person is looking at is " +
			"refused, and so is the last vault an installation has. Nothing here deletes " +
			"a folder: taking a person's notes off their disk is theirs to ask for, in " +
			"front of them.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to forget: its name, its folder, or the identity vault_list gives it"`
	}) (*sdk.CallToolResult, struct {
		Forgotten bool   `json:"forgotten"`
		Folder    string `json:"folder" jsonschema:"where the folder still is; vault_add on it brings the vault back"`
	}, error) {
		type out = struct {
			Forgotten bool   `json:"forgotten"`
			Folder    string `json:"folder" jsonschema:"where the folder still is; vault_add on it brings the vault back"`
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

func addVaultOpen(server *sdk.Server, core Core) {
	if core.Vaults == nil || core.Opens == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_open",
		Title: "Show another vault in the window",
		Description: "Put another vault in front of the person, in the window they have " +
			"open. Your session ends when it does: these tools are served for the vault " +
			"that is going and they go with it — the folder you were told about, the " +
			"notes you have looked up, this conversation. Whatever you were in the middle " +
			"of has to be asked again once the window is on the vault that arrived, and " +
			"the person is the one who asks. So call this when moving the person to " +
			"another vault is what was asked for, and never on the way to something else.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Vault string `json:"vault" jsonschema:"the vault to show: its name, its folder, or the identity vault_list gives it"`
	}) (*sdk.CallToolResult, struct {
		Opening bool   `json:"opening"`
		Says    string `json:"says"`
	}, error) {
		type out = struct {
			Opening bool   `json:"opening"`
			Says    string `json:"says"`
		}
		v, err := core.found(in.Vault)
		if err != nil {
			return nil, out{}, err
		}
		if v.ID == core.shown().Vault.ID {
			return nil, out{}, fmt.Errorf("%s is the vault the window is already showing", v.Name)
		}

		// The swap stops the endpoint this call arrived on, and that waits for
		// the calls already taken. The answer goes back first, and the window
		// moves behind it.
		go func() { _ = core.Opens(context.WithoutCancel(ctx), v) }()
		return nil, out{
			Opening: true,
			Says: fmt.Sprintf(
				"the window is moving to %s, and this session ends with the vault it was serving",
				v.Name),
		}, nil
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
