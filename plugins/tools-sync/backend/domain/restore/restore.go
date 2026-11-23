// Package restore holds the rules for a sync server that was restored from a backup: how a device tells that
// the server went back, what it sends again and which logs it replays.
package restore

import (
	"bytes"
	"errors"
	"maps"
	"slices"

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
	switch {
	case len(memory.PinnedKey) > 0 && len(look.SignKey) > 0 && !bytes.Equal(memory.PinnedKey, look.SignKey):
		return KeyChanged
	case len(look.EpochID) > 0 && !bytes.Equal(memory.EpochID, look.EpochID):
		return Changed
	case look.HeadSeq < memory.Cursor:
		return WentBack
	}
	return Same
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
	common := min(len(cache.Entries), len(server.Entries))
	for i := range common {
		if !bytes.Equal(cache.Entries[i], server.Entries[i]) {
			return Diverged
		}
	}
	switch {
	case len(server.Entries) < len(cache.Entries):
		return ServerBehind
	case len(server.Entries) > len(cache.Entries):
		return ServerAhead
	}
	return Equal
}

// GetReplay returns the entries of the copy the server lacks, in order. It is empty when the server holds the
// whole copy or more, and ErrDiverged when the logs differ at an entry both hold.
func GetReplay(cache, server Journal) ([][]byte, error) {
	switch CompareJournals(cache, server) {
	case Diverged:
		return nil, ErrDiverged
	case ServerBehind:
		return cache.Entries[len(server.Entries):], nil
	case Equal, ServerAhead:
	}
	return nil, nil
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
	actions := make([]Action, 0, len(records)+len(heads))
	known := make(map[string]bool, len(records))
	for _, record := range records {
		known[string(record.FileID)] = true
		head, found := heads[string(record.FileID)]
		_, isLocal := local[record.Path]
		actions = append(actions, planRecord(record, head, found, isLocal))
	}
	for _, id := range slices.Sorted(maps.Keys(heads)) {
		if !known[id] {
			actions = append(actions, planUnknown(heads[id], local))
		}
	}
	return actions
}

func planRecord(record note.FileRecord, head Head, found, isLocal bool) Action {
	action := Action{Kind: Keep, FileID: record.FileID, Path: record.Path}
	isSame := found && bytes.Equal(head.Rev, record.HeadRev) && !head.IsBlobMissing
	switch {
	case isSame:
	case isLocal && !found:
		action.Kind = Recreate
	case isLocal:
		action.Kind, action.ParentRev = Send, head.Rev
	case found && !head.IsDeleted:
		action.Kind, action.ParentRev = Tombstone, head.Rev
	}
	return action
}

func planUnknown(head Head, local map[string]string) Action {
	action := Action{Kind: Keep, FileID: head.FileID, Path: head.Path}
	if _, isTaken := local[head.Path]; !isTaken && !head.IsDeleted && !head.IsBlobMissing {
		action.Kind = Pull
	}
	return action
}
