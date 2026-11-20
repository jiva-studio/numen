package file

import "github.com/jiva-studio/numen/libs/shared/note/domain"

// VaultVersions counts the writes a vault has taken, so a caller can tell which of two answers about its
// files is the newer.
type VaultVersions interface {
	// GetVersion returns the number of writes the vault has taken since the core started. A write raises
	// it only when it succeeds.
	GetVersion(v domain.Vault) uint64
}
