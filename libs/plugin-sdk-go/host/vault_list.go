package host

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

var (
	// ErrVaultUnreadable is what an added folder gets when it is not a folder or cannot be read.
	ErrVaultUnreadable = errors.New("host: the folder is not a folder or cannot be read")
	// ErrVaultCopy is what an added folder gets when it carries the identity of a vault on the list
	// whose folder still carries it.
	ErrVaultCopy = errors.New("host: the folder is a copy of a vault on the list")
	// ErrVaultOverlaps is what an added folder gets when it lies inside a vault on the list or holds one.
	ErrVaultOverlaps = errors.New("host: the folder lies inside a vault on the list or holds one")
	// ErrVaultNameTaken is what a rename gets when another vault has the name.
	ErrVaultNameTaken = errors.New("host: another vault has the name")
	// ErrLastVault is what a removal gets for the only vault of the installation.
	ErrLastVault = errors.New("host: the vault is the only one of the installation")
	// ErrVaultShowing is what a removal gets for the vault the window shows.
	ErrVaultShowing = errors.New("host: the window shows the vault")
	// ErrVaultAsking is what an open gets when a tab holds changes the person has to decide on, so the
	// window kept its vault.
	ErrVaultAsking = errors.New("host: a tab holds changes the person has to decide on")
)

// KnownVault is a vault on the installation's list, and whether nothing is at its folder.
type KnownVault struct {
	domain.Vault
	IsMissing bool
}

// VaultList is the installation's list of vaults and the vault the window shows. GetVaultList needs
// vault:read; the other calls need vault:manage. A vault that is not on the list is ErrUnknownVault.
type VaultList interface {
	GetVaultList(ctx context.Context) ([]KnownVault, error)
	// AddVault adds the folder at the absolute root. An empty name takes the folder's name.
	AddVault(ctx context.Context, root, name string) (domain.Vault, error)
	RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error)
	// RemoveVault takes the vault off the list and out of the index, and returns its folder, which
	// stays on the disk.
	RemoveVault(ctx context.Context, id domain.VaultID) (folder string, err error)
	// OpenVault makes the window show the vault. It returns before the window changes vault.
	OpenVault(ctx context.Context, id domain.VaultID) error
}
