// Package vaultlink is how a local vault is tied to a synced vault: the link kept in the vault, the entries of
// the catalog that name synced vaults and the keys that seal them.
package vaultlink

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const (
	catalogContext  = "numen-sync/catalog/v1"
	stateKeyContext = "numen-sync/state-key/v1"
)

// ErrNotConnected means a vault has no link.
var ErrNotConnected = errors.New("the vault is not connected to a synced vault")

var errNoKeys = errors.New("the vault has no key")

// FeedCursor is the place in the file change feed of the host that the vault was read up to.
type FeedCursor struct {
	FeedID string `json:"feedId"`
	Seq    int64  `json:"seq"`
}

// Link is link.json of a connected vault.
type Link struct {
	RemoteVaultID  string            `json:"remoteVaultId"`
	SealedKey      []byte            `json:"sealedKey"`
	LaterKeys      []SealedKey       `json:"laterKeys,omitempty"`
	CatalogName    string            `json:"catalogName"`
	ServerSeq      uint64            `json:"serverSeq"`
	FeedCursor     FeedCursor        `json:"feedCursor"`
	RestoreEpochID []byte            `json:"restoreEpochId,omitempty"`
	Recovery       *RecoveryProgress `json:"recovery,omitempty"`
	Removed        []RemovedFile     `json:"removed,omitempty"`
	Retired        []RetiredFile     `json:"retired,omitempty"`
	Kind           string            `json:"kind,omitempty"`
	Received       *Received         `json:"received,omitempty"`
}

// RemoteVault is a synced vault of the account, with its last name.
type RemoteVault struct {
	RemoteID    string
	Name        string
	IsConnected bool
	SizeBytes   uint64
}

// CatalogBody is what an entry of the catalog seals: the vault on the computer that wrote it, the name and the
// keys of the synced vault by epoch. A body with Removed set is the newest entry of a vault whose cloud copy was
// deleted: the vault is unlinked and not listed any more.
type CatalogBody struct {
	LocalID string     `json:"localId"`
	Name    string     `json:"name"`
	Key     []byte     `json:"key"`
	Keys    []VaultKey `json:"keys"`
	Removed bool       `json:"removed,omitempty"`
}

// VaultKey is a key of a synced vault with the epoch of the account keys it was made under.
type VaultKey struct {
	Epoch uint32 `json:"epoch"`
	Key   []byte `json:"key"`
}

// GetKeys returns every key of the vault by epoch: the keys the body lists, or the one key of epoch 0 of a body that lists none.
func (b CatalogBody) GetKeys() []VaultKey {
	if len(b.Keys) > 0 {
		return b.Keys
	}
	return []VaultKey{{Epoch: 0, Key: b.Key}}
}

// Encode returns the JSON of the body.
func (b CatalogBody) Encode() ([]byte, error) {
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("encode the body of a catalog entry: %w", err)
	}
	return raw, nil
}

// ParseCatalogBody reads the JSON of a body.
func ParseCatalogBody(raw []byte) (CatalogBody, error) {
	var body CatalogBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return CatalogBody{}, fmt.Errorf("read the body of a catalog entry: %w", err)
	}
	return body, nil
}

// GetCatalogContext returns the context the body of an entry of the version is sealed under.
func GetCatalogContext(vaultID []byte, version uint64) []byte {
	sealed := append([]byte(catalogContext), vaultID...)
	return binary.BigEndian.AppendUint64(sealed, version)
}

// GetStateKeyContext returns the context the vault key in link.json is sealed under.
func GetStateKeyContext(remoteID []byte) []byte {
	return append([]byte(stateKeyContext), remoteID...)
}

// IsRemoteID reports whether text is the hex form of the id of a synced vault.
func IsRemoteID(text string) bool {
	_, err := hex.DecodeString(text)
	return err == nil
}

// RemovedFile is a file this device removed on the server: its id, its path, the revision of the removal and its
// signed version. A restored server that holds the file again gets the removal sent again.
type RemovedFile struct {
	FileID  []byte `json:"fileId"`
	Path    string `json:"path"`
	HeadRev []byte `json:"headRev"`
	Version uint64 `json:"version"`
}

const maxRemoved = 1000

// AddRemoved records a removal in place of an older one of the same file and drops the oldest beyond maxRemoved.
func (l *Link) AddRemoved(one RemovedFile) {
	l.Removed = slices.DeleteFunc(l.Removed, func(old RemovedFile) bool { return bytes.Equal(old.FileID, one.FileID) })
	l.Removed = append(l.Removed, one)
	if len(l.Removed) > maxRemoved {
		l.Removed = l.Removed[len(l.Removed)-maxRemoved:]
	}
}

// RecoveryProgress is what a recovery of one vault has still to examine: the ids of the files, 16 bytes each.
type RecoveryProgress struct {
	Pending [][]byte `json:"pending"`
}

// SealedKey is a vault key of an epoch after the first, sealed by the core.
type SealedKey struct {
	Epoch  uint32 `json:"epoch"`
	Sealed []byte `json:"sealed"`
}

// GetEpochKeyContext returns the context the vault key of an epoch after the first is sealed under in link.json.
func GetEpochKeyContext(remoteID []byte, epoch uint32) []byte {
	return binary.BigEndian.AppendUint32(GetStateKeyContext(remoteID), epoch)
}

// Sealer seals and opens data under the keys of the account.
type Sealer interface {
	SealData(ctx context.Context, dataContext, plain []byte) ([]byte, error)
	OpenData(ctx context.Context, dataContext, sealed []byte) ([]byte, error)
}

// SealKeys returns the key of the first epoch sealed for link.json, and the keys after it each sealed under its epoch. The first key is epoch 0.
func SealKeys(ctx context.Context, sealer Sealer, remoteID []byte, keys []VaultKey) ([]byte, []SealedKey, error) {
	if len(keys) == 0 {
		return nil, nil, errNoKeys
	}
	first, err := sealer.SealData(ctx, GetStateKeyContext(remoteID), keys[0].Key)
	if err != nil {
		return nil, nil, fmt.Errorf("seal the vault key: %w", err)
	}
	var later []SealedKey
	for _, key := range keys[1:] {
		sealed, err := sealer.SealData(ctx, GetEpochKeyContext(remoteID, key.Epoch), key.Key)
		if err != nil {
			return nil, nil, fmt.Errorf("seal the vault key of epoch %d: %w", key.Epoch, err)
		}
		later = append(later, SealedKey{Epoch: key.Epoch, Sealed: sealed})
	}
	return first, later, nil
}

// OpenKeys returns the keys of the link by epoch. The first key is epoch 0.
func OpenKeys(ctx context.Context, sealer Sealer, link Link) ([]VaultKey, error) {
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return nil, fmt.Errorf("read the remote id of the link: %w", err)
	}
	first, err := sealer.OpenData(ctx, GetStateKeyContext(remote), link.SealedKey)
	if err != nil {
		return nil, fmt.Errorf("open the vault key: %w", err)
	}
	keys := []VaultKey{{Epoch: 0, Key: first}}
	for _, one := range link.LaterKeys {
		key, err := sealer.OpenData(ctx, GetEpochKeyContext(remote, one.Epoch), one.Sealed)
		if err != nil {
			return nil, fmt.Errorf("open the vault key of epoch %d: %w", one.Epoch, err)
		}
		keys = append(keys, VaultKey{Epoch: one.Epoch, Key: key})
	}
	return keys, nil
}

// GetNewestEpoch returns the highest epoch of the keys, 0 for none.
func GetNewestEpoch(keys []VaultKey) uint32 {
	var newest uint32
	for _, key := range keys {
		newest = max(newest, key.Epoch)
	}
	return newest
}

const (
	// KindOwn is a link of a vault the person syncs with its own account.
	KindOwn = "own"
	// KindReceived is a link of a vault that is a share of another person.
	KindReceived = "received"
)

// Received is what a received vault knows of its share: the share, the pinned root of the owner, the sealed keys and
// the last entry of the share log it verified.
type Received struct {
	ShareRootID   string                  `json:"shareRootId"`
	OwnerRoot     []byte                  `json:"ownerRoot"`
	OwnerChain    []byte                  `json:"ownerChain"`
	SealedKeys    []byte                  `json:"sealedKeys"`
	ServerVersion uint64                  `json:"serverVersion"`
	HeadID        []byte                  `json:"headId"`
	VaultID       []byte                  `json:"vaultId"`
	Epoch         uint32                  `json:"epoch"`
	Name          string                  `json:"name"`
	IsEnded       bool                    `json:"isEnded"`
	Seen          map[string]SeenRevision `json:"seen,omitempty"`
}

// SeenRevision is the newest revision a received share served for a file: its id and its signed version.
type SeenRevision struct {
	RevID   []byte `json:"revId"`
	Version uint64 `json:"version"`
}

// IsRegressed reports whether the revision is older than the newest one the share served for the file: a lower
// version, or the same version with another id.
func (r *Received) IsRegressed(fileID, revID []byte, version uint64) bool {
	seen, isSeen := r.Seen[hex.EncodeToString(fileID)]
	if !isSeen {
		return false
	}
	return version < seen.Version || (version == seen.Version && !bytes.Equal(seen.RevID, revID))
}

// RecordRevision remembers that the share served the revision as the newest of the file.
func (r *Received) RecordRevision(fileID, revID []byte, version uint64) {
	if r.Seen == nil {
		r.Seen = map[string]SeenRevision{}
	}
	r.Seen[hex.EncodeToString(fileID)] = SeenRevision{RevID: revID, Version: version}
}

// RetiredFile is a file another computer removed and this device applied: its id and the signed version of the removal.
type RetiredFile struct {
	FileID  []byte `json:"fileId"`
	Version uint64 `json:"version"`
}

const maxRetired = 10000

// AddRetired records the removal of a file in place of an older one and drops the oldest beyond maxRetired.
func (l *Link) AddRetired(one RetiredFile) {
	l.Retired = slices.DeleteFunc(l.Retired, func(old RetiredFile) bool { return bytes.Equal(old.FileID, one.FileID) })
	l.Retired = append(l.Retired, one)
	if len(l.Retired) > maxRetired {
		l.Retired = l.Retired[len(l.Retired)-maxRetired:]
	}
}

// GetFloor returns the highest signed version at which this device saw the file removed, or 0 for a file it never saw
// removed. A revision of the file at or below it is older than the removal.
func (l *Link) GetFloor(fileID []byte) uint64 {
	var floor uint64
	for _, one := range l.Removed {
		if bytes.Equal(one.FileID, fileID) {
			floor = max(floor, one.Version)
		}
	}
	for _, one := range l.Retired {
		if bytes.Equal(one.FileID, fileID) {
			floor = max(floor, one.Version)
		}
	}
	return floor
}

// ErrNoVaults means the list of vaults of this installation is empty, or no vault of it has a folder to keep a copy in.
var ErrNoVaults = errors.New("the installation has no vault to keep the copy in")

// FolderMissingError means the folder of a vault is gone from the disk.
type FolderMissingError struct {
	Name string
	Path string
}

func (e FolderMissingError) Error() string {
	return fmt.Sprintf("the folder %s of the vault %s is missing", e.Path, e.Name)
}

// MergeRetired records each removal of the list unless the link already holds a removal of the same file at a version
// that is not lower.
func (l *Link) MergeRetired(others []RetiredFile) {
	for _, one := range others {
		at := slices.IndexFunc(l.Retired, func(old RetiredFile) bool { return bytes.Equal(old.FileID, one.FileID) })
		if at < 0 || l.Retired[at].Version < one.Version {
			l.AddRetired(one)
		}
	}
}
