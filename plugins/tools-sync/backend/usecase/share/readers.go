package share

import (
	"bytes"
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
)

// AddReaderInput is the folder, the code of the person and the label the owner gives it. IsConfirmed says the owner
// accepts a label that another person had before.
type AddReaderInput struct {
	VaultID     domain.VaultID
	Folder      string
	Code        string
	Label       string
	IsConfirmed bool
}

// Invited is what the owner sends a reader: the text of the invitation and the account fingerprint of the reader.
type Invited struct {
	Text        string
	Fingerprint string
}

type AddReader struct{ base }

// NewAddReader returns the use case over the dependencies.
func NewAddReader(deps Deps) *AddReader { return &AddReader{base: base{deps: deps}} }

// Execute seals the key of the share to the account of the reader and appends the entry that makes it a reader. It
// waits for a share whose files are not all rewrapped (ErrNotReady), and refuses the owner itself, a reader the
// folder has and a label of another person that is not confirmed.
func (u *AddReader) Execute(ctx context.Context, in AddReaderInput) (Invited, error) {
	chain, err := domainshare.ParseReaderCode(in.Code)
	if err != nil {
		return Invited{}, err
	}
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return Invited{}, err
	}
	ownChain, err := identity.ParseChain(own.Chain)
	if err != nil {
		return Invited{}, err
	}
	if bytes.Equal(ownChain.Root().Sign, chain.Root().Sign) {
		return Invited{}, domainshare.ErrOwnIdentity
	}
	at := ref{vault: in.VaultID, folder: in.Folder, missing: domainshare.ErrNotReady}
	root, err := u.change(ctx, at, func(root domainshare.Root) (draft, error) { return u.draftAdd(root, chain, in) })
	if err != nil {
		return Invited{}, err
	}
	if err := u.setPhase(ctx, in.VaultID, root.ID, domainshare.PhaseActive); err != nil {
		return Invited{}, err
	}
	fingerprint := chain.Root().Fingerprint()
	text := domainshare.NewInvitation(domainshare.Invitation{ShareRootID: root.ID, ReaderFingerprint: fingerprint, OwnerChain: own.Chain})
	return Invited{Text: text, Fingerprint: fingerprint.String()}, nil
}

func (u *AddReader) draftAdd(root domainshare.Root, chain identity.Chain, in AddReaderInput) (draft, error) {
	if root.Phase == domainshare.PhaseCreated {
		return draft{}, domainshare.ErrNotReady
	}
	readerRoot := chain.Root().Sign
	if err := domainshare.CheckReader(root.Readers, readerRoot, in.Label, in.IsConfirmed); err != nil {
		return draft{}, err
	}
	key, isKnown := root.GetKey()
	if !isKnown {
		return draft{}, errNoKey
	}
	sealed, err := domainshare.SealToReader(u.deps.Random, chain.AccountBoxKey(), root.ID, readerRoot, root.Epoch, key)
	if err != nil {
		return draft{}, err
	}
	reader := domainshare.Reader{
		Root: readerRoot, Chain: chain.Bytes(), Label: in.Label, Fingerprint: chain.Root().Fingerprint().String(),
		AddedAt: u.deps.Clock.Now().UTC(), RemovedAt: time.Time{},
	}
	body := domainshare.OwnerBody{Folder: root.Folder, Keys: root.Keys, Readers: domainshare.SetReader(root.Readers, reader)}
	return draft{
		kind: syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, subject: readerRoot, vaultID: nil, epoch: root.Epoch, body: body,
		seals: []*syncv1.ShareSeal{{ReaderRootSignKey: readerRoot, Sealed: sealed}}, previous: nil,
	}, nil
}

type RemoveReader struct{ base }

// NewRemoveReader returns the use case over the dependencies.
func NewRemoveReader(deps Deps) *RemoveReader { return &RemoveReader{base: base{deps: deps}} }

// Execute appends the entry that removes the reader with the account fingerprint and starts the next epoch: a new
// key, sealed to each reader that remains.
func (u *RemoveReader) Execute(ctx context.Context, id domain.VaultID, folder, fingerprint string) error {
	at := ref{vault: id, folder: folder, missing: domainshare.ErrNotShared}
	_, err := u.change(ctx, at, func(root domainshare.Root) (draft, error) {
		reader, isFound := root.FindReaderByFingerprint(fingerprint)
		if !isFound {
			return draft{}, domainshare.ErrUnknownReader
		}
		readers := domainshare.MarkReaderRemoved(root.Readers, reader.Root, u.deps.Clock.Now().UTC())
		return u.newRotation(root, reader.Root, readers)
	})
	return err
}

func (b base) newRotation(root domainshare.Root, subject []byte, readers []domainshare.Reader) (draft, error) {
	current, isKnown := root.GetKey()
	if !isKnown {
		return draft{}, errNoKey
	}
	next := root.Epoch + 1
	raw, err := note.NewID(b.deps.Random, keySize)
	if err != nil {
		return draft{}, err
	}
	fresh := [32]byte(raw)
	previous, err := domainshare.SealPreviousKey(b.deps.Random, fresh, current, root.ID, next)
	if err != nil {
		return draft{}, err
	}
	var seals []*syncv1.ShareSeal
	for _, reader := range readers {
		if !reader.IsActive() {
			continue
		}
		seal, err := b.sealTo(reader, root.ID, next, fresh)
		if err != nil {
			return draft{}, err
		}
		seals = append(seals, seal)
	}
	kind := syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE
	if len(subject) > 0 {
		kind = syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER
	}
	body := domainshare.OwnerBody{Folder: root.Folder, Keys: root.Keys.Add(next, fresh), Readers: readers}
	return draft{kind: kind, subject: subject, vaultID: nil, epoch: next, body: body, seals: seals, previous: previous}, nil
}

func (b base) sealTo(reader domainshare.Reader, shareID []byte, epoch uint32, key [32]byte) (*syncv1.ShareSeal, error) {
	chain, err := identity.ParseChain(reader.Chain)
	if err != nil {
		return nil, err
	}
	sealed, err := domainshare.SealToReader(b.deps.Random, chain.AccountBoxKey(), shareID, reader.Root, epoch, key)
	if err != nil {
		return nil, err
	}
	return &syncv1.ShareSeal{ReaderRootSignKey: reader.Root, Sealed: sealed}, nil
}

// StopShare stops sharing a folder.
type StopShare struct{ base }

// NewStopShare returns the use case over the dependencies.
func NewStopShare(deps Deps) *StopShare { return &StopShare{base: base{deps: deps}} }

// Execute appends the entry that closes the share and removes every reader. The files keep their share metadata
// until they are written again; the keys stay in the log for the owner to read them.
func (u *StopShare) Execute(ctx context.Context, id domain.VaultID, folder string) error {
	at := ref{vault: id, folder: folder, missing: domainshare.ErrNotShared}
	_, err := u.change(ctx, at, func(root domainshare.Root) (draft, error) {
		body := domainshare.OwnerBody{
			Folder: root.Folder, Keys: root.Keys, Readers: domainshare.MarkAllRemoved(root.Readers, u.deps.Clock.Now().UTC()),
		}
		return draft{kind: syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE, subject: nil, vaultID: nil, epoch: root.Epoch, body: body, seals: nil, previous: nil}, nil
	})
	return err
}

// ShareFolder shares a folder with a reader, making the share first when the folder has none.
type ShareFolder struct{ deps Deps }

// NewShareFolder returns the use case over the dependencies.
func NewShareFolder(deps Deps) *ShareFolder { return &ShareFolder{deps: deps} }

func (u *ShareFolder) Execute(ctx context.Context, in AddReaderInput) (Invited, error) {
	overview, err := NewGetShare(u.deps).Execute(ctx, in.VaultID, in.Folder)
	if err != nil {
		return Invited{}, err
	}
	if overview.State == domainshare.StateNotShared {
		if err := NewCreateShare(u.deps).Execute(ctx, in.VaultID, in.Folder); err != nil {
			return Invited{}, err
		}
	}
	return NewAddReader(u.deps).Execute(ctx, in)
}
