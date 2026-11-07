package hosttest

import (
	"context"
	"errors"
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

// GetOpenedVault returns the vault OpenVault last opened, and false before any was.
func (v *Vault) GetOpenedVault() (domain.Vault, bool) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if v.world.openedVault == nil {
		return domain.Vault{}, false
	}
	return v.world.openedVault.vault, true
}

// GetVaultList lists the vaults GetVaults lists, each with whether its folder is gone.
func (v *Vault) GetVaultList(ctx context.Context) ([]host.VaultEntry, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkListMember(ctx); err != nil {
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

// AddVault adds an in-memory vault at the root, in addition to this vault. Its identity is its name,
// and a duplicate name gets a numeric suffix, as in the host.
func (v *Vault) AddVault(ctx context.Context, root, name string) (domain.Vault, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkListMember(ctx); err != nil {
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

// RenameVault renames this vault or another vault on its list.
func (v *Vault) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkListMember(ctx); err != nil {
		return domain.Vault{}, err
	}
	target, isKnown := v.findByID(id)
	if !isKnown {
		return domain.Vault{}, notesdomain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), host.ErrUnknownVault)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Vault{}, errors.New("the new name is empty; a vault needs a name")
	}
	if other, isTaken := v.findByName(name); isTaken && other != target {
		return domain.Vault{}, notesdomain.NewDetailError(
			fmt.Sprintf("the vault at %s already has the name %s", other.vault.Path, other.vault.Name), host.ErrVaultNameTaken)
	}
	target.vault.Name = name
	return target.vault, nil
}

// RemoveVault removes a vault from the list. A call on it afterwards returns host.ErrUnknownVault. The last
// vault and the vault last opened are rejected.
func (v *Vault) RemoveVault(ctx context.Context, id domain.VaultID) (string, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkListMember(ctx); err != nil {
		return "", err
	}
	target, isKnown := v.findByID(id)
	if !isKnown {
		return "", notesdomain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), host.ErrUnknownVault)
	}
	if len(v.world.members) == 1 {
		return "", notesdomain.NewDetailError(
			fmt.Sprintf("%s is the only vault; the last vault cannot be removed", target.vault.Name), host.ErrLastVault)
	}
	if v.world.openedVault == target {
		return "", notesdomain.NewDetailError(fmt.Sprintf("%s is the vault the window shows", target.vault.Name), host.ErrVaultActive)
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

// SetUnsavedChanges makes OpenVault return host.ErrUnsavedChanges, as a window whose pages have
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
	if err := v.checkListMember(ctx); err != nil {
		return err
	}
	target, isKnown := v.findByID(id)
	if !isKnown {
		return notesdomain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), host.ErrUnknownVault)
	}
	if v.world.hasUnsavedChanges {
		return host.ErrUnsavedChanges
	}
	v.world.openedVault = target
	return nil
}

// checkListMember returns an error for a call on the list of vaults made through a vault not on
// the list. The caller has the lock.
func (v *Vault) checkListMember(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v.unknown {
		return host.ErrUnknownVault
	}
	return nil
}

// findByID returns the vault on the list with the identity. The caller has the lock.
func (v *Vault) findByID(id domain.VaultID) (*Vault, bool) {
	for _, member := range v.world.members {
		if member.vault.ID == id {
			return member, true
		}
	}
	return nil, false
}

// findByName returns the vault on the list with the name, compared as the host compares names. The
// caller has the lock.
func (v *Vault) findByName(name string) (*Vault, bool) {
	foldedName := notesdomain.FoldName(name)
	for _, member := range v.world.members {
		if notesdomain.FoldName(member.vault.Name) == foldedName {
			return member, true
		}
	}
	return nil, false
}

// getFreeName returns the name, or the name with the lowest number from 2 that neither a name nor an
// identity on the list has. The caller has the lock.
func (v *Vault) getFreeName(name string) string {
	candidate := name
	for attempt := 2; v.isNameTaken(candidate); attempt++ {
		candidate = name + " " + strconv.Itoa(attempt)
	}
	return candidate
}

func (v *Vault) isNameTaken(name string) bool {
	_, isNamed := v.findByName(name)
	_, isIdentity := v.findByID(domain.VaultID(name))
	return isNamed || isIdentity
}

// checkOverlap rejects a folder that is a vault on the list, is inside one or contains one, with the
// text of the core. The caller has the lock.
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
