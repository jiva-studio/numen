package commit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

type batchRun struct {
	deps   Deps
	vault  transfer.Vault
	author []byte
}

// prepared is a note sealed and signed, ready to commit.
type prepared struct {
	index    int
	blob     []byte
	revision *syncv1.Revision
	record   transfer.FileRecord
}

func (r *batchRun) execute(ctx context.Context, files []transfer.File) ([]transfer.Outcome, error) {
	outcomes := make([]transfer.Outcome, len(files))
	var items []*prepared
	for i, file := range files {
		item, err := r.seal(ctx, file, i)
		if errors.Is(err, blobcipher.ErrTooLarge) {
			outcomes[i] = transfer.Outcome{Path: file.Path, Status: transfer.TooLarge, FileID: nil, RevID: nil, Head: nil, ExistingFileID: nil, Record: transfer.NewEmptyRecord()}
			continue
		}
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return outcomes, nil
	}
	if err := r.upload(ctx, items); err != nil {
		return nil, err
	}
	if err := r.commit(ctx, items, files, outcomes); err != nil {
		return nil, err
	}
	return outcomes, nil
}

func (r *batchRun) seal(ctx context.Context, file transfer.File, index int) (*prepared, error) {
	path, err := note.NormalizePath(file.Path)
	if err != nil {
		return nil, fmt.Errorf("commit %s: %w", file.Path, err)
	}
	fileID := file.FileID
	if len(fileID) == 0 {
		if fileID, err = note.NewID(r.deps.Random, 16); err != nil {
			return nil, err
		}
	}
	key, err := note.NewID(r.deps.Random, 32)
	if err != nil {
		return nil, err
	}
	kind := note.GetKind(path)
	blob, err := blobcipher.Seal(r.deps.Random, [32]byte(key), [16]byte(fileID), file.Content, kind != note.KindBinary)
	if err != nil {
		return nil, err
	}
	meta := note.Meta{Path: path, Size: int64(len(file.Content)), MTime: file.ModifiedAt, Hash: note.Hash(file.Content), Key: key, Kind: kind}
	sealed, err := note.SealMeta(r.deps.Random, r.vault.Key, r.vault.ID, fileID, meta)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(blob)
	revision := &syncv1.Revision{
		VaultId: r.vault.ID, FileId: fileID, ParentRev: file.ParentRev, BlobHash: hash[:], BlobSize: uint64(len(blob)),
		EncMeta: sealed, PathHash: note.HashPath(r.vault.Key, path), AuthorDeviceSignKey: r.author,
	}
	sealedKey, err := r.deps.Crypto.SealData(ctx, note.GetRecordKeyContext(r.vault.ID, fileID), key)
	if err != nil {
		return nil, err
	}
	record := transfer.FileRecord{
		FileID: fileID, Path: path, BaseHash: meta.Hash, HeadRev: nil, Size: meta.Size, ModifiedAt: meta.MTime, Kind: kind,
		BlobID: nil, BlobHash: hash[:], BlobSize: uint64(len(blob)), SealedKey: sealedKey,
	}
	return &prepared{index: index, blob: blob, revision: revision, record: record}, nil
}

func (r *batchRun) upload(ctx context.Context, items []*prepared) error {
	sizes := make([]uint64, len(items))
	for i, item := range items {
		sizes[i] = uint64(len(item.blob))
	}
	uploads, err := r.deps.Blobs.ReserveBlobs(ctx, r.vault.ID, sizes)
	if err != nil {
		return fmt.Errorf("reserve blobs: %w", err)
	}
	if len(uploads) != len(items) {
		return fmt.Errorf("reserve blobs: the server made %d reservations for %d blobs", len(uploads), len(items))
	}
	for i, item := range items {
		item.revision.BlobId = uploads[i].GetBlobId()
		if err := r.deps.Blobs.PutBlob(ctx, uploads[i].GetUrl(), item.blob); err != nil {
			return err
		}
		signature, err := r.deps.Identity.SignAsDevice(ctx, syncprotocol.GetRevisionMessage(item.revision))
		if err != nil {
			return err
		}
		item.revision.Signature = signature
	}
	return nil
}

func (r *batchRun) commit(ctx context.Context, items []*prepared, files []transfer.File, outcomes []transfer.Outcome) error {
	for len(items) > 0 {
		revisions := make([]*syncv1.Revision, len(items))
		for i, item := range items {
			revisions[i] = item.revision
		}
		response, err := r.deps.Revisions.CommitRevisions(ctx, r.vault.ID, revisions)
		if err != nil {
			return err
		}
		if response.GetCommitted() {
			r.recordCommitted(items, files, outcomes, response.GetRevisions())
			return nil
		}
		var rest []*prepared
		rest, err = r.recordConflicts(items, files, outcomes, response.GetConflicts())
		if err != nil {
			return err
		}
		items = rest
	}
	return nil
}

func (r *batchRun) recordCommitted(items []*prepared, files []transfer.File, outcomes []transfer.Outcome, stored []*syncv1.StoredRevision) {
	for i, item := range items {
		item.record.HeadRev, item.record.BlobID = stored[i].GetRevId(), item.revision.GetBlobId()
		outcomes[item.index] = transfer.Outcome{
			Path: files[item.index].Path, Status: transfer.Committed, FileID: item.revision.GetFileId(), RevID: stored[i].GetRevId(),
			Head: nil, ExistingFileID: nil, Record: item.record,
		}
	}
}

func (r *batchRun) recordConflicts(items []*prepared, files []transfer.File, outcomes []transfer.Outcome, conflicts []*syncv1.Conflict) ([]*prepared, error) {
	if len(conflicts) == 0 {
		return nil, errUnexpectedAnswer
	}
	var rest []*prepared
	for _, item := range items {
		conflict := findConflict(conflicts, item.revision.GetFileId())
		if conflict == nil {
			rest = append(rest, item)
			continue
		}
		status := transfer.ParentMismatch
		if conflict.GetKind() == syncv1.ConflictKind_CONFLICT_KIND_PATH_TAKEN {
			status = transfer.PathTaken
		}
		outcomes[item.index] = transfer.Outcome{
			Path: files[item.index].Path, Status: status, FileID: item.revision.GetFileId(), RevID: nil,
			Head: conflict.GetHead(), ExistingFileID: conflict.GetExistingFileId(), Record: transfer.NewEmptyRecord(),
		}
	}
	return rest, nil
}

func findConflict(conflicts []*syncv1.Conflict, fileID []byte) *syncv1.Conflict {
	for _, conflict := range conflicts {
		if bytes.Equal(conflict.GetFileId(), fileID) {
			return conflict
		}
	}
	return nil
}
