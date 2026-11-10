// Package windowvaults lets the plugins that have vault:manage change the list of vaults and the
// vault the window shows. The window supplies each operation; the package only orders them.
package windowvaults

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// List is the vaults of the installation and the vault the window shows, as the window changes them.
type List struct {
	// Add registers the folder at the absolute root under the name, or under the folder's name.
	Add func(root, name string) (domain.Vault, error)
	// Find returns the vault the id names.
	Find func(id string) (domain.Vault, error)
	// Rename renames a vault.
	Rename func(ctx context.Context, v domain.Vault, name string) (domain.Vault, error)
	// Remove takes the vault out of the list and the index; the folder stays when isErasing is false.
	Remove func(ctx context.Context, id string, current domain.VaultID, isErasing bool) error
	// GetShown returns the vault the window shows.
	GetShown func() domain.Vault
	// Open shows the vault in the window and returns once the window shows it.
	Open func(ctx context.Context, v domain.Vault) error
	// CheckFolder reports whether the folder of the vault is missing.
	CheckFolder func(v domain.Vault) bool
}

// AddVault adds the folder at the absolute root. An empty name means the folder's name.
func (l List) AddVault(_ context.Context, root, name string) (domain.Vault, error) {
	return l.Add(root, name)
}

// RenameVault renames the vault the id names.
func (l List) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	v, err := l.Find(string(id))
	if err != nil {
		return domain.Vault{}, err
	}
	return l.Rename(ctx, v, name)
}

// RemoveVault removes the vault from the list and from the index, and does not delete its folder.
func (l List) RemoveVault(ctx context.Context, id domain.VaultID) error {
	return l.Remove(ctx, string(id), l.GetShown().ID, false)
}

// OpenVault shows the vault the id names in the window.
func (l List) OpenVault(ctx context.Context, id domain.VaultID) error {
	v, err := l.Find(string(id))
	if err != nil {
		return err
	}
	return l.Open(ctx, v)
}

// IsMissing reports whether the folder of the vault is missing.
func (l List) IsMissing(v domain.Vault) bool { return l.CheckFolder(v) }
