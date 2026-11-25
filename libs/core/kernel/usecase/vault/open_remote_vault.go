package vault

import (
	"context"
	"errors"
	"fmt"
	"time"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// NewVaultPaths is the path work of the machine for the folder of a new vault.
type NewVaultPaths interface {
	// CleanPath returns the shortest path of the machine that names the same folder.
	CleanPath(path string) string
	// GetParentPath returns the path of the folder that holds the path. A disk root is its own parent.
	GetParentPath(path string) string
	// JoinPath returns the path of the machine made of the elements.
	JoinPath(elements ...string) string
}

// NewVaultFolders is what the filesystem does for the folder of a new vault.
type NewVaultFolders interface {
	// GetFolderState returns whether the path exists, whether it is a folder and whether the folder is empty.
	GetFolderState(path string) (vaultdomain.FolderState, error)
	// CreateFolder makes the folder at the path. Its parent exists.
	CreateFolder(path string) error
	// RemoveCreated removes what the folder holds that was made at or after since. A folder that holds
	// anything older keeps all of it and is no error.
	RemoveCreated(path string, since time.Time) error
	// RemoveEmptyFolder removes the folder when it holds nothing. A folder that holds something stays and is
	// no error.
	RemoveEmptyFolder(path string) error
}

// UserFolders are the folders of the person on this machine.
type UserFolders interface {
	GetHomeFolder() (string, error)
	GetDocumentsFolder() (string, error)
}

// OpenRemoteVault makes the local folder of a remote vault, registers it as a vault and connects the remote
// vault to it.
type OpenRemoteVault struct {
	vaults   vaultport.VaultList
	remover  vaultport.VaultRemover
	paths    NewVaultPaths
	folders  NewVaultFolders
	user     UserFolders
	add      Add
	provider RemoteVaultProvider
	now      machineport.Clock
}

// NewOpenRemoteVault creates the opening over the list this installation keeps and its remover, the
// places, Add, the provider and the clock.
func NewOpenRemoteVault(
	vaults vaultport.VaultList, remover vaultport.VaultRemover, places RemoteVaultPlaces, add Add,
	provider RemoteVaultProvider, now machineport.Clock,
) *OpenRemoteVault {
	return &OpenRemoteVault{
		vaults: vaults, remover: remover, paths: places.Paths, folders: places.Folders, user: places.User, add: add, provider: provider, now: now,
	}
}

// RemoteVaultPlaces is where a remote vault may go: how paths are written, the folders of this machine and the folders of the person.
type RemoteVaultPlaces struct {
	Paths   NewVaultPaths
	Folders NewVaultFolders
	User    UserFolders
}

// Execute returns the vault the remote vault is connected to at the path. The path is absolute, its parent
// exists, and it is no folder with anything in it, no disk or home folder, and no folder inside another
// vault. A failure after the folder was made removes the vault from the list again, and removes the folder
// when this call made it and it holds nothing else.
func (u *OpenRemoteVault) Execute(ctx context.Context, remoteID, path string) (id domain.VaultID, err error) {
	path = u.paths.CleanPath(path)
	state, err := u.checkPath(path)
	if err != nil {
		return "", err
	}
	since := u.now.Now()
	if !state.Exists {
		if err := u.folders.CreateFolder(path); err != nil {
			return "", fmt.Errorf("create the folder %s: %w", path, err)
		}
	}
	var added domain.Vault
	defer func() {
		if err != nil {
			err = errors.Join(err, u.undo(path, since, !state.Exists, added.ID))
		}
	}()
	added, err = u.add.Execute(path, "")
	if err != nil {
		return "", err
	}
	if err := u.provider.ConnectRemoteVault(ctx, remoteID, added.ID); err != nil {
		return "", err
	}
	return added.ID, nil
}

func (u *OpenRemoteVault) checkPath(path string) (vaultdomain.FolderState, error) {
	if err := u.checkPlace(path); err != nil {
		return vaultdomain.FolderState{}, err
	}
	state, err := u.checkFolder(path)
	if err != nil {
		return vaultdomain.FolderState{}, err
	}
	listed, err := u.vaults.List()
	if err != nil {
		return vaultdomain.FolderState{}, err
	}
	if err := domain.Vaults(listed).CheckOverlap(path); err != nil {
		return vaultdomain.FolderState{}, err
	}
	return state, nil
}

// checkPlace refuses a path that is no absolute path, a whole disk or the home folder.
func (u *OpenRemoteVault) checkPlace(path string) error {
	if path == "." || !domain.IsAbsPath(path) {
		return newInvalidPath("%q is not an absolute path", path)
	}
	if u.paths.GetParentPath(path) == path {
		return newInvalidPath("%s is a whole disk; a vault is a folder of its own", path)
	}
	home, err := u.user.GetHomeFolder()
	if err != nil {
		return err
	}
	if path == u.paths.CleanPath(home) {
		return newInvalidPath("%s is the home folder; a vault is a folder of its own", path)
	}
	return nil
}

// checkFolder reads the folder of the path and refuses one that cannot be filled: no parent, a file, or a folder with things in it.
func (u *OpenRemoteVault) checkFolder(path string) (vaultdomain.FolderState, error) {
	parentPath := u.paths.GetParentPath(path)
	parent, err := u.folders.GetFolderState(parentPath)
	if err != nil {
		return vaultdomain.FolderState{}, err
	}
	if !parent.Exists || !parent.IsFolder {
		return vaultdomain.FolderState{}, newInvalidPath("%s is not a folder that exists", parentPath)
	}
	state, err := u.folders.GetFolderState(path)
	if err != nil {
		return vaultdomain.FolderState{}, err
	}
	if state.Exists && !state.IsFolder {
		return vaultdomain.FolderState{}, newInvalidPath("%s is a file", path)
	}
	if state.Exists && !state.IsEmpty {
		return vaultdomain.FolderState{}, newInvalidPath("%s is not empty; a remote vault is opened in a new or an empty folder", path)
	}
	return state, nil
}

func (u *OpenRemoteVault) undo(path string, since time.Time, isFolderMade bool, id domain.VaultID) error {
	var failures []error
	if id != "" {
		failures = append(failures, u.remover.Remove(id))
	}
	failures = append(failures, u.folders.RemoveCreated(path, since))
	if isFolderMade {
		failures = append(failures, u.folders.RemoveEmptyFolder(path))
	}
	return errors.Join(failures...)
}

func newInvalidPath(format string, args ...any) error {
	return domain.NewDetailError(fmt.Sprintf(format, args...), vaultdomain.ErrInvalidPath)
}
