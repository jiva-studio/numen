package share

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const keySize = 32

// Phase is how far the owner got with a share.
type Phase string

const (
	// PhaseCreated means the share root exists and the files of the folder are not all rewrapped yet.
	PhaseCreated Phase = "created"
	// PhaseRewrapped means every file of the folder is a share revision; no reader was added yet.
	PhaseRewrapped Phase = "rewrapped"
	// PhaseActive means a reader was added.
	PhaseActive Phase = "active"
)

// The states of a sharing as the owner lists it.
const (
	// StateNotShared is a folder with no share.
	StateNotShared = "notShared"
	// StateCreating is a share whose files are not rewrapped yet.
	StateCreating = "creating"
	// StateActive is a share that takes readers.
	StateActive = "active"
	// StateRemoved is a reader the owner removed.
	StateRemoved = "removed"
)

// Progress is a row of shares.json: where this device stands with a share it made.
type Progress struct {
	ShareRootID string `json:"shareRootId"`
	Phase       Phase  `json:"phase"`
}

// KeyEntry is the key of one epoch.
type KeyEntry struct {
	Epoch uint32 `json:"epoch"`
	Key   []byte `json:"key"`
}

// Keys are the keys of the epochs of a share.
type Keys []KeyEntry

func (k Keys) Get(epoch uint32) ([32]byte, bool) {
	for _, one := range k {
		if one.Epoch == epoch && len(one.Key) == keySize {
			return [32]byte(one.Key), true
		}
	}
	return [32]byte{}, false
}

// Add returns the keys with the key of the epoch set.
func (k Keys) Add(epoch uint32, key [32]byte) Keys {
	next := make(Keys, 0, len(k)+1)
	for _, one := range k {
		if one.Epoch != epoch {
			next = append(next, one)
		}
	}
	return append(next, KeyEntry{Epoch: epoch, Key: key[:]})
}

// Encode returns the JSON of the keys.
func (k Keys) Encode() ([]byte, error) {
	raw, err := json.Marshal([]KeyEntry(k))
	if err != nil {
		return nil, fmt.Errorf("encode the share keys: %w", err)
	}
	return raw, nil
}

// ParseKeys reads the JSON of keys.
func ParseKeys(raw []byte) (Keys, error) {
	var keys []KeyEntry
	if err := json.Unmarshal(raw, &keys); err != nil {
		return nil, fmt.Errorf("read the share keys: %w", err)
	}
	return Keys(keys), nil
}

// Reader is a person the owner shares a folder with, as the owner body lists it.
type Reader struct {
	Root        []byte    `json:"root"`
	Chain       []byte    `json:"chain"`
	Label       string    `json:"label"`
	Fingerprint string    `json:"fingerprint"`
	AddedAt     time.Time `json:"addedAt"`
	RemovedAt   time.Time `json:"removedAt,omitzero"`
}

// IsActive reports whether the reader reads the folder now.
func (r Reader) IsActive() bool { return r.RemovedAt.IsZero() }

// GetState returns the state of the reader as the owner lists it.
func (r Reader) GetState() string {
	if r.IsActive() {
		return StateActive
	}
	return StateRemoved
}

// OwnerBody is what the owner seals into every log entry for its own devices: the folder, the keys of every epoch
// and the readers.
type OwnerBody struct {
	Folder  string   `json:"folder"`
	Keys    Keys     `json:"keys"`
	Readers []Reader `json:"readers"`
}

// Encode returns the JSON of the body.
func (b OwnerBody) Encode() ([]byte, error) {
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("encode the owner body: %w", err)
	}
	return raw, nil
}

// ParseOwnerBody reads the JSON of an owner body.
func ParseOwnerBody(raw []byte) (OwnerBody, error) {
	var body OwnerBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return OwnerBody{}, fmt.Errorf("read the owner body: %w", err)
	}
	return body, nil
}

// Root is a share as the owner knows it: the last entry of its log, opened.
type Root struct {
	ID       []byte
	VaultID  []byte
	Folder   string
	Epoch    uint32
	Keys     Keys
	Readers  []Reader
	Version  uint64
	HeadID   [32]byte
	IsClosed bool
	Phase    Phase
}

// GetKey returns the key of the current epoch.
func (r Root) GetKey() ([32]byte, bool) { return r.Keys.Get(r.Epoch) }

// FindReaderByFingerprint returns the active reader with the account fingerprint.
func (r Root) FindReaderByFingerprint(fingerprint string) (Reader, bool) {
	for _, one := range r.Readers {
		if one.IsActive() && one.Fingerprint == fingerprint {
			return one, true
		}
	}
	return Reader{}, false
}

func (r Root) GetState() string {
	if r.Phase == PhaseCreated {
		return StateCreating
	}
	return StateActive
}

func FindRoot(roots []Root, folder string) (Root, bool) {
	for _, one := range roots {
		if !one.IsClosed && isSameFolder(one.Folder, folder) {
			return one, true
		}
	}
	return Root{}, false
}

func isSameFolder(first, second string) bool {
	one, other := getSegments(first), getSegments(second)
	return len(one) > 0 && len(one) == len(other) && hasPrefix(one, other)
}

// FindRootOf returns the open share that holds the file.
func FindRootOf(roots []Root, filePath string) (Root, bool) {
	for _, one := range roots {
		if _, isUnder := GetRelativePath(one.Folder, filePath); isUnder && !one.IsClosed {
			return one, true
		}
	}
	return Root{}, false
}

func FindRootByID(roots []Root, id []byte) (Root, bool) {
	for _, one := range roots {
		if bytes.Equal(one.ID, id) {
			return one, true
		}
	}
	return Root{}, false
}

// ListFolders returns the folders of the open shares.
func ListFolders(roots []Root) []string {
	var folders []string
	for _, one := range roots {
		if !one.IsClosed {
			folders = append(folders, one.Folder)
		}
	}
	return folders
}

// CheckReader refuses a reader the folder has already, and a label that another person used before unless the owner
// confirmed the change. root is the root key of the person the code names.
func CheckReader(readers []Reader, root []byte, label string, isConfirmed bool) error {
	for _, one := range readers {
		if bytes.Equal(one.Root, root) && one.IsActive() {
			return ErrAlreadyReader
		}
	}
	if isConfirmed {
		return nil
	}
	for _, one := range readers {
		if strings.EqualFold(one.Label, label) && !bytes.Equal(one.Root, root) {
			return ErrChangedIdentity
		}
	}
	return nil
}

// SetReader returns the readers with the reader added, or replaced when the root key is listed already.
func SetReader(readers []Reader, reader Reader) []Reader {
	next := make([]Reader, 0, len(readers)+1)
	for _, one := range readers {
		if !bytes.Equal(one.Root, reader.Root) {
			next = append(next, one)
		}
	}
	return append(next, reader)
}

func MarkReaderRemoved(readers []Reader, root []byte, at time.Time) []Reader {
	next := make([]Reader, len(readers))
	for i, one := range readers {
		next[i] = one
		if bytes.Equal(one.Root, root) {
			next[i].RemovedAt = at
		}
	}
	return next
}

// MarkAllRemoved returns the readers with every active reader marked removed at the time.
func MarkAllRemoved(readers []Reader, at time.Time) []Reader {
	next := make([]Reader, len(readers))
	for i, one := range readers {
		next[i] = one
		if one.IsActive() {
			next[i].RemovedAt = at
		}
	}
	return next
}
