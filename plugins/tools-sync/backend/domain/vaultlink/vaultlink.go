// Package vaultlink is how a local vault is tied to a synced vault: the link kept in the vault, the entries of
// the catalog that name synced vaults and the keys that seal them.
package vaultlink

import (
	"context"
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

var errNoKeys = errors.New("the vault has no key")

// Link is link.json of a connected vault.
type Link struct {
	RemoteVaultID string                `json:"remoteVaultId"`
	SealedKey     []byte                `json:"sealedKey"`
	LaterKeys     []SealedKey           `json:"laterKeys,omitempty"`
	CatalogName   string                `json:"catalogName"`
	ServerSeq     uint64                `json:"serverSeq"`
	FeedCursor    host.FileChangeCursor `json:"feedCursor"`
}

// RemoteVault is a synced vault of the account, with its last name.
type RemoteVault struct {
	RemoteID    string
	Name        string
	IsConnected bool
}

// CatalogBody is what an entry of the catalog seals: the vault on the computer that wrote it, the name and the
// keys of the synced vault by epoch.
type CatalogBody struct {
	LocalID string     `json:"localId"`
	Name    string     `json:"name"`
	Key     []byte     `json:"key"`
	Keys    []VaultKey `json:"keys"`
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
