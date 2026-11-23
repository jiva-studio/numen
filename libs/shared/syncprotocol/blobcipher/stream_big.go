package blobcipher

import (
	"errors"
	"io"
)

var (
	ErrShortPlain     = errors.New("blobcipher: the plain source is shorter than the declared size")
	errNotImplemented = errors.New("blobcipher: not implemented")
)

// Header is the parsed fixed header of a stored blob.
type Header struct {
	Flags     byte
	ChunkLog2 byte
}

// Sealer produces the stored bytes of a file of known size at any offset.
type Sealer struct{}

// PlainReader returns the payload of a stored stream chunk by chunk.
type PlainReader struct{}

func NewSealer(revisionKey [32]byte, fileID [16]byte, salt [16]byte, plainSize int64) (*Sealer, error) {
	return nil, errNotImplemented
}

func GenerateSalt(random io.Reader) ([16]byte, error) {
	return [16]byte{}, errNotImplemented
}

func (s *Sealer) StoredSize() int64 {
	return 0
}

func (s *Sealer) ReadStoredAt(p []byte, off int64, plain io.ReaderAt) (int, error) {
	return 0, errNotImplemented
}

func OpenStream(revisionKey [32]byte, fileID [16]byte, stored io.Reader) (*PlainReader, error) {
	return nil, errNotImplemented
}

func OpenStreamFrom(revisionKey [32]byte, fileID [16]byte, header [56]byte, chunkIndex uint64, stored io.Reader) (*PlainReader, error) {
	return nil, errNotImplemented
}

func (r *PlainReader) Read(p []byte) (int, error) {
	return 0, errNotImplemented
}

func (r *PlainReader) Size() int64 {
	return 0
}

func GetChunkOffset(chunkLog2 byte, chunkIndex uint64) int64 {
	return 0
}

func ParseHeader(header []byte) (Header, error) {
	return Header{}, errNotImplemented
}
