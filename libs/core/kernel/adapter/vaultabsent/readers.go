// Package vaultabsent shows the files of a registry beside the files of a vault.
package vaultabsent

import (
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Readers opens the readers of an inner source and adds the absent files of each vault.
type Readers struct {
	inner  vaultport.VaultReaders
	absent vaultport.AbsentFiles
}

// NewReaders is the decorator over the inner readers and the registry.
func NewReaders(inner vaultport.VaultReaders, absent vaultport.AbsentFiles) Readers {
	return Readers{inner: inner, absent: absent}
}

// Open returns the reader of the vault with its absent files.
func (r Readers) Open(v domain.Vault) (vaultport.VaultReader, error) {
	inner, err := r.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return reader{inner: inner, absent: r.absent, vault: v}, nil
}
