package note

import "github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"

// NoteWriting is what changes a note on disk: a vault opened to write a note,
// and the lock that keeps two writers from changing the vault at once.
type NoteWriting interface {
	vaultport.NoteWriters
	vaultport.VaultLocks
}

// MoveWriting is what files a note somewhere else and writes the links that
// reached it again: a vault opened to move files and to write notes, and the
// vault's lock.
type MoveWriting interface {
	vaultport.FileMovers
	vaultport.NoteWriters
	vaultport.VaultLocks
}
