package vault

import "github.com/jiva-studio/numen/libs/shared/note/domain"

// RemoteVault is a vault a plugin keeps remotely. VaultID is the vault of this installation it is
// connected to, and empty when it is connected to none.
type RemoteVault struct {
	RemoteID  string
	Name      string
	SizeBytes int64
	VaultID   domain.VaultID
}

// FolderState is what a path of the filesystem is: absent, a folder, and a folder that holds nothing.
type FolderState struct {
	Exists   bool
	IsFolder bool
	IsEmpty  bool
}

// DefaultFolderName is the name of the folder of a new vault whose name leaves nothing a folder can be named.
const DefaultFolderName = "Vault"
