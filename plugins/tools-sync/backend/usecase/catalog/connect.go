package catalog

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Mode is how a local vault is connected.
type Mode int

const (
	// ModeNew makes a synced vault of the local vault.
	ModeNew Mode = iota + 1
	// ModeExisting connects the local vault to a synced vault of the account.
	ModeExisting
)

const keySize = 32

// keyOtherIdentity is the prefix of the item a pass raises for a vault synced with another identity.
const keyOtherIdentity = "other-identity"

// ConnectInput names the local vault, how it is connected and, for ModeExisting, the synced vault.
type ConnectInput struct {
	VaultID  domain.VaultID
	Mode     Mode
	RemoteID string
}

// ConnectVault ties a local vault to a synced vault.
type ConnectVault struct {
	reader reader
	mu     sync.Mutex
}

// NewConnectVault returns the use case over the dependencies.
func NewConnectVault(deps Deps) *ConnectVault {
	return &ConnectVault{reader: reader{deps: deps}, mu: sync.Mutex{}}
}

// Execute connects the vault while it holds the lock of the vault, so no pass writes the link meanwhile. It waits for a pass over the vault that runs.
func (u *ConnectVault) Execute(ctx context.Context, in ConnectInput) error {
	unlock, err := u.reader.deps.Locker.LockVault(ctx, in.VaultID)
	if err != nil {
		return err
	}
	defer unlock()
	return u.ExecuteHeld(ctx, in)
}

// ExecuteHeld connects the vault for a caller that holds the lock of the vault; one call at a time runs the check and the write. It refuses a copy of a connected vault with ErrCopyOfConnectedVault.
func (u *ConnectVault) ExecuteHeld(ctx context.Context, in ConnectInput) error {
	if in.Mode != ModeNew && in.Mode != ModeExisting {
		return errUnknownMode
	}
	if _, err := u.reader.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if err := u.checkNotReceived(ctx, in.VaultID); err != nil {
		return err
	}
	if err := u.checkNotConnected(ctx, in); err != nil {
		return err
	}
	vault, err := u.getVault(ctx, in.VaultID)
	if err != nil {
		return err
	}
	if in.Mode == ModeNew {
		err = u.connectNew(ctx, in.VaultID, vault.Name)
	} else {
		err = u.connectExisting(ctx, in, vault.Name)
	}
	if errors.Is(err, host.ErrNotExist) || errors.Is(err, vaultlink.ErrNoVaults) {
		return fmt.Errorf("connect: %w", vaultlink.FolderMissingError{Name: vault.Name, Path: vault.Path})
	}
	if err != nil {
		return err
	}
	return u.reader.deps.Health.Clear(ctx, healthkey.Create(keyOtherIdentity, string(in.VaultID)))
}

func (u *ConnectVault) checkNotReceived(ctx context.Context, id domain.VaultID) error {
	link, err := u.reader.deps.State.ReadLink(ctx, id)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return nil
	}
	if err != nil {
		return err
	}
	if link.Kind == vaultlink.KindReceived {
		return share.ErrReceivedNotSynced
	}
	return nil
}

func (u *ConnectVault) getVault(ctx context.Context, id domain.VaultID) (domain.Vault, error) {
	vaults, err := u.reader.deps.Host.GetVaults(ctx)
	if err != nil {
		return domain.Vault{}, err
	}
	for _, vault := range vaults {
		if vault.ID == id {
			return vault, nil
		}
	}
	return domain.Vault{}, fmt.Errorf("vault %s: not on the list of the host", id)
}

func (u *ConnectVault) checkNotConnected(ctx context.Context, in ConnectInput) error {
	links, err := u.reader.readLinks(ctx)
	if err != nil {
		return err
	}
	own, isLinked := links[in.VaultID]
	for id, link := range links {
		if id != in.VaultID && (link.RemoteVaultID == in.RemoteID || (isLinked && link.RemoteVaultID == own.RemoteVaultID)) {
			return ErrCopyOfConnectedVault
		}
	}
	if !isLinked {
		return nil
	}
	isForeign, err := ports.IsKeyForeign(ctx, u.reader.deps.Crypto, own)
	if err != nil {
		return err
	}
	if !isForeign {
		return errAlreadyConnected
	}
	return u.reader.deps.Forget.ForgetVault(ctx, in.VaultID)
}

func (u *ConnectVault) connectNew(ctx context.Context, id domain.VaultID, name string) error {
	key, err := note.NewID(u.reader.deps.Random, keySize)
	if err != nil {
		return err
	}
	remoteID, err := note.NewID(u.reader.deps.Random, 16)
	if err != nil {
		return err
	}
	body := vaultlink.CatalogBody{LocalID: string(id), Name: name, Key: key, Keys: nil, Removed: false}
	if err := u.reader.appendEntry(ctx, entryInput{vaultID: remoteID, isCreating: true, body: body}); err != nil {
		return err
	}
	return u.writeLink(ctx, id, remoteID, name, body.GetKeys())
}

func (u *ConnectVault) connectExisting(ctx context.Context, in ConnectInput, name string) error {
	remoteID, err := hex.DecodeString(in.RemoteID)
	if err != nil {
		return ErrUnknownRemoteVault
	}
	entries, err := u.reader.read(ctx)
	if err != nil {
		return err
	}
	vaults, err := u.reader.mergeEntries(ctx, entries)
	if err != nil {
		return err
	}
	for _, one := range vaults {
		if string(one.vaultID) == string(remoteID) && !one.body.Removed {
			return u.writeLink(ctx, in.VaultID, remoteID, name, one.body.GetKeys())
		}
	}
	return ErrUnknownRemoteVault
}

func (u *ConnectVault) writeLink(ctx context.Context, id domain.VaultID, remoteID []byte, name string, keys []vaultlink.VaultKey) error {
	first, later, err := vaultlink.SealKeys(ctx, u.reader.deps.Crypto, remoteID, keys)
	if err != nil {
		return err
	}
	link := vaultlink.Link{
		RemoteVaultID: hex.EncodeToString(remoteID), SealedKey: first, LaterKeys: later, CatalogName: name, ServerSeq: 0, HighSeq: 0, RestoreEpochID: nil,
		FeedCursor: vaultlink.FeedCursor{FeedID: "", Seq: 0}, Recovery: nil, Removed: nil, Retired: nil, Kind: "", Received: nil,
	}
	return u.reader.deps.State.WriteLink(ctx, id, link)
}
