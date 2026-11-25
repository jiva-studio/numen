package restore

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

var errEpochMoved = errors.New("the sync server announced another restore while its notes were read")

// heads is the heads of the files of a vault as the server holds them, and the epoch id it answered with.
type heads struct {
	byFile  map[string]domain.Head
	blobs   map[string][]byte
	epochID []byte
}

func (u *RecoverVault) readHeads(ctx context.Context, vault transfer.Vault) (heads, error) {
	remote := vault.ID
	found := heads{byFile: map[string]domain.Head{}, blobs: map[string][]byte{}, epochID: nil}
	var since uint64
	for pages, isFirst := 1, true; ; pages, isFirst = pages+1, false {
		page, err := u.deps.Revisions.GetChanges(ctx, remote, since, 0)
		if err != nil {
			return heads{}, err
		}
		if isFirst {
			found.epochID = page.GetRestoreEpochId()
		} else if !bytes.Equal(found.epochID, page.GetRestoreEpochId()) {
			return heads{}, errEpochMoved
		}
		found.add(vault, page.GetChanges())
		if err := syncprotocol.CheckPage(pages, since, page.GetNextSeq(), page.GetHasMore()); err != nil {
			return heads{}, err
		}
		since = page.GetNextSeq()
		if !page.GetHasMore() {
			return found, nil
		}
	}
}

func (h heads) add(vault transfer.Vault, changes []*syncv1.StoredRevision) {
	for _, stored := range changes {
		rev := stored.GetRevision()
		head := domain.Head{
			FileID: rev.GetFileId(), Rev: stored.GetRevId(), ParentRev: rev.GetParentRev(), Version: rev.GetFileVersion(), Hash: "", Path: "",
			IsDeleted: rev.GetDeleted(), IsBlobMissing: false,
		}
		if key, isKnown := vault.GetKey(rev.GetEpoch()); isKnown && !rev.GetDeleted() {
			if meta, err := note.OpenMeta(key, vault.ID, rev.GetFileId(), rev.GetEncMeta()); err == nil {
				head.Hash, head.Path = meta.Hash, meta.Path
			}
		}
		h.byFile[string(rev.GetFileId())] = head
		if !rev.GetDeleted() {
			h.blobs[string(rev.GetFileId())] = rev.GetBlobId()
		}
	}
}

func (u *RecoverVault) markMissingBlobs(ctx context.Context, remote []byte, found heads, records []transfer.FileRecord) error {
	var files []string
	var ids [][]byte
	for _, record := range records {
		head, isFound := found.byFile[string(record.FileID)]
		if isFound && !head.IsDeleted && bytes.Equal(head.Rev, record.HeadRev) {
			files = append(files, string(record.FileID))
			ids = append(ids, found.blobs[string(record.FileID)])
		}
	}
	for start := 0; start < len(ids); start += syncprotocol.MaxBlobURLs {
		end := min(start+syncprotocol.MaxBlobURLs, len(ids))
		missing, err := u.readMissingBlobs(ctx, remote, ids[start:end])
		if err != nil {
			return err
		}
		for i := start; i < end; i++ {
			if missing[string(ids[i])] {
				head := found.byFile[files[i]]
				head.IsBlobMissing = true
				found.byFile[files[i]] = head
			}
		}
	}
	return nil
}

func (u *RecoverVault) readMissingBlobs(ctx context.Context, remote []byte, ids [][]byte) (map[string]bool, error) {
	missing := map[string]bool{}
	downloads, err := u.deps.Blobs.GetBlobUrls(ctx, remote, ids)
	if gone, isGone := errors.AsType[syncprotocol.BlobMissingError](err); isGone {
		for _, id := range gone.IDs {
			missing[string(id)] = true
		}
		return missing, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check the bodies of the files: %w", err)
	}
	for _, download := range downloads {
		if !download.GetFound() {
			missing[string(download.GetBlobId())] = true
		}
	}
	return missing, nil
}
