package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding"
	"errors"
	"fmt"
	"hash"
	"io"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
)

const lengthPrefix = 8

// position is where a download stands: the chunks taken and the hash of the stored bytes of them.
type position struct {
	header [56]byte
	next   uint64
	hasher hash.Hash
}

// stageStream downloads a big blob chunk by chunk, appending each chunk to the staged file after it is
// authenticated, and keeps the position every few chunks.
func (r *run) stageStream(ctx context.Context) error {
	at, err := r.findPosition(ctx)
	if err != nil {
		return err
	}
	body, err := r.deps.Blobs.OpenBlob(ctx, r.in.RemoteID, r.in.BlobID, r.getOffset(at))
	if err != nil {
		return err
	}
	err = r.stageBody(ctx, body, at)
	return errors.Join(err, body.Close())
}

func (r *run) getOffset(at *position) int64 {
	if at.next == 0 {
		return 0
	}
	return blobcipher.GetChunkOffset(chunkLog2(at.header), at.next)
}

func chunkLog2(header [56]byte) byte {
	parsed, err := blobcipher.ParseHeader(header[:])
	if err != nil {
		return 0
	}
	return parsed.ChunkLog2
}

// findPosition returns the position a kept download reached when its staged file is the size it must be, and the
// start of a download else.
func (r *run) findPosition(ctx context.Context) (*position, error) {
	fresh := &position{hasher: sha256.New()}
	entries, err := r.deps.Entries.ReadDownloads(ctx, r.in.Vault)
	if err != nil {
		return nil, err
	}
	at := slices.IndexFunc(entries, func(e Entry) bool { return bytes.Equal(e.FileID, r.in.FileID) })
	if at < 0 {
		return fresh, r.store.Delete(ctx, r.name)
	}
	kept := entries[at]
	restored, isUsable, err := r.restore(ctx, kept)
	if err != nil {
		return nil, err
	}
	if isUsable {
		return restored, nil
	}
	return fresh, errors.Join(r.store.Delete(ctx, r.name), r.deps.Entries.WriteDownloads(ctx, r.in.Vault, withoutFile(entries, r.in.FileID)))
}

func (r *run) restore(ctx context.Context, kept Entry) (*position, bool, error) {
	if !bytes.Equal(kept.BlobID, r.in.BlobID) || kept.MetaHash != r.in.Hash || len(kept.Header) != 56 || kept.NextChunk == 0 {
		return nil, false, nil
	}
	files, err := r.store.List(ctx)
	if err != nil {
		return nil, false, err
	}
	staged := slices.IndexFunc(files, func(f host.StoreFile) bool { return f.Name == kept.StagedName })
	wantSize := int64(kept.NextChunk)<<chunkLog2([56]byte(kept.Header)) - lengthPrefix
	if staged < 0 || files[staged].Size != wantSize {
		return nil, false, nil
	}
	hasher := sha256.New()
	unmarshaler, ok := hasher.(encoding.BinaryUnmarshaler)
	if !ok || unmarshaler.UnmarshalBinary(kept.StoredHashState) != nil {
		return nil, false, nil
	}
	return &position{header: [56]byte(kept.Header), next: kept.NextChunk, hasher: hasher}, true, nil
}

func (r *run) stageBody(ctx context.Context, body io.Reader, at *position) error {
	var plain *blobcipher.PlainReader
	var err error
	if at.next == 0 {
		plain, err = blobcipher.OpenStream(r.in.Key, [16]byte(r.in.FileID), body)
	} else {
		plain, err = blobcipher.OpenStreamFrom(r.in.Key, [16]byte(r.in.FileID), at.header, at.next, r.in.Size, body)
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrIntegrity, err)
	}
	if at.next == 0 {
		at.header = plain.Header()
		at.hasher.Write(at.header[:])
	}
	plain.SetStoredSink(at.hasher)
	return r.copyChunks(ctx, plain, at)
}

func (r *run) copyChunks(ctx context.Context, plain *blobcipher.PlainReader, at *position) error {
	buffer := make([]byte, 1<<20+lengthPrefix)
	total := int64(0)
	if at.next > 0 {
		total = int64(at.next)<<chunkLog2(at.header) - lengthPrefix
	}
	for {
		n, err := plain.Read(buffer)
		if n > 0 {
			if appendErr := r.store.Append(ctx, r.name, buffer[:n]); appendErr != nil {
				return appendErr
			}
			total += int64(n)
		}
		if errors.Is(err, io.EOF) {
			return r.finish(at, total)
		}
		if err != nil {
			return r.readFailure(err)
		}
		if plain.IsAtChunkBoundary() && plain.ChunkIndex()%saveEvery == 0 {
			if err := r.keep(ctx, plain, at); err != nil {
				return err
			}
		}
	}
}

func (r *run) readFailure(err error) error {
	if errors.Is(err, blobcipher.ErrCannotOpen) {
		return fmt.Errorf("%w: %w", ErrIntegrity, err)
	}
	return err
}

func (r *run) finish(at *position, total int64) error {
	if !bytes.Equal(at.hasher.Sum(nil), r.in.BlobHash) || total != r.in.Size {
		return ErrIntegrity
	}
	return nil
}

// keep writes the position after the chunks taken so far.
func (r *run) keep(ctx context.Context, plain *blobcipher.PlainReader, at *position) error {
	marshaler, ok := at.hasher.(encoding.BinaryMarshaler)
	if !ok {
		return errors.New("the hash of the stored bytes cannot be kept")
	}
	state, err := marshaler.MarshalBinary()
	if err != nil {
		return fmt.Errorf("keep the hash of the stored bytes: %w", err)
	}
	entries, err := r.deps.Entries.ReadDownloads(ctx, r.in.Vault)
	if err != nil {
		return err
	}
	header := plain.Header()
	entry := Entry{
		FileID: r.in.FileID, Path: r.in.Path, BlobID: r.in.BlobID, Header: header[:], NextChunk: plain.ChunkIndex(),
		StoredHashState: state, StagedName: r.name, MetaHash: r.in.Hash, HeadRev: r.in.HeadRev,
	}
	return r.deps.Entries.WriteDownloads(ctx, r.in.Vault, append(withoutFile(entries, r.in.FileID), entry))
}
