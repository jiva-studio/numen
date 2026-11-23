// Package transfer holds what the steps of a pass over a synced vault take and give: the notes pulled, scanned
// and committed.
package transfer

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// CommitStatus is how the server took a note.
type CommitStatus int

const (
	// Committed means the revision is the head of its file now.
	Committed CommitStatus = iota + 1
	// ParentMismatch means the head of the file is not the revision the note was changed from.
	ParentMismatch
	// PathTaken means another file of the vault has the path.
	PathTaken
	// TooLarge means the sealed note is larger than the server stores.
	TooLarge
)

// EpochKey is a key of the synced vault made under the account keys of an epoch.
type EpochKey struct {
	Epoch uint32
	Key   [32]byte
}

// Vault is the synced vault: its id on the server, its key of epoch 0, which keys the hashes of paths, and its keys of
// later epochs by ascending epoch.
type Vault struct {
	ID    []byte
	Key   [32]byte
	Later []EpochKey
}

func (v Vault) GetNewest() EpochKey {
	if len(v.Later) == 0 {
		return EpochKey{Epoch: 0, Key: v.Key}
	}
	return v.Later[len(v.Later)-1]
}

// File is a note to commit. A note the server knows has its FileID and the revision it was changed from.
type File struct {
	FileID     []byte
	ParentRev  []byte
	Path       string
	Content    []byte
	ModifiedAt int64
}

// CommitInput is the notes to commit to a vault.
type CommitInput struct {
	Vault Vault
	Files []File
}

// Outcome is what became of one note. Head is the head of the file the server holds for ParentMismatch and for
// PathTaken, and ExistingFileID is the file that has the path for PathTaken.
type Outcome struct {
	Path           string
	Status         CommitStatus
	FileID         []byte
	RevID          []byte
	Head           *syncv1.StoredRevision
	ExistingFileID []byte
}

// ScanInput is the vault and the hash each synced note had when both computers last agreed, by path.
type ScanInput struct {
	VaultID domain.VaultID
	Bases   map[string]string
}

// Change is a note that differs from its base.
type Change struct {
	Path       string
	Hash       string
	Size       int64
	ModifiedAt int64
}

// PullInput is the vault and the check of the author of a revision.
type PullInput struct {
	VaultID domain.VaultID
	Authors func(deviceSignKey []byte) bool
}

// PullResult is the notes written, the notes left as they were because both computers changed them, and the
// number of revisions skipped because their author or their content could not be verified.
type PullResult struct {
	Applied    []string
	Conflicts  []string
	Unverified int
}
