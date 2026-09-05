package vault

import (
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// FolderMissing answers whether a vault's folder is no longer where the list
// says it is.
//
// The list remembers where a vault was last seen and the vault carries the
// identity, so a folder that has gone leaves the vault on the list, marked. It
// is one question with one answer, and every screen that shows the list asks it
// here.
type FolderMissing struct {
	Readers port.VaultReaders
}

// NewFolderMissing is what the question is asked through: the vault the folder
// is opened as.
//
// It is named here because a build that cannot open a vault would answer that
// every folder is gone, which is a person told their notes have vanished.
func NewFolderMissing(readers port.VaultReaders) FolderMissing {
	return FolderMissing{Readers: readers}
}

// Execute reports whether the folder is gone. A path holding something that is
// not a folder is gone as well: it is not a vault to be read either way.
func (u FolderMissing) Execute(v domain.Vault) bool {
	_, err := u.Readers.Open(v)
	return err != nil
}
