package vault

import (
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// FolderCheck answers whether a vault's folder is no longer where the list
// says it is.
//
// The list remembers where a vault was last seen and the vault carries the
// identity, so a folder that has gone leaves the vault on the list, marked. It
// is one question with one answer, and every screen that shows the list asks it
// here.
type FolderCheck struct {
	Readers port.VaultReaders
}

// NewFolderCheck is what the question is asked through: the vault the folder
// is opened as.
//
// It is named here because a build that cannot open a vault would answer that
// every folder is gone, which is a person told their notes have vanished.
func NewFolderCheck(readers port.VaultReaders) FolderCheck {
	return FolderCheck{Readers: readers}
}

// Execute reports whether the folder is gone. A path holding something that is
// not a folder is gone as well: it is not a vault to be read either way.
//
// A build with nothing to open a vault through says no folder is gone. It knows
// nothing about the disk, and answering that every vault has vanished is a
// person told their notes are gone by a build that never looked.
func (u FolderCheck) Execute(v domain.Vault) bool {
	if u.Readers == nil {
		return false
	}
	_, err := u.Readers.Open(v)
	return err != nil
}
