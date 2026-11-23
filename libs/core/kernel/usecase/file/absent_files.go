package file

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

var errNotImplemented = errors.New("not implemented")

// PutAbsentFiles lists files whose bytes are not on this device. It refuses a path that holds a file
// of another hash.
type PutAbsentFiles struct {
	readers vaultport.VaultReaders
	absent  vaultport.AbsentFiles
}

func NewPutAbsentFiles(readers vaultport.VaultReaders, absent vaultport.AbsentFiles) *PutAbsentFiles {
	return &PutAbsentFiles{readers: readers, absent: absent}
}

func (u *PutAbsentFiles) Execute(context.Context, domain.Vault, []vault.AbsentFile) error {
	return errNotImplemented
}

// FillFile places the bytes of a store file at an absent path once their SHA-256 equals the hash of
// the entry.
type FillFile struct {
	readers vaultport.VaultReaders
	absent  vaultport.AbsentFiles
}

func NewFillFile(readers vaultport.VaultReaders, absent vaultport.AbsentFiles) *FillFile {
	return &FillFile{readers: readers, absent: absent}
}

// Execute moves storeFile, a path of the file system, onto the vault path.
func (u *FillFile) Execute(context.Context, domain.Vault, string, string) error {
	return errNotImplemented
}

// EvictFile deletes the bytes of a file whose hash is the expected one and lists the file as absent.
type EvictFile struct {
	readers vaultport.VaultReaders
	absent  vaultport.AbsentFiles
}

func NewEvictFile(readers vaultport.VaultReaders, absent vaultport.AbsentFiles) *EvictFile {
	return &EvictFile{readers: readers, absent: absent}
}

// Execute drops the file at path for the provider.
func (u *EvictFile) Execute(context.Context, domain.Vault, string, string, string) error {
	return errNotImplemented
}
