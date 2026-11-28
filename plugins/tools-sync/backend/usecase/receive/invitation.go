package receive

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// openedInvitation is an invitation that is for this account, with the chain of the owner it names, verified.
type openedInvitation struct {
	invitation domainshare.Invitation
	ownerChain identity.Chain
	reader     identity.Chain
}

// followedLog is the share log of an invitation, verified, and the chain of the owner the server stores.
type followedLog struct {
	position   domainshare.Position
	ownerChain []byte
}

func (b base) open(ctx context.Context, text string) (openedInvitation, error) {
	invitation, err := domainshare.ParseInvitation(text)
	if err != nil {
		return openedInvitation{}, err
	}
	own, err := b.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return openedInvitation{}, err
	}
	reader, err := identity.ParseChain(own.Chain)
	if err != nil {
		return openedInvitation{}, err
	}
	if reader.Root().Fingerprint() != invitation.ReaderFingerprint {
		return openedInvitation{}, domainshare.ErrNotForThisAccount
	}
	ownerChain, err := identity.ParseChain(invitation.OwnerChain)
	if err != nil {
		return openedInvitation{}, fmt.Errorf("%w: %w", domainshare.ErrBadCode, err)
	}
	return openedInvitation{invitation: invitation, ownerChain: ownerChain, reader: reader}, nil
}

func (b base) findReceivedLink(ctx context.Context, shareID []byte) (domain.VaultID, vaultlink.Link, bool, error) {
	vaults, err := b.deps.Host.GetVaults(ctx)
	if err != nil {
		return "", vaultlink.Link{}, false, err
	}
	for _, vault := range vaults {
		link, isReceived, err := b.readReceivedLink(ctx, vault.ID)
		if err != nil {
			return "", vaultlink.Link{}, false, err
		}
		if isReceived && link.Received.ShareRootID == hex.EncodeToString(shareID) {
			return vault.ID, link, true, nil
		}
	}
	return "", vaultlink.Link{}, false, nil
}

func (b base) follow(ctx context.Context, o openedInvitation) (followedLog, error) {
	entries, chain, err := b.deps.Shares.GetShareLog(ctx, o.invitation.ShareRootID, 0)
	if errors.Is(err, syncprotocol.ErrNoShare) {
		return followedLog{}, domainshare.ErrUnknownShare
	}
	if err != nil {
		return followedLog{}, err
	}
	if err := refuseShorterChain(o.invitation.OwnerChain, chain); err != nil {
		return followedLog{}, err
	}
	authors, err := trust.NewAuthors(o.invitation.OwnerChain, chain)
	if err != nil {
		return followedLog{}, err
	}
	rules := domainshare.Rules{ShareRootID: o.invitation.ShareRootID, Reader: o.reader.Root().Sign, Authors: authors, Open: b.deps.Box.OpenBox}
	follower := domainshare.NewFollower(rules, domainshare.Position{
		VaultID: nil, Version: 0, HeadID: [32]byte{}, Epoch: 0, Keys: nil, Name: "", IsActive: false, IsEnded: false,
	})
	if err := follower.Follow(ctx, entries); err != nil {
		return followedLog{}, err
	}
	position := follower.GetPosition()
	if position.IsEnded {
		return followedLog{}, domainshare.ErrEnded
	}
	if !position.IsActive {
		return followedLog{}, domainshare.ErrUnknownShare
	}
	return followedLog{position: position, ownerChain: chain}, nil
}

// Preview is what a reader sees of an invitation before it accepts.
type Preview struct {
	ShareName        string
	OwnerFingerprint string
	IsForThisAccount bool
	IsReceived       bool
}

// ReadInvitation shows what an invitation offers.
type ReadInvitation struct{ base }

// NewReadInvitation returns the use case over the dependencies.
func NewReadInvitation(deps Deps) *ReadInvitation { return &ReadInvitation{base: base{deps: deps}} }

// Execute verifies the invitation and the share log it names, and stores nothing. An invitation for another account
// is ErrNotForThisAccount.
func (u *ReadInvitation) Execute(ctx context.Context, text string) (Preview, error) {
	o, err := u.open(ctx, text)
	if err != nil {
		return Preview{}, err
	}
	_, link, isReceived, err := u.findReceivedLink(ctx, o.invitation.ShareRootID)
	if err != nil {
		return Preview{}, err
	}
	if isReceived && !bytes.Equal(link.Received.OwnerRoot, o.ownerChain.Root().Sign) {
		return Preview{}, domainshare.ErrOwnerChanged
	}
	read, err := u.follow(ctx, o)
	if err != nil && !(isReceived && errors.Is(err, domainshare.ErrEnded)) {
		return Preview{}, err
	}
	name := read.position.Name
	if name == "" && isReceived {
		name = link.Received.Name
	}
	return Preview{
		ShareName: domainshare.CleanName(name), OwnerFingerprint: o.ownerChain.Root().Fingerprint().String(),
		IsForThisAccount: true, IsReceived: isReceived && !link.Received.IsEnded,
	}, nil
}

type AcceptedShare struct {
	VaultID domain.VaultID
	Name    string
}

// AcceptInvitation receives the folder of an invitation as a vault of this computer.
type AcceptInvitation struct {
	base
	pull *PullShare
}

// NewAcceptInvitation returns the use case over the dependencies.
func NewAcceptInvitation(deps Deps) *AcceptInvitation {
	return &AcceptInvitation{base: base{deps: deps}, pull: NewPullShare(deps)}
}

// Execute verifies the invitation, makes the folder, adds it to the vault list as a vault only this plugin writes,
// writes the link with the pinned root of the owner and runs the first pass. A share received already is refused,
// and so is an invitation for the same share with another owner root. A share the owner ended, to which the owner
// added this reader again, is received into the vault it had: the reader keeps its files and the keys of the epochs
// it held, and reads the log again from its first entry.
func (u *AcceptInvitation) Execute(ctx context.Context, text string) (AcceptedShare, error) {
	o, err := u.open(ctx, text)
	if err != nil {
		return AcceptedShare{}, err
	}
	vaultID, link, isReceived, err := u.findReceivedLink(ctx, o.invitation.ShareRootID)
	if err != nil {
		return AcceptedShare{}, err
	}
	if err := checkReceivedLink(link, isReceived, o); err != nil {
		return AcceptedShare{}, err
	}
	read, err := u.follow(ctx, o)
	if err != nil {
		return AcceptedShare{}, err
	}
	if isReceived {
		return u.joinAgain(ctx, vaultID, o, read)
	}
	return u.receive(ctx, o, read)
}

func checkReceivedLink(link vaultlink.Link, isReceived bool, o openedInvitation) error {
	if !isReceived {
		return nil
	}
	if !bytes.Equal(link.Received.OwnerRoot, o.ownerChain.Root().Sign) {
		return domainshare.ErrOwnerChanged
	}
	if !link.Received.IsEnded {
		return domainshare.ErrAlreadyReceived
	}
	return nil
}

func (u *AcceptInvitation) receive(ctx context.Context, o openedInvitation, read followedLog) (AcceptedShare, error) {
	folder, err := u.deps.Folders.MakeReceivedFolder(ctx, domainshare.CleanName(read.position.Name))
	if err != nil {
		return AcceptedShare{}, err
	}
	vault, err := u.deps.Vaults.AddManagedVault(ctx, folder.Path, folder.Name, syncprotocol.SignInPluginID)
	if err != nil {
		return AcceptedShare{}, err
	}
	unlock, err := u.deps.Locker.LockVault(ctx, vault.ID)
	if err != nil {
		return AcceptedShare{}, u.dropVault(ctx, vault.ID, err)
	}
	defer unlock()
	if err := u.writeLink(ctx, vault.ID, o, read); err != nil {
		return AcceptedShare{}, u.dropVault(ctx, vault.ID, err)
	}
	if err := u.runFirstPass(ctx, vault.ID); err != nil {
		return AcceptedShare{}, err
	}
	return AcceptedShare{VaultID: vault.ID, Name: vault.Name}, nil
}

func (u *AcceptInvitation) dropVault(ctx context.Context, id domain.VaultID, cause error) error {
	if _, err := u.deps.Vaults.RemoveVault(context.WithoutCancel(ctx), id); err != nil {
		return errors.Join(cause, fmt.Errorf("remove the vault of the share: %w", err))
	}
	return cause
}

func (u *AcceptInvitation) joinAgain(ctx context.Context, id domain.VaultID, o openedInvitation, read followedLog) (AcceptedShare, error) {
	unlock, err := u.deps.Locker.LockVault(ctx, id)
	if err != nil {
		return AcceptedShare{}, err
	}
	defer unlock()
	err = u.deps.Links.UpdateLink(ctx, id, func(fresh *vaultlink.Link) error {
		var seen map[string]vaultlink.SeenRevision
		if fresh.Received != nil {
			seen = fresh.Received.Seen
		}
		received, err := u.buildReceivedRecord(ctx, o, read, seen)
		if err != nil {
			return err
		}
		fresh.Received, fresh.CatalogName, fresh.ServerSeq = &received, read.position.Name, 0
		return nil
	})
	if err != nil {
		return AcceptedShare{}, err
	}
	if err := u.runFirstPass(ctx, id); err != nil {
		return AcceptedShare{}, err
	}
	return AcceptedShare{VaultID: id, Name: domainshare.CleanName(read.position.Name)}, nil
}

func (u *AcceptInvitation) runFirstPass(ctx context.Context, id domain.VaultID) error {
	result, err := u.pull.Execute(ctx, id)
	if err != nil {
		return err
	}
	return u.deps.Statuses.WriteStatus(ctx, id, getStatus(result, u.deps.Clock.Now()))
}

func (u *AcceptInvitation) buildReceivedRecord(ctx context.Context, o openedInvitation, read followedLog, seen map[string]vaultlink.SeenRevision) (vaultlink.ShareRecord, error) {
	keys, err := read.position.Keys.Encode()
	if err != nil {
		return vaultlink.ShareRecord{}, err
	}
	sealed, err := u.deps.Crypto.SealData(ctx, vaultlink.GetStateKeyContext(o.invitation.ShareRootID), keys)
	if err != nil {
		return vaultlink.ShareRecord{}, fmt.Errorf("seal the keys of the share: %w", err)
	}
	return vaultlink.ShareRecord{
		ShareRootID: hex.EncodeToString(o.invitation.ShareRootID), OwnerRoot: o.ownerChain.Root().Sign, OwnerChain: read.ownerChain,
		SealedKeys: sealed, ServerVersion: read.position.Version, HeadID: read.position.HeadID[:], VaultID: read.position.VaultID,
		Epoch: read.position.Epoch, Name: read.position.Name, IsEnded: false, Seen: seen,
	}, nil
}

func (u *AcceptInvitation) writeLink(ctx context.Context, id domain.VaultID, o openedInvitation, read followedLog) error {
	received, err := u.buildReceivedRecord(ctx, o, read, nil)
	if err != nil {
		return err
	}
	link := vaultlink.Link{
		RemoteVaultID: "", SealedKey: nil, LaterKeys: nil, CatalogName: read.position.Name, ServerSeq: 0, HighSeq: 0, FeedCursor: vaultlink.FeedCursor{FeedID: "", Seq: 0},
		RestoreEpochID: nil, Recovery: nil, Removed: nil, Retired: nil,
		Kind: vaultlink.KindReceived, Received: &received,
	}
	return u.deps.Links.WriteLink(ctx, id, link)
}
