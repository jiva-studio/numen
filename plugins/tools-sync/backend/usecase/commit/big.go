package commit

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

func (r *batchRun) executeBig(ctx context.Context, vaultID domain.VaultID, file transfer.File) (transfer.Outcome, error) {
	path, err := note.NormalizePath(file.Path)
	if err != nil {
		return transfer.Outcome{}, fmt.Errorf("commit %s: %w", file.Path, err)
	}
	fileID := file.FileID
	if len(fileID) == 0 {
		if fileID, err = note.NewID(r.deps.Random, 16); err != nil {
			return transfer.Outcome{}, err
		}
	}
	sent, err := r.deps.Uploads.Execute(ctx, transfer.UploadInput{
		Vault: vaultID, RemoteID: r.vault.ID, Path: path, Stat: *file.Stat, FileID: fileID, ParentRev: file.ParentRev,
	})
	if errors.Is(err, blobcipher.ErrTooLarge) {
		return transfer.Outcome{
			Path: file.Path, Status: transfer.TooLarge, FileID: nil, RevID: nil, Head: nil, ExistingFileID: nil, Record: transfer.NewEmptyRecord(),
		}, nil
	}
	if err != nil {
		return transfer.Outcome{}, err
	}
	kind := note.GetKind(path)
	meta := note.Meta{Path: path, Size: file.Stat.Size, MTime: file.ModifiedAt, Hash: file.Stat.Hash, Key: sent.Key[:], Kind: kind}
	sealed, err := note.SealMeta(r.deps.Random, r.vault.Key, r.vault.ID, fileID, meta)
	if err != nil {
		return transfer.Outcome{}, err
	}
	revision := &syncv1.Revision{
		VaultId: r.vault.ID, FileId: fileID, ParentRev: file.ParentRev, BlobId: sent.BlobID, BlobHash: sent.BlobHash[:], BlobSize: sent.BlobSize,
		EncMeta: sealed, PathHash: note.HashPath(r.vault.Key, path), AuthorDeviceSignKey: r.author,
	}
	if revision.Signature, err = r.deps.Identity.SignAsDevice(ctx, syncprotocol.GetRevisionMessage(revision)); err != nil {
		return transfer.Outcome{}, err
	}
	sealedKey, err := r.deps.Crypto.SealData(ctx, note.GetRecordKeyContext(r.vault.ID, fileID), sent.Key[:])
	if err != nil {
		return transfer.Outcome{}, err
	}
	done, err := commitBatch(ctx, r.deps, r.vault, []*syncv1.Revision{revision})
	if err != nil {
		return transfer.Outcome{}, err
	}
	outcome := done[0]
	outcome.Path = file.Path
	outcome = r.settleClash(outcome)
	if outcome.Status == transfer.Committed {
		outcome.Record = transfer.FileRecord{
			FileID: fileID, Path: path, BaseHash: file.Stat.Hash, HeadRev: outcome.RevID, Size: file.Stat.Size, ModifiedAt: file.ModifiedAt, Kind: kind,
			BlobID: sent.BlobID, BlobHash: sent.BlobHash[:], BlobSize: sent.BlobSize, SealedKey: sealedKey,
		}
		return outcome, r.deps.Uploads.Forget(ctx, vaultID, fileID)
	}
	return outcome, nil
}
