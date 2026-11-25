// Package transfer holds what the steps of a pass over a synced vault take and give: the files pulled, scanned
// and committed.
package transfer

import (
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
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
	// CaseClash means another file of the vault has a name that differs from the path in letter case only.
	CaseClash
	// TooLarge means the sealed file is larger than the server stores.
	TooLarge
	// StaleEpoch means the share the note belongs to moved to another epoch, so the device must read the share log.
	StaleEpoch
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

func (v Vault) GetKey(epoch uint32) ([32]byte, bool) {
	if epoch == 0 {
		return v.Key, true
	}
	for _, one := range v.Later {
		if one.Epoch == epoch {
			return one.Key, true
		}
	}
	return [32]byte{}, false
}

// File is a file to commit. A file the server knows has its FileID and the revision it was changed from. A file up
// to the small file limit of the note package comes with its Content; a bigger one has Stat and is read in ranges.
type File struct {
	FileID        []byte
	ParentRev     []byte
	ParentVersion uint64
	Path          string
	Content       []byte
	ModifiedAt    int64
	Stat          *domain.FileStat
}

// Move is a file that has another path now: its record and the path it moved to.
type Move struct {
	Record FileRecord
	To     string
}

// RemoveInput is the records of the files removed from the vault, to commit as removals.
type RemoveInput struct {
	Vault   Vault
	Records []FileRecord
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
	Record         FileRecord
}

// ScanInput is the vault, the hash each synced file had when both computers last agreed, by path, and the records.
type ScanInput struct {
	VaultID domain.VaultID
	Bases   map[string]string
	Records []FileRecord
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
	Removals []FileRecord
}

// NewEmptyRecord returns the record of a file that has none.
func NewEmptyRecord() FileRecord {
	return FileRecord{FileID: nil, Path: "", BaseHash: "", HeadRev: nil, Size: 0, ModifiedAt: 0, Kind: "", BlobID: nil, BlobHash: nil, BlobSize: 0, SealedKey: nil, Version: 0}
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
	FileID   []byte `json:"fileId"`
	Path     string `json:"path"`
	BaseHash string `json:"baseHash"`
	RevID    []byte `json:"revId"`
}

// GuardInput is the removals a pass is about to make and the number of files it tracks.
type GuardInput struct {
	Vault      domain.VaultID
	Direction  Direction
	Candidates []Removal
	Tracked    int
}

// PausedRemovals is the removals of one vault that wait for an answer, kept in the store of the plugin.
type PausedRemovals struct {
	Direction Direction `json:"direction"`
	Files     []Removal `json:"files"`
	Tracked   int       `json:"tracked"`
	CreatedAt time.Time `json:"createdAt"`
}

// Choice is the answer of the person to paused removals.
type Choice string

const (
	// ChoiceRemove lets the removals go on.
	ChoiceRemove Choice = "remove"
	// ChoiceKeep drops the removals and keeps the files.
	ChoiceKeep Choice = "keep"
)

// Approval is the answer of the person to a pause, kept until the next pass over the vault takes it.
type Approval struct {
	Direction Direction `json:"direction"`
	Choice    Choice    `json:"choice"`
}

// Upload is one reservation on the server: its blob, the size of its parts and the parts the server holds. URL is
// set for a body sent in one request.
type Upload struct {
	Vault     []byte
	URL       string
	Blob      []byte
	PartSize  int64
	PartCount int
	Done      []int
}

// UploadEntry is an upload of a big file that is not finished, kept between runs. Its key is sealed and its hash
// state is the state of the hash of ciphertext only.
type UploadEntry struct {
	FileID          []byte `json:"fileId"`
	Path            string `json:"path"`
	PlainSize       int64  `json:"plainSize"`
	PlainHash       string `json:"plainHash"`
	MTime           int64  `json:"mtime"`
	ParentRev       []byte `json:"parentRev"`
	BlobID          []byte `json:"blobId"`
	PartSize        int64  `json:"partSize"`
	PartCount       int    `json:"partCount"`
	StoredSize      int64  `json:"storedSize"`
	Salt            []byte `json:"salt"`
	SealedKey       []byte `json:"sealedKey"`
	PartsDone       int    `json:"partsDone"`
	StoredHashState []byte `json:"storedHashState"`
}

// DownloadEntry is a download that is not finished, kept between runs. Its hash state is the state of the hash of
// the stored bytes, never of the plain ones.
type DownloadEntry struct {
	FileID          []byte `json:"fileId"`
	Path            string `json:"path"`
	BlobID          []byte `json:"blobId"`
	Header          []byte `json:"header"`
	NextChunk       uint64 `json:"nextChunk"`
	StoredHashState []byte `json:"storedHashState"`
	StagedName      string `json:"stagedName"`
	MetaHash        string `json:"metaHash"`
	HeadRev         []byte `json:"headRev"`
}

var (
	// ErrUploadGone means the server has no such reservation any more.
	ErrUploadGone = errors.New("the upload does not exist on the sync server")
	// ErrIntegrity means the bytes the server sent are not the blob its revision names.
	ErrIntegrity = errors.New("a file from the sync server is damaged")
	// ErrBlobMissing means the server has no blob for a revision.
	ErrBlobMissing = errors.New("the server has no blob for a revision")
)

// FileRecord is what this device knows of a synced file: the file on the server, where it is, the hash both
// computers last agreed on, the revision that is its head and what the head's blob is, so that a move can be
// sent without reading the file. SealedKey is the revision key of the head, sealed with the vault key. Version is the
// signed version of the head: a revision of the file below it is never taken.
type FileRecord struct {
	FileID     []byte `json:"fileId"`
	Path       string `json:"path"`
	BaseHash   string `json:"baseHash"`
	HeadRev    []byte `json:"headRev"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"mtime"`
	Kind       string `json:"kind"`
	BlobID     []byte `json:"blobId"`
	BlobHash   []byte `json:"blobHash"`
	BlobSize   uint64 `json:"blobSize"`
	SealedKey  []byte `json:"revKey"`
	Version    uint64 `json:"version"`
}
