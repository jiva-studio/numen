package blobcipher_test

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	headerSize     = 56
	tagSize        = 16
	defaultLog2    = 20
	mebibyte       = 1 << 20
	flagCompressed = 1
)

type layout struct {
	revisionKey [32]byte
	salt        [16]byte
	fileID      [16]byte
	flags       byte
	chunkLog2   byte
	reserved    [2]byte
	commit      []byte
	unmarkLast  bool
}

func newLayout() layout {
	var l layout
	for i := range l.revisionKey {
		l.revisionKey[i] = byte(0xA0 + i)
	}
	for i := range l.salt {
		l.salt[i] = byte(0x10 + i)
	}
	for i := range l.fileID {
		l.fileID[i] = byte(0x40 + i)
	}
	l.chunkLog2 = defaultLog2
	return l
}

func deriveBlobKey(tb testing.TB, l layout, info string) []byte {
	tb.Helper()
	derived, err := hkdf.Key(sha256.New, l.revisionKey[:], l.salt[:], info, 32)
	if err != nil {
		tb.Fatal(err)
	}
	return derived
}

func paddedLength(payloadLength int) int {
	m := 8 + payloadLength
	if m <= mebibyte {
		padded := 4096
		for padded < m {
			padded *= 2
		}
		return padded
	}
	return (m + mebibyte - 1) / mebibyte * mebibyte
}

func buildStream(payload []byte) []byte {
	stream := make([]byte, paddedLength(len(payload)))
	binary.BigEndian.PutUint64(stream, uint64(len(payload)))
	copy(stream[8:], payload)
	return stream
}

func buildHeader(tb testing.TB, l layout) []byte {
	tb.Helper()
	commit := l.commit
	if commit == nil {
		commit = deriveBlobKey(tb, l, "numen-sync/blob/commit/v1")
	}
	header := []byte("NSB1")
	header = append(header, l.flags, l.chunkLog2, l.reserved[0], l.reserved[1])
	header = append(header, l.salt[:]...)
	return append(header, commit...)
}

func buildChunks(tb testing.TB, l layout, stream []byte) [][]byte {
	tb.Helper()
	aead, err := chacha20poly1305.NewX(deriveBlobKey(tb, l, "numen-sync/blob/key/v1"))
	if err != nil {
		tb.Fatal(err)
	}
	header := buildHeader(tb, l)
	size := 1 << l.chunkLog2
	var chunks [][]byte
	for index, offset := uint64(0), 0; offset < len(stream); index, offset = index+1, offset+size {
		end := min(offset+size, len(stream))
		last := byte(0)
		if end == len(stream) && !l.unmarkLast {
			last = 1
		}
		nonce := make([]byte, 24)
		binary.BigEndian.PutUint64(nonce[16:], index)
		associated := append(bytes.Clone(header), l.fileID[:]...)
		associated = binary.BigEndian.AppendUint64(associated, index)
		associated = append(associated, last)
		chunks = append(chunks, aead.Seal(nil, nonce, stream[offset:end], associated))
	}
	return chunks
}

func buildBlob(tb testing.TB, l layout, payload []byte) []byte {
	tb.Helper()
	return buildBlobOfStream(tb, l, buildStream(payload))
}

func buildBlobOfStream(tb testing.TB, l layout, stream []byte) []byte {
	tb.Helper()
	blob := buildHeader(tb, l)
	for _, chunk := range buildChunks(tb, l, stream) {
		blob = append(blob, chunk...)
	}
	return blob
}

func compress(tb testing.TB, plain []byte) []byte {
	tb.Helper()
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		tb.Fatal(err)
	}
	defer func() {
		if err := encoder.Close(); err != nil {
			tb.Error(err)
		}
	}()
	return encoder.EncodeAll(plain, nil)
}

func newPlain(size int) []byte {
	plain := make([]byte, size)
	state := uint32(2463534242)
	for i := range plain {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		plain[i] = byte(state)
	}
	return plain
}
