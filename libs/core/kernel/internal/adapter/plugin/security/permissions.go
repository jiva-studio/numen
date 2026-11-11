package security

import (
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Permissions is the security permissions the manifest of a plugin declares.
type Permissions struct {
	mu        sync.Mutex
	granted   []string
	validator VaultPathValidator
}

// NewPermissions returns the permissions of a plugin that holds none.
func NewPermissions() Permissions {
	return Permissions{validator: NewVaultPathValidator()}
}

// Set stores the security permissions declared by the plugin.
func (p *Permissions) Set(permissions []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	norm := make([]string, len(permissions))
	for i, one := range permissions {
		norm[i] = manifest.NormalizePermission(one)
	}
	p.granted = norm
}

// Has reports whether the plugin holds the given permission.
func (p *Permissions) Has(permission string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Contains(p.granted, manifest.NormalizePermission(permission))
}

// Check returns an error if the plugin lacks the requested permission.
func (p *Permissions) Check(permission string) error {
	if !p.Has(permission) {
		return plugin.ErrPermissionDenied
	}
	return nil
}

// ValidatePath checks that targetPath is within vaultRoot and the plugin has vault:read permission.
func (p *Permissions) ValidatePath(vaultRoot, targetPath string) (string, error) {
	if err := p.Check(manifest.PermissionVaultRead); err != nil {
		return "", err
	}
	return p.validator.ValidatePath(vaultRoot, targetPath)
}
