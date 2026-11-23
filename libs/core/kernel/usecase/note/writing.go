package note

import "github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"

// NoteWriter is what changes a note on disk: a vault opened to write a note,
// and the lock that keeps two writers from changing the vault at once.
type NoteWriter interface {
	vaultport.NoteWriters
	vaultport.VaultLocks
}

// MoveWriter is what files a note somewhere else and writes the links that
// reached it again: a vault opened to move files and to write notes, and the
// vault's lock.
type MoveWriter interface {
	FileMovers
	vaultport.NoteWriters
	vaultport.VaultLocks
}
