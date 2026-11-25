package receive

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// The states of a received share.
const (
	// StateReceiving is a share that was not synced yet.
	StateReceiving = "receiving"
	// StateReady is a share synced at least once.
	StateReady = "ready"
	// StateEnded is a share the owner ended.
	StateEnded = "ended"
)

// Deps are what the use cases of receiving need.
type Deps struct {
	Shares   ports.Shares
	Blobs    ports.BlobURLReader
	Crypto   ports.Crypto
	Box      ports.Box
	Identity ports.Identity
	Host     ports.Host
	Notes    ports.NoteHost
	Links    ports.Links
	Records  ports.Records
	Statuses ports.Statuses
	Vaults   ports.VaultList
	Folders  ports.Folders
	Clock    ports.Clock
	Locker   ports.VaultLocker
}

type base struct{ deps Deps }

// ReaderCode is the code a person gives the owner of a share, and the fingerprint of its account.
type ReaderCode struct {
	Code        string
	Fingerprint string
}

// GetReaderCode makes the reader code of this account.
type GetReaderCode struct{ base }

// NewGetReaderCode returns the use case over the dependencies.
func NewGetReaderCode(deps Deps) *GetReaderCode { return &GetReaderCode{base: base{deps: deps}} }

// Execute returns the chain of this account as a code, with the fingerprint the two people compare.
func (u *GetReaderCode) Execute(ctx context.Context) (ReaderCode, error) {
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return ReaderCode{}, err
	}
	chain, err := identity.ParseChain(own.Chain)
	if err != nil {
		return ReaderCode{}, err
	}
	return ReaderCode{Code: domainshare.NewReaderCode(own.Chain), Fingerprint: chain.Root().Fingerprint().String()}, nil
}

// Received is a share this computer received: its vault, the owner and how its last pass ended.
type Received struct {
	VaultID          domain.VaultID
	Name             string
	OwnerFingerprint string
	State            string
	LastSyncedAt     time.Time
	ConflictCount    int
}

// ListReceived lists the shares this computer received.
type ListReceived struct{ base }

// NewListReceived returns the use case over the dependencies.
func NewListReceived(deps Deps) *ListReceived { return &ListReceived{base: base{deps: deps}} }

// Execute returns the received shares in the order of the vault list.
func (u *ListReceived) Execute(ctx context.Context) ([]Received, error) {
	vaults, err := u.deps.Host.GetVaults(ctx)
	if err != nil {
		return nil, err
	}
	var listed []Received
	for _, vault := range vaults {
		link, isReceived, err := u.readReceivedLink(ctx, vault.ID)
		if err != nil {
			return nil, err
		}
		if !isReceived {
			continue
		}
		one, err := u.describe(ctx, vault, link)
		if err != nil {
			return nil, err
		}
		listed = append(listed, one)
	}
	return listed, nil
}

func (u *ListReceived) describe(ctx context.Context, vault domain.Vault, link vaultlink.Link) (Received, error) {
	status, err := u.deps.Statuses.ReadStatus(ctx, vault.ID)
	if err != nil {
		return Received{}, err
	}
	state := StateReady
	switch {
	case link.Received.IsEnded:
		state = StateEnded
	case status.LastSyncedAt.IsZero():
		state = StateReceiving
	}
	chain, err := identity.ParseChain(link.Received.OwnerChain)
	if err != nil {
		return Received{}, err
	}
	return Received{
		VaultID: vault.ID, Name: vault.Name, OwnerFingerprint: chain.Root().Fingerprint().String(), State: state, LastSyncedAt: status.LastSyncedAt,
		ConflictCount: status.ConflictCount,
	}, nil
}

func (b base) readReceivedLink(ctx context.Context, id domain.VaultID) (vaultlink.Link, bool, error) {
	link, err := b.deps.Links.ReadLink(ctx, id)
	if err != nil {
		if errors.Is(err, vaultlink.ErrNotConnected) {
			return vaultlink.Link{}, false, nil
		}
		return vaultlink.Link{}, false, err
	}
	if link.Kind != vaultlink.KindReceived || link.Received == nil {
		return vaultlink.Link{}, false, nil
	}
	return link, true, nil
}
