package share

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// CreateShare makes the share root of a folder and rewraps the files under it.
type CreateShare struct{ base }

// NewCreateShare returns the use case over the dependencies.
func NewCreateShare(deps Deps) *CreateShare { return &CreateShare{base: base{deps: deps}} }

// Execute shares the folder of the vault with no reader yet. It refuses the whole vault, a folder inside a share
// or holding one, a vault that is received and a vault that is not connected. It returns after the files are
// rewrapped; a rewrap that ends early is resumed by the next pass over the vault.
func (u *CreateShare) Execute(ctx context.Context, id domain.VaultID, folder string) error {
	normalized := domainshare.NormalizeFolder(folder)
	if normalized == "" {
		return domainshare.ErrWholeVault
	}
	remote, err := u.getRemote(ctx, id)
	if err != nil {
		return err
	}
	roots, err := NewReadShares(u.deps).Execute(ctx, id)
	if err != nil {
		return err
	}
	if err := domainshare.CheckNesting(normalized, domainshare.ListFolders(roots)); err != nil {
		return err
	}
	root, err := u.create(ctx, id, remote, normalized)
	if err != nil {
		return err
	}
	return newRewrap(u.base).run(ctx, id, root)
}

func (u *CreateShare) create(ctx context.Context, id domain.VaultID, remote []byte, folder string) (domainshare.Root, error) {
	shareID, err := note.NewID(u.deps.Random, idSize)
	if err != nil {
		return domainshare.Root{}, err
	}
	key, err := note.NewID(u.deps.Random, keySize)
	if err != nil {
		return domainshare.Root{}, err
	}
	keys := domainshare.Keys{}.Add(0, [32]byte(key))
	body := domainshare.OwnerBody{Folder: folder, Keys: keys, Readers: nil}
	entry, err := u.sign(ctx, shareID, 1, [32]byte{}, draft{
		kind: syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, subject: nil, vaultID: remote, epoch: 0, body: body, seals: nil, previous: nil,
	})
	if err != nil {
		return domainshare.Root{}, err
	}
	if err := u.setPhase(ctx, id, shareID, domainshare.PhaseCreated); err != nil {
		return domainshare.Root{}, err
	}
	if err := u.appendEntry(ctx, entry); err != nil {
		return domainshare.Root{}, errors.Join(err, u.dropPhase(ctx, id, shareID))
	}
	return domainshare.Root{
		ID: shareID, VaultID: remote, Folder: folder, Epoch: 0, Keys: keys, Readers: nil, Version: 1,
		HeadID: syncprotocol.GetShareLogEntryID(entry), IsClosed: false, Phase: domainshare.PhaseCreated,
	}, nil
}

// RewrapFolder rewraps the files of a share that is not rewrapped yet.
type RewrapFolder struct{ base }

// NewRewrapFolder returns the use case over the dependencies.
func NewRewrapFolder(deps Deps) *RewrapFolder { return &RewrapFolder{base: base{deps: deps}} }

// Execute commits a share revision for each live file under the folder of the root that is not one yet, with the
// blob of its head, so a pass that stopped halfway continues where it stopped.
func (u *RewrapFolder) Execute(ctx context.Context, id domain.VaultID, root domainshare.Root) error {
	return newRewrap(u.base).run(ctx, id, root)
}

type rewrap struct {
	base
	remote   []byte
	vaultKey [32]byte
	shareKey [32]byte
	root     domainshare.Root
	authors  trust.Authors
	author   []byte
}

func newRewrap(b base) *rewrap {
	return &rewrap{base: b, remote: nil, vaultKey: [32]byte{}, shareKey: [32]byte{}, root: domainshare.Root{}, authors: trust.Authors{}, author: nil}
}

func (r *rewrap) run(ctx context.Context, id domain.VaultID, root domainshare.Root) error {
	if err := r.open(ctx, id, root); err != nil {
		return err
	}
	heads, err := r.readHeads(ctx)
	if err != nil {
		return err
	}
	var revisions []*syncv1.Revision
	for _, head := range heads {
		revision, err := r.build(head)
		if err != nil {
			return err
		}
		if revision != nil {
			revisions = append(revisions, revision)
		}
	}
	for start := 0; start < len(revisions); start += batchSize {
		if err := r.commit(ctx, revisions[start:min(start+batchSize, len(revisions))]); err != nil {
			return err
		}
	}
	return r.setPhase(ctx, id, root.ID, domainshare.PhaseRewrapped)
}

func (r *rewrap) open(ctx context.Context, id domain.VaultID, root domainshare.Root) error {
	link, err := r.deps.Links.ReadLink(ctx, id)
	if err != nil {
		return err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return fmt.Errorf("read the remote id of the link: %w", err)
	}
	key, err := r.deps.Crypto.OpenData(ctx, vaultlink.GetStateKeyContext(remote), link.SealedKey)
	if err != nil {
		return fmt.Errorf("open the vault key: %w", err)
	}
	if r.authors, err = r.getAuthors(ctx); err != nil {
		return err
	}
	own, err := r.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return err
	}
	shareKey, isKnown := root.GetKey()
	if !isKnown {
		return errNoKey
	}
	r.remote, r.vaultKey, r.shareKey, r.root, r.author = remote, [32]byte(key), shareKey, root, own.DeviceSignKey
	return nil
}

func (r *rewrap) readHeads(ctx context.Context) ([]*syncv1.StoredRevision, error) {
	var heads []*syncv1.StoredRevision
	var since uint64
	for {
		page, err := r.deps.Revisions.GetChanges(ctx, r.remote, since, 0)
		if err != nil {
			return nil, err
		}
		heads = append(heads, page.GetChanges()...)
		if !page.GetHasMore() {
			return heads, nil
		}
		since = page.GetNextSeq()
	}
}

func (r *rewrap) build(head *syncv1.StoredRevision) (*syncv1.Revision, error) {
	rev := head.GetRevision()
	if rev == nil || rev.GetDeleted() || len(rev.GetShareRootId()) > 0 || !r.isVerified(head) {
		return nil, nil
	}
	meta, isOpen := r.openMeta(rev)
	if !isOpen {
		return nil, nil
	}
	relative, isUnder := domainshare.GetRelativePath(r.root.Folder, meta.Path)
	if !isUnder {
		return nil, nil
	}
	meta.Path = relative
	sealed, err := domainshare.SealMeta(r.deps.Random, r.shareKey, r.root.ID, rev.GetFileId(), r.root.Epoch, meta)
	if err != nil {
		return nil, err
	}
	next := &syncv1.Revision{
		VaultId: rev.GetVaultId(), FileId: rev.GetFileId(), ParentRev: head.GetRevId(), BlobId: rev.GetBlobId(), BlobHash: rev.GetBlobHash(),
		BlobSize: rev.GetBlobSize(), EncMeta: sealed, PathHash: rev.GetPathHash(), ShareRootId: r.root.ID, Epoch: r.root.Epoch,
		AuthorDeviceSignKey: r.author,
	}
	return next, nil
}

func (r *rewrap) openMeta(rev *syncv1.Revision) (note.Meta, bool) {
	meta, err := note.OpenMeta(r.vaultKey, rev.GetVaultId(), rev.GetFileId(), rev.GetEncMeta())
	return meta, err == nil
}

func (r *rewrap) isVerified(head *syncv1.StoredRevision) bool {
	rev := head.GetRevision()
	id := syncprotocol.GetRevisionID(rev)
	return syncprotocol.CheckRevision(rev) == nil && bytes.Equal(head.GetRevId(), id[:]) && r.authors.IsTrusted(rev.GetAuthorDeviceSignKey())
}

func (r *rewrap) commit(ctx context.Context, batch []*syncv1.Revision) error {
	for range maxTries {
		for _, revision := range batch {
			signature, err := r.deps.Identity.SignAsDevice(ctx, syncprotocol.GetRevisionMessage(revision))
			if err != nil {
				return err
			}
			revision.Signature = signature
		}
		response, err := r.deps.Revisions.CommitRevisions(ctx, r.remote, batch)
		if err != nil {
			return err
		}
		if response.GetCommitted() {
			return nil
		}
		if batch, err = r.resolve(batch, response.GetConflicts()); err != nil || len(batch) == 0 {
			return err
		}
	}
	return domainshare.ErrLogBusy
}

func (r *rewrap) resolve(batch []*syncv1.Revision, conflicts []*syncv1.Conflict) ([]*syncv1.Revision, error) {
	var next []*syncv1.Revision
	for _, revision := range batch {
		at := findConflict(conflicts, revision.GetFileId())
		if at == nil {
			next = append(next, revision)
			continue
		}
		if at.GetKind() != syncv1.ConflictKind_CONFLICT_KIND_PARENT_MISMATCH {
			return nil, fmt.Errorf("%w: %v", errUnknownKind, at.GetKind())
		}
		rebuilt, err := r.build(at.GetHead())
		if err != nil {
			return nil, err
		}
		if rebuilt != nil {
			next = append(next, rebuilt)
		}
	}
	return next, nil
}

func findConflict(conflicts []*syncv1.Conflict, fileID []byte) *syncv1.Conflict {
	for _, conflict := range conflicts {
		if bytes.Equal(conflict.GetFileId(), fileID) {
			return conflict
		}
	}
	return nil
}
