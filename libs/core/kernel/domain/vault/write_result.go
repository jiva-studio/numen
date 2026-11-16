package vault

import "github.com/jiva-studio/numen/libs/note/domain"

// WriteResult is what a write of a note returns: the stat of the file now and the version of the vault.
// Version is the number of writes the vault has taken, this one included; a refused write takes none.
type WriteResult struct {
	Stat    domain.FileStat
	Version uint64
}
