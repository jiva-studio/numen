package pull

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var errBlobMissing = errors.New("the server has no blob for a revision")

type run struct {
	deps    Deps
	in      transfer.PullInput
	shares  []share.Root
	files   ports.Files
	remote  []byte
	key     [32]byte
	link    vaultlink.Link
	records []note.FileRecord
	result  transfer.PullResult
	holdSeq uint64
	isHeld  bool
}

func (r *run) apply(ctx context.Context, stored *syncv1.StoredRevision) error {
	meta, isVerified := r.verify(stored)
	if !isVerified {
		return nil
	}
	local, stat, err := r.readLocal(ctx, meta.Path)
	if err != nil {
		return err
	}
	rev := stored.GetRevision()
	record, _ := r.getRecord(rev.GetFileId())
	switch note.Decide(local, record.BaseHash, meta.Hash) {
	case note.Adopt:
		r.setRecord(note.FileRecord{FileID: rev.GetFileId(), Path: meta.Path, BaseHash: meta.Hash, HeadRev: stored.GetRevId()})
		return nil
	case note.Download:
		return r.download(ctx, stored, meta, stat, local == "")
	case note.Conflict:
	}
	r.addConflict(meta.Path, stored.GetSeq())
	return nil
}

func (r *run) addConflict(path string, seq uint64) {
	if !slices.Contains(r.result.Conflicts, path) {
		r.result.Conflicts = append(r.result.Conflicts, path)
	}
	r.hold(seq)
}

func (r *run) verify(stored *syncv1.StoredRevision) (note.Meta, bool) {
	rev := stored.GetRevision()
	if rev.GetDeleted() {
		return note.Meta{}, false
	}
	meta, err := r.check(stored)
	if errors.Is(err, errUnknownShare) {
		r.hold(stored.GetSeq())
	}
	if err != nil {
		r.result.Unverified++
		return note.Meta{}, false
	}
	return meta, true
}

var (
	errUnverified   = errors.New("the revision does not verify")
	errUnknownShare = errors.New("the revision belongs to a share this device does not know")
)

func (r *run) openMeta(rev *syncv1.Revision) (note.Meta, error) {
	if len(rev.GetShareRootId()) == 0 {
		return note.OpenMeta(r.key, rev.GetVaultId(), rev.GetFileId(), rev.GetEncMeta())
	}
	root, isFound := share.FindRootByID(r.shares, rev.GetShareRootId())
	if !isFound {
		return note.Meta{}, errUnknownShare
	}
	key, isKnown := root.Keys.Get(rev.GetEpoch())
	if !isKnown {
		return note.Meta{}, errUnverified
	}
	meta, err := share.OpenMeta[note.Meta](key, root.ID, rev.GetFileId(), rev.GetEpoch(), rev.GetEncMeta())
	if err != nil {
		return note.Meta{}, err
	}
	meta.Path = path.Join(root.Folder, meta.Path)
	if _, isUnder := share.GetRelativePath(root.Folder, meta.Path); !isUnder {
		return note.Meta{}, errUnverified
	}
	return meta, nil
}

func (r *run) check(stored *syncv1.StoredRevision) (note.Meta, error) {
	rev := stored.GetRevision()
	id := syncprotocol.GetRevisionID(rev)
	if syncprotocol.CheckRevision(rev) != nil || !bytes.Equal(stored.GetRevId(), id[:]) || len(rev.GetSignature()) != 64 ||
		!identity.VerifyForPlugin(rev.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessage(rev), [64]byte(rev.GetSignature())) ||
		!r.in.Authors(rev.GetAuthorDeviceSignKey()) {
		return note.Meta{}, errUnverified
	}
	meta, err := r.openMeta(rev)
	if errors.Is(err, errUnknownShare) {
		return note.Meta{}, err
	}
	if err != nil || len(meta.Key) != 32 {
		return note.Meta{}, errUnverified
	}
	if meta.Path, err = note.NormalizePath(meta.Path); err != nil {
		return note.Meta{}, errUnverified
	}
	return meta, nil
}

func (r *run) readLocal(ctx context.Context, path string) (string, domain.FileStat, error) {
	content, stat, err := r.files.Read(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return "", domain.FileStat{}, nil
	}
	if err != nil {
		return "", domain.FileStat{}, err
	}
	return note.Hash(content), stat, nil
}

func (r *run) getRecord(fileID []byte) (note.FileRecord, bool) {
	at := slices.IndexFunc(r.records, func(one note.FileRecord) bool { return bytes.Equal(one.FileID, fileID) })
	if at < 0 {
		return note.FileRecord{FileID: nil, Path: "", BaseHash: "", HeadRev: nil}, false
	}
	return r.records[at], true
}

func (r *run) setRecord(record note.FileRecord) {
	at := slices.IndexFunc(r.records, func(one note.FileRecord) bool { return bytes.Equal(one.FileID, record.FileID) })
	if at < 0 {
		r.records = append(r.records, record)
		return
	}
	r.records[at] = record
}

func (r *run) removeRecord(fileID []byte) {
	r.records = slices.DeleteFunc(r.records, func(one note.FileRecord) bool { return bytes.Equal(one.FileID, fileID) })
}

func (r *run) download(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, stat domain.FileStat, isAbsent bool) error {
	rev := stored.GetRevision()
	plain, err := r.fetch(ctx, rev, meta)
	if err != nil {
		return err
	}
	previous, isKnown := r.getRecord(rev.GetFileId())
	r.setRecord(note.FileRecord{FileID: rev.GetFileId(), Path: meta.Path, BaseHash: meta.Hash, HeadRev: stored.GetRevId()})
	if err := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); err != nil {
		return err
	}
	err = r.write(ctx, meta.Path, plain, stat, isAbsent)
	if err == nil {
		r.result.Applied = append(r.result.Applied, meta.Path)
		return nil
	}
	if isKnown {
		r.setRecord(previous)
	} else {
		r.removeRecord(rev.GetFileId())
	}
	if saveErr := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); saveErr != nil {
		return saveErr
	}
	if errors.Is(err, host.ErrStale) || errors.Is(err, host.ErrAlreadyExists) {
		r.addConflict(meta.Path, stored.GetSeq())
		return nil
	}
	return err
}

func (r *run) write(ctx context.Context, path string, plain []byte, stat domain.FileStat, isAbsent bool) error {
	if isAbsent {
		return r.files.Create(ctx, path, plain)
	}
	_, err := r.files.Write(ctx, path, plain, stat)
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
