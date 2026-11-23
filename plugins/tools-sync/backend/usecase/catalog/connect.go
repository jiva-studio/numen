package catalog

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
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

// ConnectInput names the local vault, how it is connected and, for ModeExisting, the synced vault.
type ConnectInput struct {
	VaultID  domain.VaultID
	Mode     Mode
	RemoteID string
}

// ConnectVault ties a local vault to a synced vault.
type ConnectVault struct{ reader reader }

// NewConnectVault returns the use case over the dependencies.
func NewConnectVault(deps Deps) *ConnectVault { return &ConnectVault{reader: reader{deps: deps}} }

// Execute connects the vault. It refuses a copy of a connected vault with ErrCopyOfConnectedVault.
func (u *ConnectVault) Execute(ctx context.Context, in ConnectInput) error {
	if in.Mode != ModeNew && in.Mode != ModeExisting {
		return errUnknownMode
	}
	if _, err := u.reader.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return err
	}
	if err := u.checkNotReceived(ctx, in.VaultID); err != nil {
		return err
	}
	if err := u.checkNotConnected(ctx, in); err != nil {
		return err
	}
	name, err := u.getVaultName(ctx, in.VaultID)
	if err != nil {
		return err
	}
	if in.Mode == ModeNew {
		return u.connectNew(ctx, in.VaultID, name)
	}
	return u.connectExisting(ctx, in, name)
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

func (u *ConnectVault) getVaultName(ctx context.Context, id domain.VaultID) (string, error) {
	vaults, err := u.reader.deps.Host.GetVaults(ctx)
	if err != nil {
		return "", err
	}
	for _, vault := range vaults {
		if vault.ID == id {
			return vault.Name, nil
		}
	}
	return "", fmt.Errorf("vault %s: not on the list of the host", id)
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
	if isLinked {
		return errAlreadyConnected
	}
	return nil
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
	body := vaultlink.CatalogBody{LocalID: string(id), Name: name, Key: key}
	if err := u.reader.appendEntry(ctx, entryInput{vaultID: remoteID, isCreating: true, body: body}); err != nil {
		return err
	}
	return u.writeLink(ctx, id, remoteID, name, key)
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
	vaults, err := u.reader.fold(ctx, entries)
	if err != nil {
		return err
	}
	for _, one := range vaults {
		if string(one.vaultID) == string(remoteID) {
			return u.writeLink(ctx, in.VaultID, remoteID, name, one.body.Key)
		}
	}
	return ErrUnknownRemoteVault
}

func (u *ConnectVault) writeLink(ctx context.Context, id domain.VaultID, remoteID []byte, name string, key []byte) error {
	sealed, err := u.reader.deps.Crypto.SealData(ctx, vaultlink.GetStateKeyContext(remoteID), key)
	if err != nil {
		return fmt.Errorf("seal the vault key: %w", err)
	}
	link := vaultlink.Link{RemoteVaultID: hex.EncodeToString(remoteID), SealedKey: sealed, CatalogName: name, ServerSeq: 0, FeedCursor: host.FileChangeCursor{FeedID: "", Seq: 0}}
	return u.reader.deps.State.WriteLink(ctx, id, link)
}
