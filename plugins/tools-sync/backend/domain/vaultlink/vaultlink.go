// Package vaultlink is how a local vault is tied to a synced vault: the link kept in the vault, the entries of
// the catalog that name synced vaults and the keys that seal them.
package vaultlink

import (
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
	// RestoreEpochID is the restore epoch this device ran last, empty when the server announced none.
	RestoreEpochID []byte `json:"restoreEpochId,omitempty"`
	// Recovery is set while a recovery of the vault runs and holds what is left of it.
	Recovery *RecoveryProgress `json:"recovery,omitempty"`
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

// RecoveryProgress is what a recovery of one vault has still to examine: the ids of the files, 16 bytes each.
type RecoveryProgress struct {
	Pending [][]byte `json:"pending"`
}
