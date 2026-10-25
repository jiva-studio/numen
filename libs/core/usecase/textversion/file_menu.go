package textversion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// FileMenu is the menu of a vault file as its text versions make it, for the
// file as it is now.
type FileMenu struct {
	versions *Versions
	readers  vaultport.VaultReaders
}

// NewFileMenu is the menu the versions make of the files the readers open.
func NewFileMenu(versions *Versions, readers vaultport.VaultReaders) (*FileMenu, error) {
	if versions == nil || readers == nil {
		return nil, errors.New("a file menu needs text versions and vault readers")
	}
	return &FileMenu{versions: versions, readers: readers}, nil
}

// GetMenu is what the file's menu offers.
func (u *FileMenu) GetMenu(ctx context.Context, v domain.Vault, path string) ([]MenuEntry, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return u.versions.GetMenu(ctx, v, path, domain.HashContent(raw))
}

// RunMenuEntry does what an entry of the file's menu offers, for a person.
func (u *FileMenu) RunMenuEntry(ctx context.Context, v domain.Vault, path string, entry MenuEntry) error {
	return u.versions.RunMenuEntry(ctx, v, path, entry)
}

// The errors of a menu entry the file's menu does not hold, for a driving adapter.
var (
	ErrNoText           = textport.ErrNoText
	ErrNotTextPlugin    = textport.ErrNotTextPlugin
	ErrInvalidVersion   = textport.ErrInvalidVersion
	ErrInputNotAccepted = textport.ErrInputNotAccepted
)
