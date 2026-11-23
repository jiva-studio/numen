package blobcipher

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	bigChunkLog2     = 20
	bigChunkSize     = 1 << bigChunkLog2
	maxMultipartSize = 5<<40 - 5<<20
)

var ErrShortPlain = errors.New("blobcipher: the plain source is shorter than the declared size")

// Header is the parsed fixed header of a stored blob.
type Header struct {
	Flags     byte
	ChunkLog2 byte
}

// Sealer produces the stored bytes of a file of known size at any offset, so a break loses nothing: the bytes of
// every position are a pure function of the key, the salt, the file id and the plain content.
type Sealer struct {
	chunks    chunkSealer
	plainSize int64
	padded    int64
}

// NewSealer returns the sealer of a plain body of plainSize bytes, never compressed, in chunks of 1 MiB. It
// returns ErrTooLarge when the stored size is over 5 TiB less 5 MiB.
func NewSealer(revisionKey [32]byte, fileID [16]byte, salt [16]byte, plainSize int64) (*Sealer, error) {
	if plainSize < 0 {
		return nil, fmt.Errorf("blobcipher: the plain size %d is negative", plainSize)
	}
	if plainSize > maxMultipartSize {
		return nil, ErrTooLarge
	}
	padded := computePaddedLength64(plainSize)
	s := &Sealer{plainSize: plainSize, padded: padded}
	if s.StoredSize() > maxMultipartSize {
		return nil, ErrTooLarge
	}
	aead, commit, err := deriveKeys(revisionKey, salt[:])
	if err != nil {
		return nil, err
	}
	s.chunks = chunkSealer{aead: aead, header: newHeader(0, bigChunkLog2, salt[:], commit), fileID: fileID}
	return s, nil
}

func GenerateSalt(random io.Reader) ([16]byte, error) {
	var salt [16]byte
	if _, err := io.ReadFull(random, salt[:]); err != nil {
		return salt, fmt.Errorf("read the salt: %w", err)
	}
	return salt, nil
}

// StoredSize is the size of the whole stored stream: the header, the padded plain stream and a tag per chunk.
func (s *Sealer) StoredSize() int64 {
	chunks := (s.padded + bigChunkSize - 1) / bigChunkSize
	return headerSize + s.padded + chunks*tagSize
}

// ReadStoredAt fills p with the stored bytes from off. It reads from plain only the range of the chunks that
// overlap, and returns io.EOF with the count read at the end of the stream.
func (s *Sealer) ReadStoredAt(p []byte, off int64, plain io.ReaderAt) (int, error) {
	total := s.StoredSize()
	if off < 0 {
		return 0, fmt.Errorf("blobcipher: the offset %d is negative", off)
	}
	count := 0
	if off < headerSize && off < total {
		count = copy(p, s.chunks.header[off:])
		off += int64(count)
	}
	var stream, sealed []byte
	for count < len(p) && off < total {
		if stream == nil {
			stream = make([]byte, bigChunkSize)
			sealed = make([]byte, 0, bigChunkSize+tagSize)
		}
		index, within := uint64((off-headerSize)/(bigChunkSize+tagSize)), (off-headerSize)%(bigChunkSize+tagSize)
		chunk, err := s.sealChunk(sealed, stream, index, plain)
		if err != nil {
			return count, err
		}
		taken := copy(p[count:], chunk[within:])
		count += taken
		off += int64(taken)
	}
	if count < len(p) {
		return count, io.EOF
	}
	return count, nil
}

func (s *Sealer) sealChunk(sealed, stream []byte, index uint64, plain io.ReaderAt) ([]byte, error) {
	start := int64(index) * bigChunkSize
	end := min(start+bigChunkSize, s.padded)
	chunk := stream[:end-start]
	clear(chunk)
	if start < lengthPrefixSize {
		var prefix [lengthPrefixSize]byte
		binary.BigEndian.PutUint64(prefix[:], uint64(s.plainSize))
		copy(chunk, prefix[start:])
	}
	from, to := max(start, lengthPrefixSize), min(end, lengthPrefixSize+s.plainSize)
	if to > from {
		want := to - from
		n, err := plain.ReadAt(chunk[from-start:to-start], from-lengthPrefixSize)
		if int64(n) < want {
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("read the plain body: %w", err)
			}
			return nil, ErrShortPlain
		}
	}
	return s.chunks.seal(sealed[:0], index, end == s.padded, chunk), nil
}

// ParseHeader reads the flags and the chunk size of a header, so a resumed download can continue.
func ParseHeader(header []byte) (Header, error) {
	_, flags, chunkLog2, err := parseHeader(header)
	if err != nil {
		return Header{}, err
	}
	return Header{Flags: flags, ChunkLog2: chunkLog2}, nil
}

func GetChunkOffset(chunkLog2 byte, chunkIndex uint64) int64 {
	return headerSize + int64(chunkIndex)*(int64(1)<<chunkLog2+tagSize)
}

// PlainReader returns the payload of a stored stream chunk by chunk, authenticating each chunk before it returns
// any of its bytes.
type PlainReader struct {
	chunks      chunkSealer
	source      *bufio.Reader
	sealed      []byte
	opened      []byte
	pending     []byte
	index       uint64
	remaining   int64
	hasSize     bool
	isFinished  bool
	failure     error
	chunkStride int
	header      [headerSize]byte
	sink        io.Writer
}

// OpenStream reads the header of a stored stream and returns its payload, never the prefix or the padding. A
// missing last flag, data after the last chunk, a length prefix larger than the stream, or a compressed body
// fail with ErrCannotOpen.
func OpenStream(revisionKey [32]byte, fileID [16]byte, stored io.Reader) (*PlainReader, error) {
	source := bufio.NewReaderSize(stored, 4096)
	var header [headerSize]byte
	if _, err := io.ReadFull(source, header[:]); err != nil {
		return nil, fmt.Errorf("%w: the stream is shorter than a header", ErrCannotOpen)
	}
	return newPlainReader(revisionKey, fileID, header, 0, 0, source)
}

// OpenStreamFrom resumes at a chunk boundary of a download with the header of the first request. plainSize is the
// payload size, which a chunk after the first does not carry.
func OpenStreamFrom(revisionKey [32]byte, fileID [16]byte, header [headerSize]byte, chunkIndex uint64, plainSize int64, stored io.Reader) (*PlainReader, error) {
	return newPlainReader(revisionKey, fileID, header, chunkIndex, plainSize, bufio.NewReaderSize(stored, 4096))
}

func newPlainReader(revisionKey [32]byte, fileID [16]byte, header [headerSize]byte, index uint64, plainSize int64, source *bufio.Reader) (*PlainReader, error) {
	h, flags, chunkLog2, err := parseHeader(header[:])
	if err != nil {
		return nil, err
	}
	if flags&flagCompressed != 0 {
		return nil, fmt.Errorf("%w: a compressed body is opened whole", ErrCannotOpen)
	}
	chunks, err := newSealerFromHeader(h, revisionKey, fileID)
	if err != nil {
		return nil, err
	}
	stride := 1<<chunkLog2 + tagSize
	reader := &PlainReader{chunks: chunks, source: source, index: index, chunkStride: stride, header: header, sealed: make([]byte, stride), opened: make([]byte, 0, stride)}
	if index > 0 {
		reader.remaining, reader.hasSize = plainSize-(int64(index)<<chunkLog2-lengthPrefixSize), true
	}
	return reader, nil
}

// Header returns the header the stream began with, which a resume of a download keeps.
func (r *PlainReader) Header() [headerSize]byte { return r.header }

// ChunkIndex is the number of the next chunk to load, so the number of chunks loaded since the start of the stream.
func (r *PlainReader) ChunkIndex() uint64 { return r.index }

// IsAtChunkBoundary reports whether Read returned every byte of the chunks loaded so far.
func (r *PlainReader) IsAtChunkBoundary() bool { return len(r.pending) == 0 }

// SetStoredSink passes the stored bytes of each chunk, as loaded and before it opens, to the writer, so a caller
// can hash exactly the chunks it has taken.
func (r *PlainReader) SetStoredSink(sink io.Writer) { r.sink = sink }

// Read returns payload bytes.
func (r *PlainReader) Read(p []byte) (int, error) {
	for len(r.pending) == 0 {
		if r.failure != nil {
			return 0, r.failure
		}
		if r.isFinished {
			r.failure = io.EOF
			if r.remaining > 0 {
				r.failure = fmt.Errorf("%w: the length prefix is larger than the stream", ErrCannotOpen)
			}
			continue
		}
		if err := r.loadChunk(); err != nil {
			r.failure = err
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

func (r *PlainReader) loadChunk() error {
	n, err := io.ReadFull(r.source, r.sealed)
	switch {
	case errors.Is(err, io.EOF):
		return fmt.Errorf("%w: the stream has no chunk", ErrCannotOpen)
	case errors.Is(err, io.ErrUnexpectedEOF):
		r.isFinished = true
	case err != nil:
		return fmt.Errorf("read the stored stream: %w", err)
	default:
		if _, peekErr := r.source.Peek(1); errors.Is(peekErr, io.EOF) {
			r.isFinished = true
		}
	}
	if n < tagSize {
		return fmt.Errorf("%w: a chunk is shorter than its tag", ErrCannotOpen)
	}
	if r.sink != nil {
		if _, err := r.sink.Write(r.sealed[:n]); err != nil {
			return fmt.Errorf("pass on the stored chunk: %w", err)
		}
	}
	opened, err := r.chunks.open(r.opened[:0], r.index, r.isFinished, r.sealed[:n])
	if err != nil {
		return err
	}
	r.index++
	if !r.hasSize {
		if len(opened) < lengthPrefixSize {
			return fmt.Errorf("%w: the stream is shorter than its length prefix", ErrCannotOpen)
		}
		length := binary.BigEndian.Uint64(opened)
		if length > maxMultipartSize {
			return fmt.Errorf("%w: the length prefix is larger than the stream", ErrCannotOpen)
		}
		r.remaining, r.hasSize = int64(length), true
		opened = opened[lengthPrefixSize:]
	}
	take := min(int64(len(opened)), r.remaining)
	r.remaining -= take
	r.pending = opened[:take]
	return nil
}
