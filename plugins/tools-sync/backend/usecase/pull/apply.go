package pull

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// KeyRollback is the health item of a revision from the server that is older than the head this device holds.
const KeyRollback = "rollback-detected"

var errBlobMissing = errors.New("the server has no blob for a revision")

type run struct {
	deps    Deps
	in      ports.PullInput
	files   ports.Files
	moves   ports.Transfers
	remote  []byte
	keys    []vaultlink.VaultKey
	link    vaultlink.Link
	records []transfer.FileRecord
	result  restore.PullResult
	holdSeq uint64
	isHeld  bool
	next    uint64
	listed  map[string]string

	removals   []removal
	removalSeq uint64
	isListing  bool
	rolledBack []string
	jumped     []string
	failures   []error
	unsaved    []string
	retries    *Retries
}

func (r *run) applyPage(ctx context.Context, page []*syncv1.StoredRevision) error {
	if err := r.applyFolderMoves(ctx, page); err != nil {
		return err
	}
	for _, stored := range page {
		err := r.apply(ctx, stored)
		if err == nil {
			continue
		}
		if refusal, isRefused := getRefusal(ctx, err); isRefused {
			if err := checkRoom(len(r.unsaved)); err != nil {
				return err
			}
			r.skipFile(stored, refusal)
			continue
		}
		if !isBlobFailure(err) {
			return err
		}
		if err := checkRoom(len(r.failures)); err != nil {
			return err
		}
		r.hold(stored.GetSeq())
		r.failures = append(r.failures, err)
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
	if isJump(rev, r.link.GetFloor(rev.GetFileId()), record, isKnown) {
		return r.skipJump(stored, meta)
	}
	if (isKnown || !rev.GetDeleted()) && r.isRegressed(rev, record, isKnown) {
		return r.recordRollback(meta, rev)
	}
	if rev.GetDeleted() {
		return r.applyDeletion(ctx, stored, record, isKnown)
	}
	if r.isWaiting(stored) {
		r.keepUnsaved(stored, meta.Path)
		return nil
	}
	if err := r.applyContent(ctx, stored, meta, record, isKnown); err != nil {
		return err
	}
	r.retries.forget(r.getRetryKey(stored))
	return nil
}

func (r *run) applyContent(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, record transfer.FileRecord, isKnown bool) error {
	local, stat, isListed, err := r.readLocal(ctx, meta.Path)
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
		if r.isLazy(meta, isListed) {
			return r.list(ctx, listEntry{stored: stored, meta: meta, previous: record, local: local, isListed: isListed, isKnown: isKnown})
		}
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
	if errors.Is(err, errUnknownShare) || errors.Is(err, errUnknownEpoch) {
		r.hold(stored.GetSeq())
	}
	if err != nil {
		r.result.Unverified++
		return note.Meta{}, false
	}
	return meta, true
}

var errUnknownShare = errors.New("the revision belongs to a share this device does not know")

var errUnknownEpoch = errors.New("the revision is sealed with a key epoch this device has not fetched")

func (r *run) checkSharedRevision(stored *syncv1.StoredRevision) (note.Meta, error) {
	rev := stored.GetRevision()
	root, isFound := share.FindRootByID(r.in.Shares, rev.GetShareRootId())
	if !isFound {
		return note.Meta{}, errUnknownShare
	}
	if err := note.CheckRevision(stored, r.remote, r.in.IsAuthor); err != nil || rev.GetDeleted() {
		return note.Meta{}, err
	}
	key, isKnown := root.Keys.Get(rev.GetEpoch())
	if !isKnown {
		return note.Meta{}, errUnknownEpoch
	}
	meta, err := share.OpenMeta[note.Meta](key, root.ID, rev.GetFileId(), rev.GetEpoch(), rev.GetEncMeta())
	if err != nil || len(meta.Key) != 32 {
		return note.Meta{}, note.ErrUnverified
	}
	meta.Path = path.Join(root.Folder, meta.Path)
	if _, isUnder := share.GetRelativePath(root.Folder, meta.Path); !isUnder {
		return note.Meta{}, note.ErrUnverified
	}
	if meta.Path, err = note.NormalizePath(meta.Path); err != nil || note.CheckRemotePath(meta.Path) != nil {
		return note.Meta{}, note.ErrUnverified
	}
	return meta, nil
}

func (r *run) check(stored *syncv1.StoredRevision) (note.Meta, error) {
	rev := stored.GetRevision()
	if len(rev.GetShareRootId()) > 0 {
		return r.checkSharedRevision(stored)
	}
	key, isKnown := r.getKey(rev.GetEpoch())
	if !isKnown && !rev.GetDeleted() {
		return note.Meta{}, errUnknownEpoch
	}
	return note.OpenRevision(key, stored, r.remote, r.in.IsAuthor)
}

func (r *run) readLocal(ctx context.Context, path string) (hash string, stat domain.FileStat, isListed bool, err error) {
	stat, err = r.files.StatFile(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return "", domain.FileStat{}, false, nil
	}
	if errors.Is(err, host.ErrFileAbsent) {
		hash, err = r.getListedHash(ctx, path)
		return hash, domain.FileStat{}, err == nil, err
	}
	if err != nil {
		return "", domain.FileStat{}, false, err
	}
	return stat.Hash, stat, false, nil
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
		Version: rev.GetFileVersion(),
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
	if err := write(ctx, baseline); err != nil {
		return r.settleWrite(ctx, stored, meta, err)
	}
	r.setRecord(record)
	r.result.Applied = append(r.result.Applied, meta.Path)
	return r.deps.Downloads.Forget(ctx, r.in.VaultID, rev.GetFileId())
}

func (r *run) settleWrite(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, err error) error {
	settled := r.settleFailure(ctx, stored, meta, err)
	if settled != nil && errors.Is(settled, err) && ctx.Err() == nil {
		return &fileError{path: meta.Path, cause: err}
	}
	return settled
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
	return note.OpenBlob(meta, rev.GetFileId(), rev.GetBlobHash(), blob)
}

func (r *run) getKey(epoch uint32) ([32]byte, bool) {
	at := slices.IndexFunc(r.keys, func(one vaultlink.VaultKey) bool { return one.Epoch == epoch })
	if at < 0 || len(r.keys[at].Key) != 32 {
		return [32]byte{}, false
	}
	return [32]byte(r.keys[at].Key), true
}

func (r *run) isRegressed(rev *syncv1.Revision, record transfer.FileRecord, isKnown bool) bool {
	floor := r.link.GetFloor(rev.GetFileId())
	if isKnown {
		floor = max(floor, record.Version)
	}
	return rev.GetFileVersion() <= floor
}

func (r *run) recordRollback(meta note.Meta, rev *syncv1.Revision) error {
	if err := checkRoom(len(r.rolledBack)); err != nil {
		return err
	}
	text := meta.Path
	if rev.GetDeleted() {
		text = fmt.Sprintf("%x", rev.GetFileId())
	}
	r.rolledBack = append(r.rolledBack, text)
	return nil
}

func checkRoom(held int) error {
	if held >= syncprotocol.MaxCatalogEntries {
		return syncprotocol.ErrTooManyPages
	}
	return nil
}

func (r *run) reportRollbacks(ctx context.Context) error {
	if len(r.rolledBack) == 0 {
		return r.deps.Health.Clear(ctx, KeyRollback)
	}
	return r.deps.Health.Report(ctx, health.Item{
		Key: KeyRollback, Severity: health.Error, Text: "The sync server sent an older version of a file", Detail: r.rolledBack[0], Action: nil,
	})
}
