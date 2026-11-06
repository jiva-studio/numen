package hosttest

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

var _ host.VaultList = (*Vault)(nil)

// SetMissing marks the vault as one whose folder is gone, as the list of vaults reports it. A vault
// kept on disk whose folder was removed is missing too.
func (v *Vault) SetMissing(isMissing bool) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.isMissing = isMissing
}

// GetOpenedVault is the vault OpenVault last opened, and false before any was.
func (v *Vault) GetOpenedVault() (domain.Vault, bool) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if v.world.opened == nil {
		return domain.Vault{}, false
	}
	return v.world.opened.vault, true
}

// GetVaultList lists the vaults GetVaults lists, each with whether its folder is gone.
func (v *Vault) GetVaultList(ctx context.Context) ([]host.VaultEntry, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gateList(ctx); err != nil {
		return nil, err
	}
	out := []host.VaultEntry{}
	for _, member := range v.world.members {
		if member.isShown {
			out = append(out, host.VaultEntry{Vault: member.vault, IsMissing: member.isMissing || member.reach() != nil})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// AddVault adds a vault held in memory at the root, beside this one. Its identity is its name, and a
// name another vault has takes a number, as the host gives it.
func (v *Vault) AddVault(ctx context.Context, root, name string) (domain.Vault, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gateList(ctx); err != nil {
		return domain.Vault{}, err
	}
	if !path.IsAbs(root) {
		return domain.Vault{}, notesdomain.NewDetailError(fmt.Sprintf("%s is not an absolute path", root), host.ErrVaultUnreadable)
	}
	root = path.Clean(root)
	if err := v.checkOverlap(root); err != nil {
		return domain.Vault{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = path.Base(root)
	}
	name = v.getFreeName(name)
	added := newVault(domain.Vault{ID: domain.VaultID(name), Name: name, Path: root}, newMemory())
	added.world = v.world
	v.world.members = append(v.world.members, added)
	return added.vault, nil
}

// RenameVault changes the name of a vault beside this one, or of this one.
func (v *Vault) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gateList(ctx); err != nil {
		return domain.Vault{}, err
	}
	target, isKnown := v.findMember(id)
	if !isKnown {
		return domain.Vault{}, notesdomain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), host.ErrUnknownVault)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Vault{}, domain.ErrInvalidName
	}
	if other, isTaken := v.findNamed(name); isTaken && other != target {
		return domain.Vault{}, notesdomain.NewDetailError(
			fmt.Sprintf("the vault at %s is already called %s", other.vault.Path, other.vault.Name), host.ErrVaultNameTaken)
	}
	target.vault.Name = name
	return target.vault, nil
}

// RemoveVault takes a vault off the list. A call on it afterwards gets host.ErrUnknownVault. The last
// vault and the vault last opened are refused.
func (v *Vault) RemoveVault(ctx context.Context, id domain.VaultID) (string, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gateList(ctx); err != nil {
		return "", err
	}
	target, isKnown := v.findMember(id)
	if !isKnown {
		return "", notesdomain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), host.ErrUnknownVault)
	}
	if len(v.world.members) == 1 {
		return "", notesdomain.NewDetailError(
			fmt.Sprintf("%s is the only vault; the last vault cannot be removed", target.vault.Name), host.ErrLastVault)
	}
	if v.world.opened == target {
		return "", notesdomain.NewDetailError(fmt.Sprintf("%s is the vault the window shows", target.vault.Name), host.ErrVaultShowing)
	}
	remaining := make([]*Vault, 0, len(v.world.members)-1)
	for _, member := range v.world.members {
		if member != target {
			remaining = append(remaining, member)
		}
	}
	v.world.members = remaining
	target.unknown = true
	return target.vault.Path, nil
}

// SetUnsavedChanges makes OpenVault return host.ErrUnsavedChanges, as a window whose pages hold
// unsaved changes does, and false lets it open vaults again.
func (v *Vault) SetUnsavedChanges(hasUnsavedChanges bool) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.world.hasUnsavedChanges = hasUnsavedChanges
}

// OpenVault records the vault as the one the window shows; GetOpenedVault returns it.
func (v *Vault) OpenVault(ctx context.Context, id domain.VaultID) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gateList(ctx); err != nil {
		return err
	}
	target, isKnown := v.findMember(id)
	if !isKnown {
		return notesdomain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), host.ErrUnknownVault)
	}
	if v.world.hasUnsavedChanges {
		return host.ErrUnsavedChanges
	}
	v.world.opened = target
	return nil
}

// gateList fails a call on the list of vaults made through a vault the list does not hold. The caller
// holds the lock.
func (v *Vault) gateList(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v.unknown {
		return host.ErrUnknownVault
	}
	return nil
}

// findMember is the vault of the identity on the list. The caller holds the lock.
func (v *Vault) findMember(id domain.VaultID) (*Vault, bool) {
	for _, member := range v.world.members {
		if member.vault.ID == id {
			return member, true
		}
	}
	return nil, false
}

// findNamed is the vault on the list called the name, compared as the host compares names. The
// caller holds the lock.
func (v *Vault) findNamed(name string) (*Vault, bool) {
	folded := notesdomain.FoldName(name)
	for _, member := range v.world.members {
		if notesdomain.FoldName(member.vault.Name) == folded {
			return member, true
		}
	}
	return nil, false
}

// getFreeName is the name, or the name with the lowest number from 2 that neither a name nor an
// identity on the list has. The caller holds the lock.
func (v *Vault) getFreeName(name string) string {
	candidate := name
	for attempt := 2; v.isNameTaken(candidate); attempt++ {
		candidate = name + " " + strconv.Itoa(attempt)
	}
	return candidate
}

func (v *Vault) isNameTaken(name string) bool {
	_, isNamed := v.findNamed(name)
	_, isIdentity := v.findMember(domain.VaultID(name))
	return isNamed || isIdentity
}

// checkOverlap refuses a folder that is a vault on the list, is inside one or contains one, in the
// sentence of the core. The caller holds the lock.
func (v *Vault) checkOverlap(root string) error {
	vaults := make(notesdomain.Vaults, 0, len(v.world.members))
	for _, member := range v.world.members {
		if member.vault.Path == root {
			return notesdomain.NewDetailError(fmt.Sprintf("%s is the vault %s", root, member.vault.Name), host.ErrVaultOverlaps)
		}
		vaults = append(vaults, member.vault)
	}
	if err := vaults.CheckOverlap(root); err != nil {
		return notesdomain.NewDetailError(err.Error(), host.ErrVaultOverlaps, err)
	}
	return nil
}
