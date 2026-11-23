// Package transfer holds what the steps of a pass over a synced vault take and give: the files pulled, scanned
// and committed.
package transfer

import (
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

// CommitStatus is how the server took a file.
type CommitStatus int

const (
	// Committed means the revision is the head of its file now.
	Committed CommitStatus = iota + 1
	// ParentMismatch means the head of the file is not the revision the file was changed from.
	ParentMismatch
	// PathTaken means another file of the vault has the path.
	PathTaken
	// TooLarge means the sealed file is larger than the server stores.
	TooLarge
)

// Vault is the synced vault: its id on the server and its key.
type Vault struct {
	ID  []byte
	Key [32]byte
}

// File is a file to commit. A file the server knows has its FileID and the revision it was changed from. A file up
// to note.SmallFileLimit comes with its Content; a bigger one has Stat and is read in ranges.
type File struct {
	FileID     []byte
	ParentRev  []byte
	Path       string
	Content    []byte
	ModifiedAt int64
	Stat       *domain.FileStat
}

// CommitInput is the files to commit to a vault. VaultID names the vault on this device.
type CommitInput struct {
	VaultID domain.VaultID
	Vault   Vault
	Files   []File
}

// Move is a file that has another path now: its record and the path it moved to.
type Move struct {
	Record note.FileRecord
	To     string
}

// MoveInput is the moves to commit to a vault.
type MoveInput struct {
	Vault Vault
	Moves []Move
}

// RemoveInput is the records of the files removed from the vault, to commit as removals.
type RemoveInput struct {
	Vault   Vault
	Records []note.FileRecord
}

// Outcome is what became of one file. Head is the head of the file the server holds for ParentMismatch and for
// PathTaken, and ExistingFileID is the file that has the path for PathTaken. Record is what this device keeps of a
// Committed file.
type Outcome struct {
	Path           string
	Status         CommitStatus
	FileID         []byte
	RevID          []byte
	Head           *syncv1.StoredRevision
	ExistingFileID []byte
	Record         note.FileRecord
}

// ScanInput is the vault, the hash each synced file had when both computers last agreed, by path, and the records.
type ScanInput struct {
	VaultID domain.VaultID
	Bases   map[string]string
	Records []note.FileRecord
}

// Change is a file that differs from its base.
type Change struct {
	Path       string
	Hash       string
	Size       int64
	ModifiedAt int64
}

// Scan is what a look at the vault found: files that changed, files that have another path and records whose file
// is gone.
type Scan struct {
	Changes  []Change
	Moves    []Move
	Removals []note.FileRecord
}

// PullInput is the vault and the check of the author of a revision.
type PullInput struct {
	VaultID domain.VaultID
	Authors func(deviceSignKey []byte) bool
}

// PullResult is the files written, the files left as they were because both computers changed them, the number
// of revisions skipped because their author or their content could not be verified, whether the server asked for
// a full comparison.
type PullResult struct {
	Applied    []string
	Conflicts  []string
	Unverified int
	IsReset    bool
}

// UploadInput names the file to send in one body or in parts.
type UploadInput struct {
	Vault     domain.VaultID
	RemoteID  []byte
	Path      string
	Stat      domain.FileStat
	FileID    []byte
	ParentRev []byte
}

// UploadResult is what the revision of an uploaded file names.
type UploadResult struct {
	BlobID   []byte
	BlobHash [32]byte
	BlobSize uint64
	Key      [32]byte
}

// DownloadInput names the file to bring down. Baseline is the stat of the local file the download replaces, nil
// when the path must be free.
type DownloadInput struct {
	Vault    domain.VaultID
	RemoteID []byte
	FileID   []byte
	Path     string
	BlobID   []byte
	BlobHash []byte
	BlobSize uint64
	Key      [32]byte
	Hash     string
	Size     int64
	HeadRev  []byte
	Baseline *domain.FileStat
}

// Direction is which way removals go: sent to the server or applied in the vault.
type Direction string

const (
	// DirectionSend is the removal of files the person removed.
	DirectionSend Direction = "send"
	// DirectionApply is the removal of files another computer removed.
	DirectionApply Direction = "apply"
)

// Verdict is what the guard says of removals.
type Verdict int

const (
	// Go means the removals may go on.
	Go Verdict = iota + 1
	// Pause means the pass leaves the removals until the person answers.
	Pause
	// Keep means the person answered to keep the files: the removals are dropped.
	Keep
)

// Removal is a file a pass is about to remove on the server or in the vault. RevID is the revision of the
// removal for Direction apply.
type Removal struct {
	FileID   []byte
	Path     string
	BaseHash string
	RevID    []byte
}

// GuardInput is the removals a pass is about to make and the number of files it tracks.
type GuardInput struct {
	Vault      domain.VaultID
	Direction  Direction
	Candidates []Removal
	Tracked    int
}

var (
	// ErrIntegrity means the bytes the server sent are not the blob its revision names.
	ErrIntegrity = errors.New("a file from the sync server is damaged")
	// ErrBlobMissing means the server has no blob for a revision.
	ErrBlobMissing = errors.New("the server has no blob for a revision")
)
