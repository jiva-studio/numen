package blobcipher

import (
	"bytes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	headerSize      = 56
	tagSize         = chacha20poly1305.Overhead
	magic           = "NSB1"
	flagCompressed  = 1
	defaultLog2     = 20
	minLog2         = 16
	maxLog2         = 22
	saltSize        = 16
	commitSize      = 32
	saltOffset      = 8
	commitOffset    = saltOffset + saltSize
	keyInfo         = "numen-sync/blob/key/v1"
	commitInfo      = "numen-sync/blob/commit/v1"
	nonceCounterAt  = chacha20poly1305.NonceSizeX - 8
	derivedKeyBytes = 32
)

var (
	ErrTooLarge   = errors.New("blobcipher: stored size is over the limit")
	ErrCannotOpen = errors.New("blobcipher: cannot open the blob")
)

type header [headerSize]byte

type chunkSealer struct {
	aead   cipher.AEAD
	header header
	fileID [16]byte
}

func newHeader(flags, chunkLog2 byte, salt, commit []byte) header {
	var h header
	copy(h[:4], magic)
	h[4] = flags
	h[5] = chunkLog2
	copy(h[saltOffset:], salt)
	copy(h[commitOffset:], commit)
	return h
}

func deriveKeys(revisionKey [32]byte, salt []byte) (aead cipher.AEAD, commit []byte, err error) {
	key, err := hkdf.Key(sha256.New, revisionKey[:], salt, keyInfo, derivedKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("derive the blob key: %w", err)
	}
	commit, err = hkdf.Key(sha256.New, revisionKey[:], salt, commitInfo, commitSize)
	if err != nil {
		return nil, nil, fmt.Errorf("derive the commitment: %w", err)
	}
	aead, err = chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create the cipher: %w", err)
	}
	return aead, commit, nil
}

func (s chunkSealer) getNonceAndData(index uint64, isLast bool) (nonce, data []byte) {
	nonce = make([]byte, chacha20poly1305.NonceSizeX)
	binary.BigEndian.PutUint64(nonce[nonceCounterAt:], index)
	data = append(bytes.Clone(s.header[:]), s.fileID[:]...)
	data = binary.BigEndian.AppendUint64(data, index)
	if isLast {
		return nonce, append(data, 1)
	}
	return nonce, append(data, 0)
}

func (s chunkSealer) seal(dst []byte, index uint64, isLast bool, plain []byte) []byte {
	nonce, data := s.getNonceAndData(index, isLast)
	return s.aead.Seal(dst, nonce, plain, data)
}

func (s chunkSealer) open(dst []byte, index uint64, isLast bool, sealed []byte) ([]byte, error) {
	nonce, data := s.getNonceAndData(index, isLast)
	opened, err := s.aead.Open(dst, nonce, sealed, data)
	if err != nil {
		return nil, fmt.Errorf("%w: chunk %d does not verify", ErrCannotOpen, index)
	}
	return opened, nil
}

func parseHeader(blob []byte) (h header, flags byte, chunkLog2 byte, err error) {
	if len(blob) < headerSize {
		return h, 0, 0, fmt.Errorf("%w: %d bytes are less than a header", ErrCannotOpen, len(blob))
	}
	copy(h[:], blob)
	flags, chunkLog2 = h[4], h[5]
	isKnown := string(h[:4]) == magic && flags&^flagCompressed == 0 && h[6] == 0 && h[7] == 0
	if !isKnown || chunkLog2 < minLog2 || chunkLog2 > maxLog2 {
		return h, 0, 0, fmt.Errorf("%w: unknown header", ErrCannotOpen)
	}
	return h, flags, chunkLog2, nil
}

func newSealerFromHeader(h header, revisionKey [32]byte, fileID [16]byte) (chunkSealer, error) {
	aead, commit, err := deriveKeys(revisionKey, h[saltOffset:commitOffset])
	if err != nil {
		return chunkSealer{}, err
	}
	if subtle.ConstantTimeCompare(commit, h[commitOffset:]) != 1 {
		return chunkSealer{}, fmt.Errorf("%w: the commitment does not match the key", ErrCannotOpen)
	}
	return chunkSealer{aead: aead, header: h, fileID: fileID}, nil
}
