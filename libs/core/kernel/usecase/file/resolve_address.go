package file

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ResolveAddress finds the file an address names: plugin://<plugin id>/<path>, a file of that
// plugin's store of the vault, or any other text, a path of the vault.
type ResolveAddress struct {
	stat  *StatFile
	open  *OpenPluginFile
	types MediaTyper
}

// ErrNoPluginFiles is a plugin:// address where the installation keeps no plugin stores.
var ErrNoPluginFiles = errors.New("the installation keeps no files of plugins")

// NewResolveAddress resolves the vault.s files with stat and the plugins. files with open, and names the
// media type of a file with types, which may be nothing; without open a plugin:// address is
// ErrNoPluginFiles.
func NewResolveAddress(stat *StatFile, open *OpenPluginFile, types MediaTyper) *ResolveAddress {
	return &ResolveAddress{stat: stat, open: open, types: types}
}

// ResolvedFile is the file an address names: the address read, the stat of a file of the vault, and
// the size of the file and the media type it is played as, empty where none is known.
type ResolvedFile struct {
	Address   Address
	Stat      domain.FileStat
	Size      int64
	MediaType string
}

// Execute resolves the address over the vault. A malformed address is ErrInvalidAddress, and a file
// that is not there the error of its store.
func (u *ResolveAddress) Execute(ctx context.Context, v domain.Vault, text string) (ResolvedFile, error) {
	address, err := ParseAddress(text)
	if err != nil {
		return ResolvedFile{}, err
	}
	if !address.IsPluginFile() {
		stat, err := u.stat.Execute(ctx, v, address.Path)
		if err != nil {
			return ResolvedFile{}, err
		}
		return ResolvedFile{Address: address, Stat: stat, Size: stat.Size, MediaType: getMediaType(u.types, address.Path)}, nil
	}
	if u.open == nil {
		return ResolvedFile{}, ErrNoPluginFiles
	}
	opened, err := u.open.Execute(ctx, v, address.PluginID, address.Path)
	if err != nil {
		return ResolvedFile{}, err
	}
	if err := opened.Body.Close(); err != nil {
		return ResolvedFile{}, err
	}
	return ResolvedFile{Address: address, Size: opened.Size, MediaType: getMediaType(u.types, address.Path)}, nil
}
