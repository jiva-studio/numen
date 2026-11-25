// Package endorse makes the revisions that a device about to be revoked wrote count after the revoke: a device that stays signs a new
// revision over each head the revoked device authored, with the same file, so every head of the account has an author that is
// trusted. What the revoked device writes later has no endorsement and no device trusts it.
package endorse

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const maxRounds = 3

// ErrNotEndorsed means heads of the revoked devices were still there after the last round, because other devices wrote to the same files
// again and again.
var ErrNotEndorsed = errors.New("the files of the devices to revoke changed while they were endorsed")

var errNoConflict = errors.New("the server refused the endorsements and named no file")

var errUnknownConflict = errors.New("the server refused the endorsements and named none of the files it was sent")

// Deps are what the use case needs.
type Deps struct {
	Identity  ports.Identity
	Host      ports.Host
	State     ports.Links
	Revisions ports.Revisions
}

// Endorse signs the heads of the revoked devices.
type Endorse struct{ deps Deps }

// NewEndorse returns the use case over the dependencies.
func NewEndorse(deps Deps) *Endorse { return &Endorse{deps: deps} }

// Execute signs, in every synced vault of this device, a revision over each head whose valid author is one of the revoked devices. A head
// that does not verify, a note of a folder received from another account and a head of another author are left as they are.
func (u *Endorse) Execute(ctx context.Context, revoked [][]byte) error {
	if len(revoked) == 0 {
		return nil
	}
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return err
	}
	vaults, err := u.deps.Host.GetVaults(ctx)
	if err != nil {
		return err
	}
	for _, vault := range vaults {
		link, err := u.deps.State.ReadLink(ctx, vault.ID)
		if errors.Is(err, vaultlink.ErrNotConnected) || (err == nil && link.Kind == vaultlink.KindReceived) {
			continue
		}
		if err != nil {
			return err
		}
		remote, err := decodeRemote(link)
		if err != nil {
			return err
		}
		if err := u.confirmVault(ctx, remote, own.DeviceSignKey, revoked); err != nil {
			return fmt.Errorf("endorse the vault %s: %w", vault.ID, err)
		}
	}
	return nil
}

func (u *Endorse) confirmVault(ctx context.Context, remote, own []byte, revoked [][]byte) error {
	for range maxRounds {
		stored, err := u.listHeads(ctx, remote, revoked)
		if err != nil || len(stored) == 0 {
			return err
		}
		for start := 0; start < len(stored); start += note.BatchFiles {
			if err := u.commit(ctx, remote, own, stored[start:min(start+note.BatchFiles, len(stored))]); err != nil {
				return err
			}
		}
	}
	left, err := u.listHeads(ctx, remote, revoked)
	if err != nil {
		return err
	}
	if len(left) > 0 {
		return ErrNotEndorsed
	}
	return nil
}

func (u *Endorse) listHeads(ctx context.Context, remote []byte, revoked [][]byte) ([]*syncv1.StoredRevision, error) {
	isRevoked := func(sign []byte) bool {
		return slices.ContainsFunc(revoked, func(key []byte) bool { return bytes.Equal(key, sign) })
	}
	var found []*syncv1.StoredRevision
	var since uint64
	for pages := 1; ; pages++ {
		page, err := u.deps.Revisions.GetChanges(ctx, remote, since, 0)
		if err != nil {
			return nil, err
		}
		for _, stored := range page.GetChanges() {
			if note.CheckRevision(stored, remote, isRevoked) == nil {
				found = append(found, stored)
			}
		}
		if err := syncprotocol.CheckPage(pages, since, page.GetNextSeq(), page.GetHasMore()); err != nil {
			return nil, err
		}
		since = page.GetNextSeq()
		if !page.GetHasMore() {
			return found, nil
		}
	}
}

func (u *Endorse) commit(ctx context.Context, remote, own []byte, heads []*syncv1.StoredRevision) error {
	revisions := make([]*syncv1.Revision, 0, len(heads))
	for _, head := range heads {
		revision, err := u.sign(ctx, own, head)
		if err != nil {
			return err
		}
		revisions = append(revisions, revision)
	}
	for len(revisions) > 0 {
		response, err := u.deps.Revisions.CommitRevisions(ctx, remote, revisions)
		if err != nil {
			return err
		}
		if response.GetCommitted() {
			return nil
		}
		if len(response.GetConflicts()) == 0 {
			return errNoConflict
		}
		rest := withoutConflicts(revisions, response.GetConflicts())
		if len(rest) == len(revisions) {
			return errUnknownConflict
		}
		revisions = rest
	}
	return nil
}

func (u *Endorse) sign(ctx context.Context, own []byte, head *syncv1.StoredRevision) (*syncv1.Revision, error) {
	old := head.GetRevision()
	revision := &syncv1.Revision{
		VaultId: old.GetVaultId(), FileId: old.GetFileId(), ParentRev: head.GetRevId(), BlobId: old.GetBlobId(), BlobHash: old.GetBlobHash(),
		BlobSize: old.GetBlobSize(), EncMeta: old.GetEncMeta(), PathHash: old.GetPathHash(), ShareRootId: old.GetShareRootId(),
		Deleted: old.GetDeleted(), Epoch: old.GetEpoch(), AuthorDeviceSignKey: own, FileVersion: old.GetFileVersion() + 1,
	}
	signature, err := u.deps.Identity.SignAsDevice(ctx, syncprotocol.GetRevisionMessage(revision))
	if err != nil {
		return nil, err
	}
	revision.Signature = signature
	return revision, nil
}

func withoutConflicts(revisions []*syncv1.Revision, conflicts []*syncv1.Conflict) []*syncv1.Revision {
	var rest []*syncv1.Revision
	for _, revision := range revisions {
		isConflict := false
		for _, conflict := range conflicts {
			isConflict = isConflict || bytes.Equal(conflict.GetFileId(), revision.GetFileId())
		}
		if !isConflict {
			rest = append(rest, revision)
		}
	}
	return rest
}

func decodeRemote(link vaultlink.Link) ([]byte, error) {
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return nil, fmt.Errorf("read the remote id of the link: %w", err)
	}
	return remote, nil
}
