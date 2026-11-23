package blobcipher

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

const (
	lengthPrefixSize = 8
	minPaddedLength  = 4096
	mebibyte         = 1 << 20
	maxStoredSize    = 64 * mebibyte
	maxPlainSize     = 256 * mebibyte
)

// PaddedLength returns the length of the plain stream for a payload of the given size: the 8-byte length prefix,
// the payload and zero bytes up to the next step of the ladder.
func PaddedLength(payloadLength int) int {
	return int(paddedLength(int64(payloadLength)))
}

func paddedLength(payloadLength int64) int64 {
	m := lengthPrefixSize + payloadLength
	if m <= mebibyte {
		padded := int64(minPaddedLength)
		for padded < m {
			padded *= 2
		}
		return padded
	}
	return (m + mebibyte - 1) / mebibyte * mebibyte
}

func getStoredSize(payloadLength, chunkSize int) int {
	padded := PaddedLength(payloadLength)
	chunks := (padded + chunkSize - 1) / chunkSize
	return headerSize + padded + chunks*tagSize
}

func createStream(payload []byte) []byte {
	stream := make([]byte, PaddedLength(len(payload)))
	binary.BigEndian.PutUint64(stream, uint64(len(payload)))
	copy(stream[lengthPrefixSize:], payload)
	return stream
}

func readPayload(stream []byte) ([]byte, error) {
	if len(stream) < lengthPrefixSize {
		return nil, fmt.Errorf("%w: the stream is shorter than its length prefix", ErrCannotOpen)
	}
	length := binary.BigEndian.Uint64(stream)
	if length > uint64(len(stream)-lengthPrefixSize) {
		return nil, fmt.Errorf("%w: the length prefix is larger than the stream", ErrCannotOpen)
	}
	return stream[lengthPrefixSize : lengthPrefixSize+int(length)], nil
}

func compress(plain []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
	if err != nil {
		return nil, fmt.Errorf("create the zstd encoder: %w", err)
	}
	compressed := encoder.EncodeAll(plain, nil)
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("close the zstd encoder: %w", err)
	}
	return compressed, nil
}

func decompress(compressed []byte) ([]byte, error) {
	if len(compressed) == 0 {
		return nil, nil
	}
	decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(maxPlainSize))
	if err != nil {
		return nil, fmt.Errorf("create the zstd decoder: %w", err)
	}
	defer decoder.Close()
	plain, err := decoder.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: the body is not valid zstd", ErrCannotOpen)
	}
	return plain, nil
}

func readSalt(random io.Reader) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(random, salt); err != nil {
		return nil, fmt.Errorf("read the salt: %w", err)
	}
	return salt, nil
}

func sealStream(s chunkSealer, stream []byte, chunkSize int) []byte {
	sealed := make([]byte, 0, len(stream)+(len(stream)/chunkSize+1)*tagSize)
	for index, offset := uint64(0), 0; offset < len(stream); index, offset = index+1, offset+chunkSize {
		end := min(offset+chunkSize, len(stream))
		sealed = s.seal(sealed, index, end == len(stream), stream[offset:end])
	}
	return sealed
}

func openStream(s chunkSealer, body []byte, chunkSize int) ([]byte, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: the blob has no chunk", ErrCannotOpen)
	}
	stream := make([]byte, 0, len(body))
	for index := uint64(0); len(body) > 0; index++ {
		size := min(chunkSize+tagSize, len(body))
		var err error
		stream, err = s.open(stream, index, size == len(body), body[:size])
		if err != nil {
			return nil, err
		}
		body = body[size:]
	}
	return stream, nil
}
