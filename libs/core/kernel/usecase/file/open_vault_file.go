package file

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNoVaultNamed is an address that names a vault this installation does not hold.
var ErrNoVaultNamed = errors.New("no vault of that name")

// VaultFinder finds the registered vault of an identity.
type VaultFinder interface {
	Execute(id string) (domain.Vault, error)
}

// MediaTyper returns the media type a path is served under, or nothing where it has none.
type MediaTyper interface {
	Execute(path string) string
}

func findVault(find VaultFinder, showing domain.Vault, id string) (domain.Vault, error) {
	if string(showing.ID) == id {
		return showing, nil
	}
	if find == nil {
		return domain.Vault{}, ErrNoVaultNamed
	}
	found, err := find.Execute(id)
	if err != nil {
		return domain.Vault{}, ErrNoVaultNamed
	}
	return found, nil
}

func getMediaType(types MediaTyper, path string) string {
	if types == nil {
		return ""
	}
	return types.Execute(path)
}

// VaultFile is a file open to read and the media type it is served under.
type VaultFile struct {
	OpenedFile
	MediaType string
}

// OpenVaultFile opens a file of the vault an address names, or a file in the store of a plugin of it.
type OpenVaultFile struct {
	find       VaultFinder
	open       *OpenFile
	openPlugin *OpenPluginFile
	types      MediaTyper
}

// NewOpenVaultFile returns the use case over the registry of vaults, the opening of a file, the opening of a
// file of a plugin, which may be nothing, and the table of media types.
func NewOpenVaultFile(find VaultFinder, open *OpenFile, openPlugin *OpenPluginFile, types MediaTyper) *OpenVaultFile {
	return &OpenVaultFile{find: find, open: open, openPlugin: openPlugin, types: types}
}

// Execute opens the file at the path of the vault with the id, which is the vault the window shows or a
// registered one; any other id is ErrNoVaultNamed. The caller closes the body. See OpenFile.Execute for the
// file stat.
func (u *OpenVaultFile) Execute(ctx context.Context, showing domain.Vault, id, path string, named domain.FileStat) (VaultFile, error) {
	target, err := findVault(u.find, showing, id)
	if err != nil {
		return VaultFile{}, err
	}
	opened, err := u.open.Execute(ctx, target, path, named)
	if err != nil {
		return VaultFile{}, err
	}
	return VaultFile{OpenedFile: opened, MediaType: getMediaType(u.types, opened.Path)}, nil
}

// VaultPluginFile is a file of a plugin's store open to read and the media type it is served under.
type VaultPluginFile struct {
	PluginFile
	MediaType string
}

// ExecutePluginFile opens the file at the path in the store of the plugin of the vault with the id, which is the vault
// the window shows or a registered one; any other id is ErrNoVaultNamed. The caller closes the body.
func (u *OpenVaultFile) ExecutePluginFile(ctx context.Context, showing domain.Vault, id, pluginID, path string) (VaultPluginFile, error) {
	target, err := findVault(u.find, showing, id)
	if err != nil {
		return VaultPluginFile{}, err
	}
	if u.openPlugin == nil {
		return VaultPluginFile{}, ErrNoPluginFiles
	}
	opened, err := u.openPlugin.Execute(ctx, target, pluginID, path)
	if err != nil {
		return VaultPluginFile{}, err
	}
	return VaultPluginFile{PluginFile: opened, MediaType: getMediaType(u.types, path)}, nil
}
