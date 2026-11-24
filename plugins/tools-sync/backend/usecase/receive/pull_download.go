package receive

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
)

func (r *pullRun) download(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, stat domain.FileStat, isAbsent bool) error {
	rev := stored.GetRevision()
	plain, err := r.fetch(ctx, rev, meta)
	if limited, isLimited := errors.AsType[syncprotocol.RateLimitedError](err); isLimited {
		r.hold(stored.GetSeq())
		return fmt.Errorf("%w: retry after %s", errRateLimited, limited.RetryAfter)
	}
	if err != nil {
		return err
	}
	previous, isKnown := r.getRecord(rev.GetFileId())
	r.setRecord(newPulledRecord(rev.GetFileId(), meta.Path, meta.Hash, stored.GetRevId()))
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

func refuseShorterChain(stored, served []byte) error {
	storedChain, err := identity.ParseChain(stored)
	if err != nil {
		return fmt.Errorf("read the owner chain held: %w", err)
	}
	servedChain, err := identity.ParseChain(served)
	if err != nil {
		return fmt.Errorf("read the owner chain served: %w", err)
	}
	if servedChain.Len() < storedChain.Len() && storedChain.IsExtensionOf(servedChain) {
		return fmt.Errorf("%w: the owner chain served is shorter than the one held", domainshare.ErrRolledBack)
	}
	return nil
}
