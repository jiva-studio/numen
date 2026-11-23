package pull

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var errBlobMissing = errors.New("the server has no blob for a revision")

type run struct {
	deps    Deps
	in      transfer.PullInput
	files   ports.Files
	moves   ports.Transfers
	remote  []byte
	key     [32]byte
	link    vaultlink.Link
	records []transfer.FileRecord
	result  restore.PullResult
	holdSeq uint64
	isHeld  bool
	next    uint64

	removals   []removal
	removalSeq uint64
	isListing  bool
}

func (r *run) applyPage(ctx context.Context, page []*syncv1.StoredRevision) error {
	if err := r.applyFolderMoves(ctx, page); err != nil {
		return err
	}
	for _, stored := range page {
		if err := r.apply(ctx, stored); err != nil {
			return err
		}
	}
	return nil
}

func (r *run) apply(ctx context.Context, stored *syncv1.StoredRevision) error {
	meta, isVerified := r.verify(stored)
	if !isVerified {
		return nil
	}
	rev := stored.GetRevision()
	record, isKnown := r.getRecord(rev.GetFileId())
	if isKnown && bytes.Equal(record.HeadRev, stored.GetRevId()) {
		return nil
	}
	if rev.GetDeleted() {
		return r.applyDeletion(ctx, stored, record, isKnown)
	}
	return r.applyContent(ctx, stored, meta, record, isKnown)
}

func (r *run) applyContent(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, record transfer.FileRecord, isKnown bool) error {
	local, stat, err := r.readLocal(ctx, meta.Path)
	if err != nil {
		return err
	}
	if isKnown && record.Path != meta.Path && isMetadataOnly(stored.GetRevision(), record, meta) {
		isHandled, err := r.applyMove(ctx, stored, meta, record, local, stat)
		if isHandled || err != nil {
			return err
		}
	}
	switch note.Decide(local, record.BaseHash, meta.Hash) {
	case note.Adopt:
		return r.adopt(ctx, stored, meta, record, isKnown)
	case note.Download:
		return r.download(ctx, stored, meta, stat, local == "")
	case note.Conflict:
	}
	return r.resolve(ctx, stored, meta, local, stat)
}

func (r *run) adopt(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, previous transfer.FileRecord, isKnown bool) error {
	record, err := r.makeRecord(ctx, stored, meta, previous, isKnown)
	if err != nil {
		return err
	}
	r.setRecord(record)
	return nil
}

func isMetadataOnly(rev *syncv1.Revision, record transfer.FileRecord, meta note.Meta) bool {
	return len(record.BlobID) > 0 && bytes.Equal(rev.GetBlobId(), record.BlobID) && meta.Hash == record.BaseHash
}

func (r *run) addConflict(path string, seq uint64) {
	if !slices.Contains(r.result.Conflicts, path) {
		r.result.Conflicts = append(r.result.Conflicts, path)
	}
	r.hold(seq)
}

func (r *run) verify(stored *syncv1.StoredRevision) (note.Meta, bool) {
	meta, err := r.check(stored)
	if err != nil {
		r.result.Unverified++
		return note.Meta{}, false
	}
	return meta, true
}

func (r *run) check(stored *syncv1.StoredRevision) (note.Meta, error) {
	return note.OpenRevision(r.key, stored, r.in.IsAuthor)
}

func (r *run) readLocal(ctx context.Context, path string) (string, domain.FileStat, error) {
	stat, err := r.files.StatFile(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return "", domain.FileStat{}, nil
	}
	if err != nil {
		return "", domain.FileStat{}, err
	}
	return stat.Hash, stat, nil
}

func (r *run) getRecord(fileID []byte) (transfer.FileRecord, bool) {
	at := slices.IndexFunc(r.records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, fileID) })
	if at < 0 {
		return transfer.FileRecord{}, false
	}
	return r.records[at], true
}

func (r *run) setRecord(record transfer.FileRecord) {
	at := slices.IndexFunc(r.records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, record.FileID) })
	if at < 0 {
		r.records = append(r.records, record)
		return
	}
	r.records[at] = record
}

func (r *run) removeRecord(fileID []byte) {
	r.records = slices.DeleteFunc(r.records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, fileID) })
}

func (r *run) makeRecord(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, previous transfer.FileRecord, isKnown bool) (transfer.FileRecord, error) {
	rev := stored.GetRevision()
	record := transfer.FileRecord{
		FileID: rev.GetFileId(), Path: meta.Path, BaseHash: meta.Hash, HeadRev: stored.GetRevId(), Size: meta.Size, ModifiedAt: meta.MTime,
		Kind: meta.Kind, BlobID: rev.GetBlobId(), BlobHash: rev.GetBlobHash(), BlobSize: rev.GetBlobSize(), SealedKey: nil,
	}
	if isKnown && len(previous.SealedKey) > 0 && bytes.Equal(previous.BlobID, rev.GetBlobId()) {
		record.SealedKey = previous.SealedKey
		return record, nil
	}
	sealed, err := r.deps.Crypto.SealData(ctx, note.GetRecordKeyContext(r.remote, rev.GetFileId()), meta.Key)
	if err != nil {
		return transfer.FileRecord{}, fmt.Errorf("seal the key of a revision: %w", err)
	}
	record.SealedKey = sealed
	return record, nil
}

// body is the writing of a file into the vault over the baseline, made ready before the record is moved to the new base.
// A nil baseline writes a file that is not there yet.
type body func(ctx context.Context, baseline *domain.FileStat) error

func (r *run) download(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, stat domain.FileStat, isAbsent bool) error {
	rev := stored.GetRevision()
	write, err := r.prepare(ctx, stored, meta)
	if err != nil {
		return r.settleFailure(ctx, stored, meta, err)
	}
	var baseline *domain.FileStat
	if !isAbsent {
		baseline = &stat
	}
	previous, isKnown := r.getRecord(rev.GetFileId())
	record, err := r.makeRecord(ctx, stored, meta, previous, isKnown)
	if err != nil {
		return err
	}
	r.setRecord(record)
	if err := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); err != nil {
		return err
	}
	err = write(ctx, baseline)
	if err == nil {
		r.result.Applied = append(r.result.Applied, meta.Path)
		return r.deps.Downloads.Forget(ctx, r.in.VaultID, rev.GetFileId())
	}
	if isKnown {
		r.setRecord(previous)
	} else {
		r.removeRecord(rev.GetFileId())
	}
	if saveErr := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); saveErr != nil {
		return saveErr
	}
	return r.settleFailure(ctx, stored, meta, err)
}

func (r *run) settleFailure(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, err error) error {
	rev := stored.GetRevision()
	switch {
	case errors.Is(err, host.ErrStale), errors.Is(err, host.ErrAlreadyExists):
		return r.settleRace(ctx, stored, meta)
	case errors.Is(err, host.ErrHashMismatch), errors.Is(err, transfer.ErrIntegrity):
		r.hold(stored.GetSeq())
		return errors.Join(r.deps.Downloads.Discard(ctx, r.in.VaultID, rev.GetFileId()), r.deps.Health.Report(ctx, health.Item{
			Key: KeyIntegrity, Severity: health.Error, Text: "A file from the sync server is damaged", Detail: meta.Path, Action: nil,
		}))
	}
	return err
}

func (r *run) prepare(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta) (body, error) {
	rev := stored.GetRevision()
	if meta.Kind == note.KindMarkdown && meta.Size <= note.SmallFileLimit {
		plain, err := r.fetch(ctx, rev, meta)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, baseline *domain.FileStat) error {
			return r.write(ctx, meta.Path, plain, baseline)
		}, nil
	}
	in := transfer.DownloadInput{
		Vault: r.in.VaultID, RemoteID: r.remote, FileID: rev.GetFileId(), Path: meta.Path, BlobID: rev.GetBlobId(), BlobHash: rev.GetBlobHash(),
		BlobSize: rev.GetBlobSize(), Key: [32]byte(meta.Key), Hash: meta.Hash, Size: meta.Size, HeadRev: stored.GetRevId(), Baseline: nil,
	}
	name, err := r.deps.Downloads.Execute(ctx, in)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, baseline *domain.FileStat) error {
		_, err := r.moves.PlaceFile(ctx, name, meta.Path, meta.Hash, baseline)
		return err
	}, nil
}

func (r *run) write(ctx context.Context, path string, plain []byte, baseline *domain.FileStat) error {
	if baseline == nil {
		return r.files.Create(ctx, path, plain)
	}
	_, err := r.files.Write(ctx, path, plain, *baseline)
	return err
}

func (r *run) fetch(ctx context.Context, rev *syncv1.Revision, meta note.Meta) ([]byte, error) {
	downloads, err := r.deps.Blobs.GetBlobUrls(ctx, r.remote, [][]byte{rev.GetBlobId()})
	if err != nil {
		return nil, err
	}
	if len(downloads) != 1 || !downloads[0].GetFound() {
		return nil, errBlobMissing
	}
	blob, err := r.deps.Blobs.GetBlob(ctx, downloads[0].GetUrl())
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(blob)
	if !bytes.Equal(sum[:], rev.GetBlobHash()) {
		return nil, fmt.Errorf("%w: its hash differs", ErrCorruptBlob)
	}
	plain, err := blobcipher.Open([32]byte(meta.Key), [16]byte(rev.GetFileId()), blob)
	if err != nil || note.Hash(plain) != meta.Hash {
		return nil, fmt.Errorf("%w: it does not open to the note", ErrCorruptBlob)
	}
	return plain, nil
}
