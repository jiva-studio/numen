// Package restore holds the rules for a sync server that was restored from a backup: how a device tells that
// the server went back, what it sends again and which logs it replays.
package restore

import (
	"errors"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

// Verdict is what one look at the server says about its history.
type Verdict int

const (
	// Same means the server is where this device saw it last.
	Same Verdict = iota + 1
	// Changed means the server announced a restore this device has not run yet.
	Changed
	// KeyChanged means the key that signs restore epochs is not the key this device pinned.
	KeyChanged
	// WentBack means the server holds an earlier state and has announced nothing.
	WentBack
)

// Memory is what this device remembers of the server: the epoch it ran last, the position in the changes and the
// pinned key that signs epochs. An empty key is none pinned yet.
type Memory struct {
	EpochID   []byte
	Cursor    uint64
	PinnedKey []byte
}

// Look is what one answer of the server says: the epoch id, the highest sequence number of the vault and, when the
// epoch was fetched, the key that signs it.
type Look struct {
	EpochID []byte
	HeadSeq uint64
	SignKey []byte
}

// GetVerdict tells what the look means. A key that differs from the pinned one wins over everything, then a
// different epoch id, then a head below the cursor.
func GetVerdict(memory Memory, look Look) Verdict {
	return 0
}

// Kind names a log the device keeps a copy of.
type Kind string

const (
	// KindCatalog is the catalog of the vaults of the account.
	KindCatalog Kind = "catalog"
	// KindChain is the chain of the devices of the account.
	KindChain Kind = "chain"
)

// Journal is the signed entries of one log in order, as the device last verified them.
type Journal struct {
	Kind    Kind
	Entries [][]byte
}

// Relation is how the log on the server stands to the copy of the device.
type Relation int

const (
	// Equal means the server holds the same entries.
	Equal Relation = iota + 1
	// ServerBehind means the server holds a strict prefix of the copy.
	ServerBehind
	// ServerAhead means the copy is a strict prefix of the log on the server.
	ServerAhead
	// Diverged means the logs differ at an entry they both hold.
	Diverged
)

// ErrDiverged means the log on the server differs from the copy at an entry both hold.
var ErrDiverged = errors.New("the log on the sync server differs from the copy of this device")

// CompareJournals tells how the log on the server stands to the copy.
func CompareJournals(cache, server Journal) Relation {
	return 0
}

// GetReplay returns the entries of the copy the server lacks, in order. It is empty when the server holds the
// whole copy or more, and ErrDiverged when the logs differ at an entry both hold.
func GetReplay(cache, server Journal) ([][]byte, error) {
	return nil, errors.New("not implemented")
}

// ActionKind is what a recovery does with one note.
type ActionKind int

const (
	// Keep does nothing for the note.
	Keep ActionKind = iota + 1
	// Send commits the local content on top of the head the server holds.
	Send
	// Recreate commits the local content as a new file of the known id, with no parent, for a file the server lost.
	Recreate
	// Tombstone commits a deletion on top of the head the server holds.
	Tombstone
	// Pull reads a note this device never knew from the server.
	Pull
)

// Head is the head of a file as the server holds it. IsBlobMissing is true when the server lost the body.
type Head struct {
	FileID        []byte
	Rev           []byte
	Path          string
	IsDeleted     bool
	IsBlobMissing bool
}

// Action is what a recovery does with a file. ParentRev is the head the commit goes on top of, empty for Recreate.
type Action struct {
	Kind      ActionKind
	FileID    []byte
	Path      string
	ParentRev []byte
}

// PlanRecovery decides what to do with each file: records are what this device knows of its synced notes, heads
// are the heads the server holds by file id, and local is the hash of each local note by path. The content of the
// device is the truth: a note on the device is never replaced by the server's, and only a note this device never
// knew is read from the server.
func PlanRecovery(records []note.FileRecord, heads map[string]Head, local map[string]string) []Action {
	return nil
}
