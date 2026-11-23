// Package vaultlink is how a local vault is tied to a synced vault: the link kept in the vault, the entries of
// the catalog that name synced vaults and the keys that seal them.
package vaultlink

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

const (
	catalogContext  = "numen-sync/catalog/v1"
	stateKeyContext = "numen-sync/state-key/v1"
)

// ErrNotConnected means a vault has no link.
var ErrNotConnected = errors.New("the vault is not connected to a synced vault")

// Link is link.json of a connected vault.
type Link struct {
	RemoteVaultID string                `json:"remoteVaultId"`
	SealedKey     []byte                `json:"sealedKey"`
	CatalogName   string                `json:"catalogName"`
	ServerSeq     uint64                `json:"serverSeq"`
	FeedCursor    host.FileChangeCursor `json:"feedCursor"`
	Kind          string                `json:"kind,omitempty"`
	Received      *Received             `json:"received,omitempty"`
}

// RemoteVault is a synced vault of the account, with its last name.
type RemoteVault struct {
	RemoteID    string
	Name        string
	IsConnected bool
}

// CatalogBody is what an entry of the catalog seals: the vault on the computer that wrote it, the name and the
// key of the synced vault.
type CatalogBody struct {
	LocalID string `json:"localId"`
	Name    string `json:"name"`
	Key     []byte `json:"key"`
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

const (
	// KindOwn is a link of a vault the person syncs with its own account.
	KindOwn = "own"
	// KindReceived is a link of a vault that is a share of another person.
	KindReceived = "received"
)

// Received is what a received vault knows of its share: the share, the pinned root of the owner, the sealed keys and
// the last entry of the share log it verified.
type Received struct {
	ShareRootID   string              `json:"shareRootId"`
	OwnerRoot     []byte              `json:"ownerRoot"`
	OwnerChain    []byte              `json:"ownerChain"`
	SealedKeys    []byte              `json:"sealedKeys"`
	ServerVersion uint64              `json:"serverVersion"`
	HeadID        []byte              `json:"headId"`
	VaultID       []byte              `json:"vaultId"`
	Epoch         uint32              `json:"epoch"`
	Name          string              `json:"name"`
	IsEnded       bool                `json:"isEnded"`
	Seen          map[string][][]byte `json:"seen,omitempty"`
}

// maxSeen is how many revisions of a file a received share remembers.
const maxSeen = 8

// IsRegressed reports whether the revision is one the share served for the file before its newest one.
func (r *Received) IsRegressed(fileID, revID []byte) bool {
	seen := r.Seen[hex.EncodeToString(fileID)]
	if len(seen) == 0 {
		return false
	}
	for _, one := range seen[:len(seen)-1] {
		if bytes.Equal(one, revID) {
			return true
		}
	}
	return false
}

// RecordRevision remembers that the share served the revision as the newest of the file.
func (r *Received) RecordRevision(fileID, revID []byte) {
	if r.Seen == nil {
		r.Seen = map[string][][]byte{}
	}
	key := hex.EncodeToString(fileID)
	seen := r.Seen[key]
	if len(seen) > 0 && bytes.Equal(seen[len(seen)-1], revID) {
		return
	}
	seen = append(seen, revID)
	if len(seen) > maxSeen {
		seen = seen[len(seen)-maxSeen:]
	}
	r.Seen[key] = seen
}
