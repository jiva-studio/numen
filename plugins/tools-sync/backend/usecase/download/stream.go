package download

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const lengthPrefix = 8

// position is where a download stands: the chunks taken and the hash of the stored bytes of them. state is the
// marshaled hash after next chunks, so a break can keep the position of the last whole chunk.
type position struct {
	header [56]byte
	next   uint64
	hasher ports.Digest
	state  []byte
}

// counted counts the bytes a body gave and notes that it ended, to tell a body that ended early from a body that
// does not open.
type counted struct {
	source  io.Reader
	count   int64
	want    int64
	isEnded bool
}

func (c *counted) Read(p []byte) (int, error) {
	n, err := c.source.Read(p)
	c.count += int64(n)
	c.isEnded = c.isEnded || err != nil
	return n, err
}

func (r *run) stageStream(ctx context.Context) error {
	at, err := r.findPosition(ctx)
	if err != nil {
		return err
	}
	offset := r.getOffset(at)
	body, err := r.deps.Blobs.OpenBlob(ctx, r.in.RemoteID, r.in.BlobID, offset)
	if err != nil {
		return err
	}
	err = r.stageBody(ctx, &counted{source: body, count: 0, want: int64(r.in.BlobSize) - offset, isEnded: false}, at)
	return errors.Join(err, body.Close())
}

func (r *run) getOffset(at *position) int64 {
	if at.next == 0 {
		return 0
	}
	return blobcipher.GetChunkOffset(getChunkLog2(at.header), at.next)
}

func getChunkLog2(header [56]byte) byte {
	parsed, err := blobcipher.ParseHeader(header[:])
	if err != nil {
		return 0
	}
	return parsed.ChunkLog2
}

func (r *run) findPosition(ctx context.Context) (*position, error) {
	fresh := &position{header: [56]byte{}, next: 0, hasher: r.deps.Digests.Create(), state: nil}
	entries, err := r.deps.Entries.ReadDownloads(ctx, r.in.Vault)
	if err != nil {
		return nil, err
	}
	at := slices.IndexFunc(entries, func(e transfer.DownloadEntry) bool { return bytes.Equal(e.FileID, r.in.FileID) })
	if at < 0 {
		return fresh, r.store.Delete(ctx, r.name)
	}
	restored, isUsable, err := r.restore(ctx, entries[at])
	if err != nil {
		return nil, err
	}
	if isUsable {
		return restored, nil
	}
	return fresh, errors.Join(r.store.Delete(ctx, r.name), r.deps.Entries.WriteDownloads(ctx, r.in.Vault, withoutFile(entries, r.in.FileID)))
}

func (r *run) restore(ctx context.Context, kept transfer.DownloadEntry) (*position, bool, error) {
	if !bytes.Equal(kept.BlobID, r.in.BlobID) || kept.MetaHash != r.in.Hash || len(kept.Header) != 56 || kept.NextChunk == 0 {
		return nil, false, nil
	}
	files, err := r.store.List(ctx)
	if err != nil {
		return nil, false, err
	}
	staged := slices.IndexFunc(files, func(f host.StoreFile) bool { return f.Name == kept.StagedName })
	wantSize := int64(kept.NextChunk)<<getChunkLog2([56]byte(kept.Header)) - lengthPrefix
	if staged < 0 || files[staged].Size != wantSize {
		return nil, false, nil
	}
	hasher := r.deps.Digests.Create()
	restored := hasher.Restore(kept.StoredHashState) == nil
	if !restored {
		return nil, false, nil
	}
	return &position{header: [56]byte(kept.Header), next: kept.NextChunk, hasher: hasher, state: kept.StoredHashState}, true, nil
}

func (r *run) stageBody(ctx context.Context, body *counted, at *position) error {
	var plain *blobcipher.PlainReader
	var err error
	if at.next == 0 {
		plain, err = blobcipher.OpenStream(r.in.Key, [16]byte(r.in.FileID), body)
	} else {
		plain, err = blobcipher.OpenStreamFrom(r.in.Key, [16]byte(r.in.FileID), at.header, at.next, r.in.Size, body)
	}
	if err != nil {
		return r.getReadFailure(err, body)
	}
	if at.next == 0 {
		at.header = plain.Header()
		if _, err := at.hasher.Write(at.header[:]); err != nil {
			return fmt.Errorf("hash the stored bytes: %w", err)
		}
	}
	plain.SetStoredSink(at.hasher)
	return r.copyChunks(ctx, plain, at, body)
}

func (r *run) copyChunks(ctx context.Context, plain *blobcipher.PlainReader, at *position, body *counted) error {
	buffer := make([]byte, 1<<20+lengthPrefix)
	total := int64(0)
	if at.next > 0 {
		total = int64(at.next)<<getChunkLog2(at.header) - lengthPrefix
	}
	for {
		n, err := plain.Read(buffer)
		if n > 0 {
			if appendErr := r.store.Append(ctx, r.name, buffer[:n]); appendErr != nil {
				return appendErr
			}
			total += int64(n)
			if advanceErr := r.advance(ctx, plain, at); advanceErr != nil {
				return advanceErr
			}
		}
		if errors.Is(err, io.EOF) {
			return r.finish(at, total)
		}
		if err != nil {
			return r.settleReadFailure(ctx, err, at, body)
		}
	}
}

func (r *run) settleReadFailure(ctx context.Context, err error, at *position, body *counted) error {
	failure := r.getReadFailure(err, body)
	if errors.Is(failure, ErrIntegrity) {
		return failure
	}
	return errors.Join(failure, r.keepStarted(ctx, at))
}

func (r *run) getReadFailure(err error, body *counted) error {
	if !errors.Is(err, blobcipher.ErrCannotOpen) {
		return err
	}
	if body.isEnded && body.count < body.want {
		return fmt.Errorf("the sync server stopped sending the file: %w", io.ErrUnexpectedEOF)
	}
	return fmt.Errorf("%w: %w", ErrIntegrity, err)
}

func (r *run) finish(at *position, total int64) error {
	if !bytes.Equal(at.hasher.Sum(), r.in.BlobHash) || total != r.in.Size {
		return ErrIntegrity
	}
	return nil
}

func (r *run) advance(ctx context.Context, plain *blobcipher.PlainReader, at *position) error {
	if !plain.IsAtChunkBoundary() {
		return nil
	}
	state, err := at.hasher.Save()
	if err != nil {
		return fmt.Errorf("keep the hash of the stored bytes: %w", err)
	}
	at.next, at.state = plain.ChunkIndex(), state
	if at.next%saveEvery != 0 {
		return nil
	}
	return r.keep(ctx, at)
}

func (r *run) keepStarted(ctx context.Context, at *position) error {
	if at.next == 0 {
		return nil
	}
	return r.keep(ctx, at)
}

func (r *run) keep(ctx context.Context, at *position) error {
	entries, err := r.deps.Entries.ReadDownloads(ctx, r.in.Vault)
	if err != nil {
		return err
	}
	entry := transfer.DownloadEntry{
		FileID: r.in.FileID, Path: r.in.Path, BlobID: r.in.BlobID, Header: at.header[:], NextChunk: at.next,
		StoredHashState: at.state, StagedName: r.name, MetaHash: r.in.Hash, HeadRev: r.in.HeadRev,
	}
	return r.deps.Entries.WriteDownloads(ctx, r.in.Vault, append(withoutFile(entries, r.in.FileID), entry))
}
