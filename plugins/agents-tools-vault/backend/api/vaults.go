package api

// VaultEntry is a vault of the installation.
type VaultEntry struct {
	ID        string `json:"id" jsonschema:"vault identifier, which stays the same when its folder moves"`
	Name      string `json:"name" jsonschema:"vault display name"`
	Path      string `json:"path" jsonschema:"vault root directory on this machine"`
	IsCurrent bool   `json:"isCurrent" jsonschema:"true for the current vault"`
	IsMissing bool   `json:"isMissing,omitempty" jsonschema:"true when the vault folder does not exist"`
}

type VaultListOutput struct {
	Vaults []VaultEntry `json:"vaults"`
}

type VaultProblemListInput struct {
	Checks []string `json:"checks,omitempty" jsonschema:"the names of the checks to run, as the description lists them; the default checks when left out"`
}

// VaultProblemItem is a problem found in one file of the vault.
type VaultProblemItem struct {
	Path       string   `json:"path" jsonschema:"path of the file with the problem"`
	Check      string   `json:"check" jsonschema:"the check that found the problem"`
	Detail     string   `json:"detail" jsonschema:"description of the problem"`
	Address    string   `json:"address,omitempty" jsonschema:"the link target as written, for the ambiguous and dangling checks"`
	Candidates []string `json:"candidates,omitempty" jsonschema:"paths of the notes an ambiguous link could resolve to"`
}

type VaultProblemListOutput struct {
	Problems []VaultProblemItem `json:"problems"`
	Checks   []string           `json:"checks" jsonschema:"the checks this result covers"`
}

type VaultAddInput struct {
	Path string `json:"path" jsonschema:"absolute path of the folder on this machine"`
	Name string `json:"name,omitempty" jsonschema:"display name of the vault; the folder's name when left out"`
}

type VaultAddOutput struct {
	ID   string `json:"id" jsonschema:"identifier of the new vault"`
	Name string `json:"name" jsonschema:"display name of the new vault"`
	Path string `json:"path" jsonschema:"root directory of the new vault"`
}

type VaultRenameInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, path or identifier, as vault_list returns them"`
	Name  string `json:"name" jsonschema:"the new display name"`
}

type VaultRenameOutput struct {
	ID   string `json:"id" jsonschema:"identifier of the vault"`
	Name string `json:"name" jsonschema:"display name of the vault after the rename"`
	Path string `json:"path" jsonschema:"root directory of the vault"`
}

type VaultRemoveInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, path or identifier, as vault_list returns them"`
}

type VaultRemoveOutput struct {
	Path string `json:"path" jsonschema:"root directory of the vault, which is not deleted"`
}

type VaultOpenInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, path or identifier, as vault_list returns them"`
}

type VaultOpenOutput struct {
	ID   string `json:"id" jsonschema:"identifier of the vault the window opens"`
	Name string `json:"name" jsonschema:"display name of the vault the window opens"`
	Path string `json:"path" jsonschema:"root directory of the vault the window opens"`
}

// VaultInfoInput is the input of the vault_get tool.
type VaultInfoInput struct {
	Vault string `json:"vault,omitempty" jsonschema:"the vault's name, path or identifier, as vault_list returns them; the vault of the call when left out"`
}

// VaultInfoOutput is the output of the vault_get tool.
type VaultInfoOutput struct {
	ID        string `json:"id" jsonschema:"vault unique identifier"`
	Name      string `json:"name" jsonschema:"vault display name"`
	Path      string `json:"path" jsonschema:"vault root directory path on disk"`
	NoteCount int    `json:"noteCount" jsonschema:"total number of indexed notes in the vault"`
}

type VaultListInput struct{}
