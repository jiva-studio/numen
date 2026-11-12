package host

import (
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var (
	// ErrVaultUnreadable is returned when the folder to add is not a folder or cannot be read.
	ErrVaultUnreadable = errors.New("the folder cannot be read as a vault")
	// ErrVaultDuplicate is returned when the folder to add has the identity of a vault on the list
	// whose folder still has it.
	ErrVaultDuplicate = errors.New("the folder is a copy of a vault on the list")
	// ErrVaultOverlaps is returned when the folder to add is inside a vault on the list or contains one.
	ErrVaultOverlaps = errors.New("a vault cannot contain or be inside another vault")
	// ErrVaultNameTaken is returned when a rename uses the name of another vault.
	ErrVaultNameTaken = errors.New("another vault has this name")
	// ErrLastVault is returned when the vault to remove is the only vault of the installation.
	ErrLastVault = errors.New("the last vault cannot be removed")
	// ErrVaultActive is returned when the vault to remove is the one the window shows.
	ErrVaultActive = errors.New("the vault is open in the window")
	// ErrUnsavedChanges is returned by an open while the window prompts the user about unsaved
	// changes. The window keeps its vault.
	ErrUnsavedChanges = errors.New("the window has unsaved changes and is prompting the user; retry after the prompt closes")
)

// VaultEntry is a vault on the installation's list, and whether its folder is missing.
type VaultEntry struct {
	domain.Vault
	IsMissing bool
}
