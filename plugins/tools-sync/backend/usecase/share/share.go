package share

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	maxTries  = 3
	idSize    = 16
	keySize   = 32
	batchSize = 100
)

var (
	errMoved       = errors.New("another device appended to the share log first")
	errNoKey       = errors.New("the share has no key for its epoch")
	errNoHead      = errors.New("the share log has no entry at its head")
	errBadHead     = errors.New("the entry at the head of the share log does not verify")
	errUnknownKind = errors.New("the server refused a rewrap with a conflict that has no answer")
)

// Deps are what the use cases of sharing need.
type Deps struct {
	Shares    ports.Shares
	Revisions ports.Revisions
	Chain     ports.Chain
	Crypto    ports.Crypto
	Identity  ports.Identity
	Links     ports.Links
	States    ports.ShareStates
	Clock     ports.Clock
	Random    io.Reader
}

type base struct{ deps Deps }

// ref names the folder of a vault. missing is what a use case answers for a folder that is not shared.
type ref struct {
	vault   domain.VaultID
	folder  string
	missing error
}

// getRemote returns the id the server knows the vault by, and refuses a vault that is not connected or is received.
func (b base) getRemote(ctx context.Context, id domain.VaultID) ([]byte, error) {
	link, err := b.deps.Links.ReadLink(ctx, id)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return nil, domainshare.ErrNotConnected
	}
	if err != nil {
		return nil, err
	}
	if link.Kind == vaultlink.KindReceived {
		return nil, domainshare.ErrReceivedVault
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return nil, fmt.Errorf("read the remote id of the link: %w", err)
	}
	return remote, nil
}

func (b base) getAuthors(ctx context.Context) (trust.Authors, error) {
	own, err := b.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	stored, err := b.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	return trust.NewAuthors(own.Chain, stored)
}

// ReadShares reads the shares of a vault from the share log: the head entry of each, opened.
type ReadShares struct{ base }

// NewReadShares returns the use case over the dependencies.
func NewReadShares(deps Deps) *ReadShares { return &ReadShares{base: base{deps: deps}} }

// Execute returns every share of the vault, closed ones too, whose keys still open what was shared. A vault that is
// not connected, or is received, has none.
func (u *ReadShares) Execute(ctx context.Context, id domain.VaultID) ([]domainshare.Root, error) {
	remote, err := u.getRemote(ctx, id)
	if errors.Is(err, domainshare.ErrNotConnected) || errors.Is(err, domainshare.ErrReceivedVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	summaries, err := u.deps.Shares.ListShares(ctx)
	if err != nil {
		return nil, err
	}
	states, err := u.deps.States.ReadShareStates(ctx, id)
	if err != nil {
		return nil, err
	}
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	var roots []domainshare.Root
	for _, summary := range summaries {
		if !bytes.Equal(summary.GetVaultId(), remote) {
			continue
		}
		root, err := u.readRoot(ctx, summary, own.Chain)
		if err != nil {
			return nil, err
		}
		root.Phase = getPhase(states, root.ID)
		roots = append(roots, root)
	}
	return roots, nil
}

func getPhase(states []domainshare.Progress, id []byte) domainshare.Phase {
	for _, one := range states {
		if one.ShareRootID == hex.EncodeToString(id) {
			return one.Phase
		}
	}
	return domainshare.PhaseActive
}

func (u *ReadShares) readRoot(ctx context.Context, summary *syncv1.ShareSummary, ownChain []byte) (domainshare.Root, error) {
	since := uint64(0)
	if summary.GetHeadVersion() > 1 {
		since = summary.GetHeadVersion() - 1
	}
	entries, chain, err := u.deps.Shares.GetShareLog(ctx, summary.GetShareRootId(), since)
	if err != nil {
		return domainshare.Root{}, err
	}
	if len(entries) == 0 {
		return domainshare.Root{}, errNoHead
	}
	entry := entries[len(entries)-1]
	authors, err := trust.NewAuthors(ownChain, chain)
	if err != nil {
		return domainshare.Root{}, err
	}
	if err := checkHead(entry, summary.GetShareRootId(), authors); err != nil {
		return domainshare.Root{}, err
	}
	plain, err := u.deps.Crypto.OpenData(ctx, domainshare.GetOwnerBodyContext(entry.GetShareRootId(), entry.GetVersion()), entry.GetEncOwnerBody())
	if err != nil {
		return domainshare.Root{}, fmt.Errorf("open the owner body of share entry %d: %w", entry.GetVersion(), err)
	}
	body, err := domainshare.ParseOwnerBody(plain)
	if err != nil {
		return domainshare.Root{}, err
	}
	return domainshare.Root{
		ID: entry.GetShareRootId(), VaultID: summary.GetVaultId(), Folder: domainshare.NormalizeFolder(body.Folder), Epoch: entry.GetEpoch(),
		Keys: body.Keys, Readers: body.Readers, Version: entry.GetVersion(), HeadID: syncprotocol.GetShareLogEntryID(entry),
		IsClosed: summary.GetIsClosed() || entry.GetKind() == syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE, Phase: domainshare.PhaseActive,
	}, nil
}

func checkHead(entry *syncv1.ShareLogEntry, id []byte, authors trust.Authors) error {
	if syncprotocol.CheckShareLogEntry(entry) != nil || !bytes.Equal(entry.GetShareRootId(), id) || len(entry.GetSignature()) != 64 ||
		!authors.IsTrusted(entry.GetAuthorDeviceSignKey()) ||
		!identity.VerifyForPlugin(entry.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID, syncprotocol.GetShareLogEntryMessage(entry), [64]byte(entry.GetSignature())) {
		return errBadHead
	}
	return nil
}

// getRoot returns the open share of the folder.
func (b base) getRoot(ctx context.Context, at ref) (domainshare.Root, error) {
	if _, err := b.getRemote(ctx, at.vault); err != nil {
		return domainshare.Root{}, err
	}
	roots, err := NewReadShares(b.deps).Execute(ctx, at.vault)
	if err != nil {
		return domainshare.Root{}, err
	}
	root, isFound := domainshare.FindRoot(roots, at.folder)
	if !isFound {
		return domainshare.Root{}, at.missing
	}
	return root, nil
}

// draft is an entry of the share log before its place in the log, its bodies and its signature are set.
type draft struct {
	kind     syncv1.ShareLogKind
	subject  []byte
	vaultID  []byte
	epoch    uint32
	body     domainshare.OwnerBody
	seals    []*syncv1.ShareSeal
	previous []byte
}

// sign returns the entry of the share log that follows the entry with the id prev, signed by this device.
func (b base) sign(ctx context.Context, id []byte, version uint64, prev [32]byte, d draft) (*syncv1.ShareLogEntry, error) {
	key, isKnown := d.body.Keys.Get(d.epoch)
	if !isKnown {
		return nil, errNoKey
	}
	owner, err := d.body.Encode()
	if err != nil {
		return nil, err
	}
	sealedOwner, err := b.deps.Crypto.SealData(ctx, domainshare.GetOwnerBodyContext(id, version), owner)
	if err != nil {
		return nil, fmt.Errorf("seal the owner body: %w", err)
	}
	name := domainshare.Body{Name: domainshare.GetShareName(d.body.Folder)}
	sealedShare, err := domainshare.SealBody(b.deps.Random, key, id, d.epoch, name)
	if err != nil {
		return nil, err
	}
	author, err := b.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	entry := &syncv1.ShareLogEntry{
		ShareRootId: id, Version: version, Kind: d.kind, VaultId: d.vaultID, Epoch: d.epoch, SubjectRootSignKey: d.subject,
		EncOwnerBody: sealedOwner, EncShareBody: sealedShare, Seals: d.seals, PrevKeySeal: d.previous, AuthorDeviceSignKey: author.DeviceSignKey,
	}
	if version > 1 {
		entry.PrevHash = prev[:]
	}
	if entry.Signature, err = b.deps.Identity.SignAsDevice(ctx, syncprotocol.GetShareLogEntryMessage(entry)); err != nil {
		return nil, err
	}
	return entry, nil
}

// appendEntry adds the entry to the log. It returns errMoved when another device was first.
func (b base) appendEntry(ctx context.Context, entry *syncv1.ShareLogEntry) error {
	response, err := b.deps.Shares.AppendShareLogEntry(ctx, entry)
	if errors.Is(err, syncprotocol.ErrReaderRefused) {
		return domainshare.ErrReaderNotAdded
	}
	if err != nil {
		return err
	}
	if !response.GetCommitted() {
		return errMoved
	}
	return nil
}

// change appends the entry build makes from the share of the folder as it is now. Another device that appended
// first makes it read the share again, a few times.
func (b base) change(ctx context.Context, at ref, build func(root domainshare.Root) (draft, error)) (domainshare.Root, error) {
	for range maxTries {
		root, err := b.getRoot(ctx, at)
		if err != nil {
			return domainshare.Root{}, err
		}
		d, err := build(root)
		if err != nil {
			return domainshare.Root{}, err
		}
		entry, err := b.sign(ctx, root.ID, root.Version+1, root.HeadID, d)
		if err != nil {
			return domainshare.Root{}, err
		}
		err = b.appendEntry(ctx, entry)
		if err == nil {
			return root, nil
		}
		if !errors.Is(err, errMoved) {
			return domainshare.Root{}, err
		}
	}
	return domainshare.Root{}, domainshare.ErrLogBusy
}

// setPhase records where this device stands with the share.
func (b base) setPhase(ctx context.Context, id domain.VaultID, shareID []byte, phase domainshare.Phase) error {
	states, err := b.deps.States.ReadShareStates(ctx, id)
	if err != nil {
		return err
	}
	key := hex.EncodeToString(shareID)
	kept := make([]domainshare.Progress, 0, len(states)+1)
	for _, one := range states {
		if one.ShareRootID != key {
			kept = append(kept, one)
		}
	}
	return b.deps.States.WriteShareStates(ctx, id, append(kept, domainshare.Progress{ShareRootID: key, Phase: phase}))
}

// dropPhase forgets where this device stands with the share.
func (b base) dropPhase(ctx context.Context, id domain.VaultID, shareID []byte) error {
	states, err := b.deps.States.ReadShareStates(ctx, id)
	if err != nil {
		return err
	}
	key := hex.EncodeToString(shareID)
	kept := make([]domainshare.Progress, 0, len(states))
	for _, one := range states {
		if one.ShareRootID != key {
			kept = append(kept, one)
		}
	}
	return b.deps.States.WriteShareStates(ctx, id, kept)
}

// Overview is the sharing of one folder as the owner lists it.
type Overview struct {
	State   string
	Readers []domainshare.Reader
}

// GetShare reads the sharing of a folder.
type GetShare struct{ base }

// NewGetShare returns the use case over the dependencies.
func NewGetShare(deps Deps) *GetShare { return &GetShare{base: base{deps: deps}} }

// Execute returns the state of the folder and its readers, the removed ones too.
func (u *GetShare) Execute(ctx context.Context, id domain.VaultID, folder string) (Overview, error) {
	root, err := u.getRoot(ctx, ref{vault: id, folder: folder, missing: domainshare.ErrNotShared})
	if errors.Is(err, domainshare.ErrNotShared) {
		return Overview{State: domainshare.StateNotShared, Readers: nil}, nil
	}
	if err != nil {
		return Overview{}, err
	}
	return Overview{State: root.GetState(), Readers: root.Readers}, nil
}
