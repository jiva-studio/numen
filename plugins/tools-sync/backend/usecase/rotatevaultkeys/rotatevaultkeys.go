// Package rotatevaultkeys gives every connected vault a key of the epoch of the account keys.
package rotatevaultkeys

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	maxTries = 3
	keySize  = 32
)

// ErrCatalogBusy means other devices appended to the catalog first, every time.
var ErrCatalogBusy = errors.New("the catalog changed on another device again and again")

// Deps are what the use case needs.
type Deps struct {
	Catalog  ports.Catalog
	Chain    ports.Chain
	Crypto   ports.Crypto
	Identity ports.Identity
	Host     ports.Host
	State    ports.Links
	Random   io.Reader
}

// RotateVaultKeys appends the catalog entries that carry a new vault key for the epoch of the account.
type RotateVaultKeys struct{ deps Deps }

// NewRotateVaultKeys returns the use case over the dependencies.
func NewRotateVaultKeys(deps Deps) *RotateVaultKeys { return &RotateVaultKeys{deps: deps} }

// Execute returns the number of vaults it gave a new key. A vault whose newest key has the epoch of the account keys, or a later
// one, is left alone, so the device that loses the race of the catalog finds the entry of the winner and makes no second key.
func (u *RotateVaultKeys) Execute(ctx context.Context) (int, error) {
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil || own.KeyEpoch == 0 || own.IsRevoked {
		return 0, err
	}
	vaults, err := u.deps.Host.GetVaults(ctx)
	if err != nil {
		return 0, err
	}
	rotated := 0
	for _, vault := range vaults {
		isRotated, err := u.rotate(ctx, vault.ID, own)
		if err != nil {
			return rotated, err
		}
		if isRotated {
			rotated++
		}
	}
	return rotated, nil
}

func (u *RotateVaultKeys) rotate(ctx context.Context, id domain.VaultID, own session.Identity) (bool, error) {
	link, err := u.deps.State.ReadLink(ctx, id)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return false, fmt.Errorf("read the remote id of the link: %w", err)
	}
	target := connected{id: id, remote: remote, link: link}
	for range maxTries {
		entries, err := u.deps.Catalog.GetCatalog(ctx, 0)
		if err != nil {
			return false, err
		}
		last := getLastEntry(entries, remote)
		if last == nil {
			return false, nil
		}
		body, err := u.openBody(ctx, last)
		if err != nil {
			return false, err
		}
		if err := u.store(ctx, &target, body.GetKeys()); err != nil {
			return false, err
		}
		if vaultlink.GetNewestEpoch(body.GetKeys()) >= own.KeyEpoch {
			return false, nil
		}
		keys, err := u.append(ctx, entries, last, body, own)
		if err != nil {
			return false, err
		}
		if keys != nil {
			return true, u.store(ctx, &target, keys)
		}
	}
	return false, ErrCatalogBusy
}

// connected is a connected vault with the link it keeps.
type connected struct {
	id     domain.VaultID
	remote []byte
	link   vaultlink.Link
}

func (u *RotateVaultKeys) store(ctx context.Context, target *connected, keys []vaultlink.VaultKey) error {
	known := uint32(0)
	if len(target.link.LaterKeys) > 0 {
		known = target.link.LaterKeys[len(target.link.LaterKeys)-1].Epoch
	}
	if vaultlink.GetNewestEpoch(keys) <= known {
		return nil
	}
	first, later, err := vaultlink.SealKeys(ctx, u.deps.Crypto, target.remote, keys)
	if err != nil {
		return err
	}
	target.link.SealedKey, target.link.LaterKeys = first, later
	return u.deps.State.WriteLink(ctx, target.id, target.link)
}

func getLastEntry(entries []*syncv1.CatalogEntry, vaultID []byte) *syncv1.CatalogEntry {
	for i := len(entries) - 1; i >= 0; i-- {
		if bytes.Equal(entries[i].GetVaultId(), vaultID) {
			return entries[i]
		}
	}
	return nil
}

func (u *RotateVaultKeys) openBody(ctx context.Context, entry *syncv1.CatalogEntry) (vaultlink.CatalogBody, error) {
	plain, err := u.deps.Crypto.OpenData(ctx, vaultlink.GetCatalogContext(entry.GetVaultId(), entry.GetVersion()), entry.GetEncBody())
	if err != nil {
		return vaultlink.CatalogBody{}, fmt.Errorf("open the body of catalog entry %d: %w", entry.GetVersion(), err)
	}
	return vaultlink.ParseCatalogBody(plain)
}

func (u *RotateVaultKeys) append(ctx context.Context, entries []*syncv1.CatalogEntry, last *syncv1.CatalogEntry, body vaultlink.CatalogBody, own session.Identity) ([]vaultlink.VaultKey, error) {
	fresh := make([]byte, keySize)
	if _, err := io.ReadFull(u.deps.Random, fresh); err != nil {
		return nil, fmt.Errorf("make a vault key: %w", err)
	}
	next := vaultlink.CatalogBody{
		LocalID: body.LocalID, Name: body.Name, Key: body.Key,
		Keys: append(append([]vaultlink.VaultKey(nil), body.GetKeys()...), vaultlink.VaultKey{Epoch: own.KeyEpoch, Key: fresh}),
	}
	entry, err := u.newEntry(ctx, entries, last.GetVaultId(), next, own)
	if err != nil {
		return nil, err
	}
	response, err := u.deps.Catalog.AppendCatalogEntry(ctx, entry)
	if err != nil {
		return nil, err
	}
	if !response.GetCommitted() {
		return nil, nil
	}
	return next.Keys, nil
}

func (u *RotateVaultKeys) newEntry(ctx context.Context, entries []*syncv1.CatalogEntry, vaultID []byte, body vaultlink.CatalogBody, own session.Identity) (*syncv1.CatalogEntry, error) {
	entry := &syncv1.CatalogEntry{
		Version: uint64(len(entries)) + 1, VaultId: vaultID, CreatesVault: false, AuthorDeviceSignKey: own.DeviceSignKey,
	}
	if len(entries) > 0 {
		previous := syncprotocol.GetCatalogEntryID(entries[len(entries)-1])
		entry.PrevHash = previous[:]
	}
	plain, err := body.Encode()
	if err != nil {
		return nil, err
	}
	if entry.EncBody, err = u.deps.Crypto.SealData(ctx, vaultlink.GetCatalogContext(vaultID, entry.GetVersion()), plain); err != nil {
		return nil, fmt.Errorf("seal the body of a catalog entry: %w", err)
	}
	if entry.Signature, err = u.deps.Identity.SignAsDevice(ctx, syncprotocol.GetCatalogEntryMessage(entry)); err != nil {
		return nil, err
	}
	return entry, nil
}

const roundInterval = 10 * time.Second

// Loop gives the vaults the keys of a new epoch as soon as the device holds the account keys of it, and again at each round.
type Loop struct {
	rotate *RotateVaultKeys
	clock  ports.Clock
}

// NewLoop returns the loop over the dependencies and the clock it waits on.
func NewLoop(deps Deps, clock ports.Clock) *Loop {
	return &Loop{rotate: NewRotateVaultKeys(deps), clock: clock}
}

// Execute runs until ctx ends and returns its error. A failed round is tried again at the next one.
func (u *Loop) Execute(ctx context.Context) error {
	for {
		_, _ = u.rotate.Execute(ctx)
		if err := u.clock.Wait(ctx, roundInterval); err != nil {
			return err
		}
	}
}
