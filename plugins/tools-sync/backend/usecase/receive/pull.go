package receive

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var (
	errNotReceived = errors.New("the vault is not a received share")
	errUnverified  = errors.New("the change does not verify")
	errRateLimited = errors.New("the server limits the downloads of the share")
	errBlobMissing = errors.New("the server has no blob for a revision")
	errCorruptBlob = errors.New("the blob of a revision is damaged")
	errBadHead     = errors.New("the head of the share log kept in the link is not an entry id")
)

func statusOf(result transfer.PullResult, now time.Time) ports.Status {
	return ports.Status{LastSyncedAt: now, ConflictCount: len(result.Conflicts)}
}

// PullShare brings the files of a received share into its vault.
type PullShare struct{ base }

// NewPullShare returns the use case over the dependencies.
func NewPullShare(deps Deps) *PullShare { return &PullShare{base: base{deps: deps}} }

// Execute reads the share log after the entry it verified last, then the changes of the share, and applies them by
// the rule of three hashes: a file that differs from its base is left as it is and counted. It writes nothing to the
// server. A share the owner ended stops the pass with ErrEnded and keeps the files.
func (u *PullShare) Execute(ctx context.Context, id domain.VaultID) (transfer.PullResult, error) {
	link, isReceived, err := u.readReceived(ctx, id)
	if err != nil {
		return transfer.PullResult{}, err
	}
	if !isReceived {
		return transfer.PullResult{}, errNotReceived
	}
	if link.Received.IsEnded {
		return transfer.PullResult{}, domainshare.ErrEnded
	}
	r, err := u.prepare(ctx, id, link)
	if err != nil {
		return transfer.PullResult{}, err
	}
	err = r.execute(ctx)
	var ended syncprotocol.ShareEndedError
	if errors.As(err, &ended) {
		err = r.end(ctx)
	}
	return r.result, err
}

type pullRun struct {
	deps    Deps
	id      domain.VaultID
	files   ports.Files
	shareID []byte
	reader  []byte
	link    vaultlink.Link
	keys    domainshare.Keys
	authors trust.Authors
	records []note.FileRecord
	result  transfer.PullResult
	holdSeq uint64
	isHeld  bool
}

func (u *PullShare) prepare(ctx context.Context, id domain.VaultID, link vaultlink.Link) (*pullRun, error) {
	shareID, err := hex.DecodeString(link.Received.ShareRootID)
	if err != nil {
		return nil, fmt.Errorf("read the share id of the link: %w", err)
	}
	plain, err := u.deps.Crypto.OpenData(ctx, vaultlink.GetStateKeyContext(shareID), link.Received.SealedKeys)
	if err != nil {
		return nil, fmt.Errorf("open the keys of the share: %w", err)
	}
	keys, err := domainshare.ParseKeys(plain)
	if err != nil {
		return nil, err
	}
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	chain, err := identity.ParseChain(own.Chain)
	if err != nil {
		return nil, err
	}
	records, err := u.deps.Records.ReadRecords(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pullRun{
		deps: u.deps, id: id, files: u.deps.Host.Files(id), shareID: shareID, reader: chain.Root().Sign, link: link, keys: keys,
		authors: trust.Authors{}, records: records, result: transfer.PullResult{Applied: nil, Conflicts: nil, Unverified: 0}, holdSeq: 0, isHeld: false,
	}, nil
}

func (r *pullRun) execute(ctx context.Context) error {
	if err := r.followLog(ctx); err != nil {
		return err
	}
	since := r.link.ServerSeq
	for {
		page, err := r.deps.Shares.GetShareChanges(ctx, r.shareID, since, 0)
		if err != nil {
			return err
		}
		isStopped, err := r.applyPage(ctx, page)
		if err != nil {
			return err
		}
		if err := r.save(ctx, page.GetNextSeq()); err != nil {
			return err
		}
		if isStopped || !page.GetHasMore() {
			return nil
		}
		since = page.GetNextSeq()
	}
}

func (r *pullRun) applyPage(ctx context.Context, page *syncv1.GetShareChangesResponse) (bool, error) {
	for _, change := range page.GetChanges() {
		err := r.apply(ctx, change)
		if errors.Is(err, errRateLimited) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

// followLog verifies the entries of the share log after the last one verified and keeps the keys they bring.
func (r *pullRun) followLog(ctx context.Context) error {
	received := r.link.Received
	headID, err := toEntryID(received.HeadID)
	if err != nil {
		return err
	}
	entries, chain, err := r.deps.Shares.GetShareLog(ctx, r.shareID, received.ServerVersion-1)
	if err != nil {
		return err
	}
	if entries, err = domainshare.ContinueFrom(entries, received.ServerVersion, headID); err != nil {
		return err
	}
	if r.authors, err = trust.NewAuthors(received.OwnerChain, chain); err != nil {
		return err
	}
	rules := domainshare.Rules{ShareRootID: r.shareID, Reader: r.reader, Authors: r.authors, Open: r.deps.Box.OpenBox}
	follower := domainshare.NewFollower(rules, domainshare.Position{
		VaultID: received.VaultID, Version: received.ServerVersion, HeadID: headID, Epoch: received.Epoch, Keys: r.keys, Name: received.Name,
		IsActive: true, IsEnded: false,
	})
	if err := follower.Follow(ctx, entries); err != nil {
		return err
	}
	position := follower.GetPosition()
	if len(position.Keys) != len(r.keys) {
		if err := r.sealKeys(ctx, position.Keys); err != nil {
			return err
		}
	}
	r.keys = position.Keys
	received.ServerVersion, received.HeadID, received.Epoch, received.Name = position.Version, position.HeadID[:], position.Epoch, position.Name
	received.OwnerChain, received.IsEnded = chain, position.IsEnded
	if err := r.deps.Links.WriteLink(ctx, r.id, r.link); err != nil {
		return err
	}
	if position.IsEnded {
		return domainshare.ErrEnded
	}
	return nil
}

func (r *pullRun) sealKeys(ctx context.Context, keys domainshare.Keys) error {
	plain, err := keys.Encode()
	if err != nil {
		return err
	}
	sealed, err := r.deps.Crypto.SealData(ctx, vaultlink.GetStateKeyContext(r.shareID), plain)
	if err != nil {
		return fmt.Errorf("seal the keys of the share: %w", err)
	}
	r.link.Received.SealedKeys = sealed
	return nil
}

// end marks the share ended in the link and returns ErrEnded.
func (r *pullRun) end(ctx context.Context) error {
	r.link.Received.IsEnded = true
	if err := r.deps.Links.WriteLink(ctx, r.id, r.link); err != nil {
		return err
	}
	return domainshare.ErrEnded
}

func toEntryID(raw []byte) ([32]byte, error) {
	if len(raw) != len([32]byte{}) {
		return [32]byte{}, errBadHead
	}
	return [32]byte(raw), nil
}

func (r *pullRun) save(ctx context.Context, next uint64) error {
	if err := r.deps.Records.WriteRecords(ctx, r.id, r.records); err != nil {
		return err
	}
	r.link.ServerSeq = next
	if r.isHeld {
		r.link.ServerSeq = r.holdSeq
	}
	return r.deps.Links.WriteLink(ctx, r.id, r.link)
}

func (r *pullRun) hold(seq uint64) {
	if !r.isHeld || seq-1 < r.holdSeq {
		r.holdSeq, r.isHeld = seq-1, true
	}
}

func (r *pullRun) addConflict(path string, seq uint64) {
	if !slices.Contains(r.result.Conflicts, path) {
		r.result.Conflicts = append(r.result.Conflicts, path)
	}
	r.hold(seq)
}

func (r *pullRun) apply(ctx context.Context, change *syncv1.ShareChange) error {
	if err := r.verify(change); err != nil {
		if errors.Is(err, domainshare.ErrRolledBack) {
			return err
		}
		r.result.Unverified++
		return nil
	}
	stored := change.GetChange()
	rev := stored.GetRevision()
	if r.link.Received.IsRegressed(rev.GetFileId(), stored.GetRevId()) {
		return fmt.Errorf("%w: a file is served at a revision older than the one this reader has seen", domainshare.ErrRolledBack)
	}
	r.link.Received.NoteRevision(rev.GetFileId(), stored.GetRevId())
	if change.GetLeft() || rev.GetDeleted() {
		return r.remove(ctx, rev.GetFileId(), stored.GetSeq())
	}
	return r.place(ctx, stored)
}

// verify checks that the change is what the owner signed: the id and the signature of the revision, with the hash of
// the metadata the owner signed, and that it belongs to this share.
func (r *pullRun) verify(change *syncv1.ShareChange) error {
	if err := syncprotocol.CheckShareRevision(change); err != nil {
		return err
	}
	metaHash, err := getMetaHash(change)
	if err != nil {
		return err
	}
	if !r.isSigned(change.GetChange(), metaHash) {
		return errUnverified
	}
	return r.checkBelonging(change)
}

func getMetaHash(change *syncv1.ShareChange) ([32]byte, error) {
	if len(change.GetEncMetaHash()) != len([32]byte{}) {
		return [32]byte{}, errUnverified
	}
	metaHash := [32]byte(change.GetEncMetaHash())
	if !change.GetLeft() && domainshare.GetMetaHash(change.GetChange().GetRevision().GetEncMeta()) != metaHash {
		return [32]byte{}, errUnverified
	}
	return metaHash, nil
}

func (r *pullRun) isSigned(stored *syncv1.StoredRevision, metaHash [32]byte) bool {
	rev := stored.GetRevision()
	id := syncprotocol.GetRevisionIDWithMetaHash(rev, metaHash)
	author := rev.GetAuthorDeviceSignKey()
	return len(rev.GetSignature()) == 64 && bytes.Equal(stored.GetRevId(), id[:]) && r.authors.IsTrusted(author) &&
		bytes.Equal(rev.GetVaultId(), r.link.Received.VaultID) &&
		identity.VerifyForPlugin(author, syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessageWithMetaHash(rev, metaHash), [64]byte(rev.GetSignature()))
}

// checkBelonging requires a change that left the share not to name it, and any other change to name it with an epoch
// the reader knows.
func (r *pullRun) checkBelonging(change *syncv1.ShareChange) error {
	rev := change.GetChange().GetRevision()
	isOfShare := bytes.Equal(rev.GetShareRootId(), r.shareID)
	if change.GetLeft() == isOfShare {
		return errUnverified
	}
	if isOfShare && rev.GetEpoch() > r.link.Received.Epoch {
		return fmt.Errorf("%w: a revision has epoch %d and the share log held ends at epoch %d", domainshare.ErrRolledBack, rev.GetEpoch(), r.link.Received.Epoch)
	}
	return nil
}

func (r *pullRun) openMeta(rev *syncv1.Revision) (note.Meta, error) {
	key, isKnown := r.keys.Get(rev.GetEpoch())
	if !isKnown {
		return note.Meta{}, errUnverified
	}
	meta, err := domainshare.OpenMeta(key, r.shareID, rev.GetFileId(), rev.GetEpoch(), rev.GetEncMeta())
	if err != nil || len(meta.Key) != len([32]byte{}) {
		return note.Meta{}, errUnverified
	}
	if meta.Path, err = note.NormalizePath(meta.Path); err != nil {
		return note.Meta{}, errUnverified
	}
	return meta, nil
}

func (r *pullRun) place(ctx context.Context, stored *syncv1.StoredRevision) error {
	rev := stored.GetRevision()
	meta, err := r.openMeta(rev)
	if err != nil {
		r.result.Unverified++
		return nil
	}
	local, stat, err := r.readLocal(ctx, meta.Path)
	if err != nil {
		return err
	}
	record, isKnown := r.getRecord(rev.GetFileId())
	switch note.Decide(local, record.BaseHash, meta.Hash) {
	case note.Adopt:
		r.setRecord(note.FileRecord{FileID: rev.GetFileId(), Path: meta.Path, BaseHash: meta.Hash, HeadRev: stored.GetRevId()})
		return r.retire(ctx, record, isKnown, meta.Path, stored.GetSeq())
	case note.Download:
		return r.download(ctx, stored, meta, stat, local == "")
	case note.Conflict:
	}
	r.addConflict(meta.Path, stored.GetSeq())
	return nil
}

// retire removes the file a note had before it was moved inside the share, when the reader did not change it.
func (r *pullRun) retire(ctx context.Context, previous note.FileRecord, isKnown bool, path string, seq uint64) error {
	if !isKnown || previous.Path == path {
		return nil
	}
	local, _, err := r.readLocal(ctx, previous.Path)
	if err != nil || local == "" {
		return err
	}
	if local != previous.BaseHash {
		r.addConflict(previous.Path, seq)
		return nil
	}
	return r.trash(ctx, previous.Path)
}

func (r *pullRun) remove(ctx context.Context, fileID []byte, seq uint64) error {
	record, isKnown := r.getRecord(fileID)
	if !isKnown {
		return nil
	}
	local, _, err := r.readLocal(ctx, record.Path)
	if err != nil {
		return err
	}
	if local != "" && local != record.BaseHash {
		r.addConflict(record.Path, seq)
		return nil
	}
	if local != "" {
		if err := r.trash(ctx, record.Path); err != nil {
			return err
		}
	}
	r.removeRecord(fileID)
	return nil
}

func (r *pullRun) trash(ctx context.Context, path string) error {
	removals, err := r.deps.Host.Remover(r.id).RemoveNotes(ctx, []string{path})
	if err != nil {
		return err
	}
	for _, removal := range removals {
		if removal.Err != nil && !errors.Is(removal.Err, host.ErrNotExist) {
			return removal.Err
		}
	}
	return nil
}

func (r *pullRun) readLocal(ctx context.Context, path string) (string, domain.FileStat, error) {
	content, stat, err := r.files.Read(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return "", domain.FileStat{}, nil
	}
	if err != nil {
		return "", domain.FileStat{}, err
	}
	return note.Hash(content), stat, nil
}

func (r *pullRun) getRecord(fileID []byte) (note.FileRecord, bool) {
	at := slices.IndexFunc(r.records, func(one note.FileRecord) bool { return bytes.Equal(one.FileID, fileID) })
	if at < 0 {
		return note.FileRecord{FileID: nil, Path: "", BaseHash: "", HeadRev: nil}, false
	}
	return r.records[at], true
}

func (r *pullRun) setRecord(record note.FileRecord) {
	at := slices.IndexFunc(r.records, func(one note.FileRecord) bool { return bytes.Equal(one.FileID, record.FileID) })
	if at < 0 {
		r.records = append(r.records, record)
		return
	}
	r.records[at] = record
}

func (r *pullRun) removeRecord(fileID []byte) {
	r.records = slices.DeleteFunc(r.records, func(one note.FileRecord) bool { return bytes.Equal(one.FileID, fileID) })
}

func (r *pullRun) download(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, stat domain.FileStat, isAbsent bool) error {
	rev := stored.GetRevision()
	plain, err := r.fetch(ctx, rev, meta)
	var limited syncprotocol.RateLimitedError
	if errors.As(err, &limited) {
		r.hold(stored.GetSeq())
		return errRateLimited
	}
	if err != nil {
		return err
	}
	previous, isKnown := r.getRecord(rev.GetFileId())
	r.setRecord(note.FileRecord{FileID: rev.GetFileId(), Path: meta.Path, BaseHash: meta.Hash, HeadRev: stored.GetRevId()})
	if err := r.deps.Records.WriteRecords(ctx, r.id, r.records); err != nil {
		return err
	}
	err = r.write(ctx, meta.Path, plain, stat, isAbsent)
	if err == nil {
		r.result.Applied = append(r.result.Applied, meta.Path)
		return r.retire(ctx, previous, isKnown, meta.Path, stored.GetSeq())
	}
	if isKnown {
		r.setRecord(previous)
	} else {
		r.removeRecord(rev.GetFileId())
	}
	if saveErr := r.deps.Records.WriteRecords(ctx, r.id, r.records); saveErr != nil {
		return saveErr
	}
	if errors.Is(err, host.ErrStale) || errors.Is(err, host.ErrAlreadyExists) {
		r.addConflict(meta.Path, stored.GetSeq())
		return nil
	}
	return err
}

func (r *pullRun) write(ctx context.Context, path string, plain []byte, stat domain.FileStat, isAbsent bool) error {
	if isAbsent {
		return r.files.Create(ctx, path, plain)
	}
	_, err := r.files.Write(ctx, path, plain, stat)
	return err
}

func (r *pullRun) fetch(ctx context.Context, rev *syncv1.Revision, meta note.Meta) ([]byte, error) {
	downloads, err := r.deps.Shares.GetShareBlobUrls(ctx, r.shareID, [][]byte{rev.GetBlobId()})
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
		return nil, fmt.Errorf("%w: its hash differs", errCorruptBlob)
	}
	plain, err := blobcipher.Open([32]byte(meta.Key), [16]byte(rev.GetFileId()), blob)
	if err != nil || note.Hash(plain) != meta.Hash {
		return nil, fmt.Errorf("%w: it does not open to the note", errCorruptBlob)
	}
	return plain, nil
}
