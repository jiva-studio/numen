package upload

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
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

const keyContext = "numen-sync/upload-key/v1"

type run struct {
	deps  Deps
	in    transfer.UploadInput
	plain *plainReader
}

func (r *run) execute(ctx context.Context) (transfer.UploadResult, error) {
	source := r.deps.Host.Transfers(r.in.Vault)
	r.plain = &plainReader{
		read: func(start, length int64) (host.ByteRange, error) {
			return source.ReadBytes(ctx, r.in.Path, start, length)
		},
		stat: r.in.Stat,
		size: r.in.Stat.Size,
	}
	key, salt, err := r.newSecrets()
	if err != nil {
		return transfer.UploadResult{}, err
	}
	sealer, err := blobcipher.NewSealer(key, [16]byte(r.in.FileID), salt, r.in.Stat.Size)
	if err != nil {
		return transfer.UploadResult{}, err
	}
	if !isBig(sealer.StoredSize()) {
		return r.sendWhole(ctx, key)
	}
	return r.sendParts(ctx, key, salt)
}

func (r *run) newSecrets() (key [32]byte, salt [16]byte, err error) {
	if _, err := r.deps.Random.Read(key[:]); err != nil {
		return key, salt, fmt.Errorf("make the key of a file: %w", err)
	}
	salt, err = blobcipher.GenerateSalt(r.deps.Random)
	return key, salt, err
}

func (r *run) sendWhole(ctx context.Context, key [32]byte) (transfer.UploadResult, error) {
	whole, err := readAll(r.plain, r.in.Stat.Size)
	if err != nil {
		return transfer.UploadResult{}, err
	}
	blob, err := blobcipher.Seal(r.deps.Random, key, [16]byte(r.in.FileID), whole, isText(r.in.Path))
	if err != nil {
		return transfer.UploadResult{}, err
	}
	up, err := r.deps.Uploads.Start(ctx, r.in.RemoteID, int64(len(blob)))
	if err != nil {
		return transfer.UploadResult{}, err
	}
	if err := r.deps.Uploads.PutPart(ctx, up, 1, blob); err != nil {
		return transfer.UploadResult{}, err
	}
	if err := r.deps.Uploads.Finish(ctx, up); err != nil {
		return transfer.UploadResult{}, err
	}
	return transfer.UploadResult{BlobID: up.Blob, BlobHash: sha256.Sum256(blob), BlobSize: uint64(len(blob)), Key: key}, nil
}

// session is the upload of a big file in progress: its reservation, its key and what is sent of it.
type session struct {
	up     transfer.Upload
	entry  transfer.UploadEntry
	key    [32]byte
	sealer *blobcipher.Sealer
	hasher hash.Hash
}

func (r *run) sendParts(ctx context.Context, key [32]byte, salt [16]byte) (transfer.UploadResult, error) {
	s, err := r.open(ctx, key, salt)
	if err != nil {
		return transfer.UploadResult{}, err
	}
	result, err := r.sendRest(ctx, s)
	if errors.Is(err, ErrFileChanged) {
		return transfer.UploadResult{}, errors.Join(err, r.drop(ctx, s))
	}
	return result, err
}

// open takes up the kept upload of the file when the file is the one it began with, and starts a new one else.
func (r *run) open(ctx context.Context, key [32]byte, salt [16]byte) (*session, error) {
	entries, err := r.deps.Entries.ReadUploads(ctx, r.in.Vault)
	if err != nil {
		return nil, err
	}
	if at := slices.IndexFunc(entries, func(e transfer.UploadEntry) bool { return bytes.Equal(e.FileID, r.in.FileID) }); at >= 0 {
		kept, err := r.resume(ctx, entries[at])
		if err == nil {
			return kept, nil
		}
		if !errors.Is(err, transfer.ErrUploadGone) {
			return nil, err
		}
		entries = slices.Delete(entries, at, at+1)
	}
	return r.start(ctx, entries, key, salt)
}

func (r *run) isSame(e transfer.UploadEntry) bool {
	return e.PlainHash == r.in.Stat.Hash && e.PlainSize == r.in.Stat.Size && e.MTime == r.in.Stat.ModTime.UnixNano()
}

func (r *run) resume(ctx context.Context, e transfer.UploadEntry) (*session, error) {
	up, err := r.deps.Uploads.Resume(ctx, r.in.RemoteID, e.BlobID)
	if err != nil {
		return nil, err
	}
	up.PartSize, up.PartCount = e.PartSize, e.PartCount
	if !r.isSame(e) {
		return nil, errors.Join(transfer.ErrUploadGone, r.deps.Uploads.Abort(ctx, up), r.removeEntry(ctx, e))
	}
	opened, err := r.deps.Crypto.OpenData(ctx, r.getKeyContext(), e.SealedKey)
	if err != nil {
		return nil, fmt.Errorf("open the key of an upload: %w", err)
	}
	s := &session{up: up, entry: e, key: [32]byte(opened), sealer: nil, hasher: sha256.New()}
	if s.sealer, err = blobcipher.NewSealer(s.key, [16]byte(r.in.FileID), [16]byte(e.Salt), e.PlainSize); err != nil {
		return nil, err
	}
	return s, r.restoreHash(s)
}

func (r *run) getKeyContext() []byte {
	return slices.Concat([]byte(keyContext), r.in.RemoteID, r.in.FileID)
}

func (r *run) removeEntry(ctx context.Context, gone transfer.UploadEntry) error {
	entries, err := r.deps.Entries.ReadUploads(ctx, r.in.Vault)
	if err != nil {
		return err
	}
	return r.deps.Entries.WriteUploads(ctx, r.in.Vault, withoutFile(entries, gone.FileID))
}

func withoutFile(entries []transfer.UploadEntry, fileID []byte) []transfer.UploadEntry {
	return slices.DeleteFunc(slices.Clone(entries), func(e transfer.UploadEntry) bool { return bytes.Equal(e.FileID, fileID) })
}

func (r *run) start(ctx context.Context, entries []transfer.UploadEntry, key [32]byte, salt [16]byte) (*session, error) {
	sealer, err := blobcipher.NewSealer(key, [16]byte(r.in.FileID), salt, r.in.Stat.Size)
	if err != nil {
		return nil, err
	}
	up, err := r.deps.Uploads.Start(ctx, r.in.RemoteID, sealer.StoredSize())
	if err != nil {
		return nil, err
	}
	sealed, err := r.deps.Crypto.SealData(ctx, r.getKeyContext(), key[:])
	if err != nil {
		return nil, fmt.Errorf("seal the key of an upload: %w", err)
	}
	entry := transfer.UploadEntry{
		FileID: r.in.FileID, Path: r.in.Path, PlainSize: r.in.Stat.Size, PlainHash: r.in.Stat.Hash, MTime: r.in.Stat.ModTime.UnixNano(),
		ParentRev: r.in.ParentRev, BlobID: up.Blob, PartSize: up.PartSize, PartCount: up.PartCount, StoredSize: sealer.StoredSize(),
		Salt: salt[:], SealedKey: sealed, PartsDone: 0, StoredHashState: nil,
	}
	if err := r.deps.Entries.WriteUploads(ctx, r.in.Vault, append(entries, entry)); err != nil {
		return nil, err
	}
	return &session{up: up, entry: entry, key: key, sealer: sealer, hasher: sha256.New()}, nil
}

// restoreHash brings the hash of the stored bytes to the parts the server holds in a row from the first: the state kept
// when it covers no more of them, else a hash rebuilt from the file, which asks the network for nothing.
func (r *run) restoreHash(s *session) error {
	held := 0
	for slices.Contains(s.up.Done, held+1) {
		held++
	}
	if s.entry.PartsDone <= held {
		return r.unmarshalHash(s)
	}
	s.entry.PartsDone, s.entry.StoredHashState = 0, nil
	for number := 1; number <= held; number++ {
		if _, err := r.readPart(s, number, make([]byte, r.getPartLength(s, number))); err != nil {
			return err
		}
	}
	s.entry.PartsDone = held
	return nil
}

func (r *run) unmarshalHash(s *session) error {
	if s.entry.PartsDone == 0 {
		return nil
	}
	unmarshaler, ok := s.hasher.(encoding.BinaryUnmarshaler)
	if !ok {
		return errors.New("the hash of the stored bytes cannot be restored")
	}
	if err := unmarshaler.UnmarshalBinary(s.entry.StoredHashState); err != nil {
		return fmt.Errorf("restore the hash of the stored bytes: %w", err)
	}
	return nil
}

func (r *run) getPartLength(s *session, number int) int64 {
	return min(s.up.PartSize, s.sealer.StoredSize()-int64(number-1)*s.up.PartSize)
}

// readPart fills the buffer with the stored bytes of the part and advances the hash by them.
func (r *run) readPart(s *session, number int, buffer []byte) ([]byte, error) {
	if _, err := s.sealer.ReadStoredAt(buffer, int64(number-1)*s.up.PartSize, r.plain); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	s.hasher.Write(buffer)
	return buffer, nil
}

func (r *run) sendRest(ctx context.Context, s *session) (transfer.UploadResult, error) {
	buffer := make([]byte, s.up.PartSize)
	for number := s.entry.PartsDone + 1; number <= s.up.PartCount; number++ {
		part, err := r.readPart(s, number, buffer[:r.getPartLength(s, number)])
		if err != nil {
			return transfer.UploadResult{}, err
		}
		if err := r.deps.Uploads.PutPart(ctx, s.up, number, part); err != nil {
			return transfer.UploadResult{}, err
		}
		if err := r.keep(ctx, s, number); err != nil {
			return transfer.UploadResult{}, err
		}
	}
	if err := r.deps.Uploads.Finish(ctx, s.up); err != nil {
		return transfer.UploadResult{}, err
	}
	return transfer.UploadResult{BlobID: s.up.Blob, BlobHash: [32]byte(s.hasher.Sum(nil)), BlobSize: uint64(s.sealer.StoredSize()), Key: s.key}, nil
}

// keep writes the state of the upload after a part, with the hash of what was sent.
func (r *run) keep(ctx context.Context, s *session, number int) error {
	marshaler, ok := s.hasher.(encoding.BinaryMarshaler)
	if !ok {
		return errors.New("the hash of the stored bytes cannot be kept")
	}
	state, err := marshaler.MarshalBinary()
	if err != nil {
		return fmt.Errorf("keep the hash of the stored bytes: %w", err)
	}
	s.entry.PartsDone, s.entry.StoredHashState = number, state
	entries, err := r.deps.Entries.ReadUploads(ctx, r.in.Vault)
	if err != nil {
		return err
	}
	return r.deps.Entries.WriteUploads(ctx, r.in.Vault, append(withoutFile(entries, s.entry.FileID), s.entry))
}

// drop ends an upload of a file that changed: the reservation is aborted and the state forgotten.
func (r *run) drop(ctx context.Context, s *session) error {
	return errors.Join(r.deps.Uploads.Abort(ctx, s.up), r.removeEntry(ctx, s.entry))
}
