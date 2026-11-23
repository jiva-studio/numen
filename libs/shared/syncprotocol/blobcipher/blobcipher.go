package blobcipher

import (
	"fmt"
	"io"
)

// Seal reads the salt from random and returns the stored bytes of plain. A text body is compressed first.
// It returns ErrTooLarge when the stored size is over 64 MiB or a text body is over 256 MiB.
func Seal(random io.Reader, revisionKey [32]byte, fileID [16]byte, plain []byte, isText bool) ([]byte, error) {
	payload, flags, err := createPayload(plain, isText)
	if err != nil {
		return nil, err
	}
	const chunkSize = 1 << defaultLog2
	if getStoredSize(len(payload), chunkSize) > maxStoredSize {
		return nil, ErrTooLarge
	}
	salt, err := readSalt(random)
	if err != nil {
		return nil, err
	}
	aead, commit, err := deriveKeys(revisionKey, salt)
	if err != nil {
		return nil, err
	}
	h := newHeader(flags, defaultLog2, salt, commit)
	sealer := chunkSealer{aead: aead, header: h, fileID: fileID}
	return append(h[:], sealStream(sealer, createStream(payload), chunkSize)...), nil
}

// Open returns the plain body of a blob made by Seal. Every failure to read the blob is ErrCannotOpen.
func Open(revisionKey [32]byte, fileID [16]byte, blob []byte) ([]byte, error) {
	h, flags, chunkLog2, err := parseHeader(blob)
	if err != nil {
		return nil, err
	}
	sealer, err := newSealerFromHeader(h, revisionKey, fileID)
	if err != nil {
		return nil, err
	}
	stream, err := openStream(sealer, blob[headerSize:], 1<<chunkLog2)
	if err != nil {
		return nil, err
	}
	payload, err := readPayload(stream)
	if err != nil {
		return nil, err
	}
	if flags&flagCompressed != 0 {
		return decompress(payload)
	}
	return payload, nil
}

func createPayload(plain []byte, isText bool) (payload []byte, flags byte, err error) {
	if !isText {
		return plain, 0, nil
	}
	if len(plain) > maxPlainSize {
		return nil, 0, ErrTooLarge
	}
	payload, err = compress(plain)
	if err != nil {
		return nil, 0, fmt.Errorf("compress the body: %w", err)
	}
	return payload, flagCompressed, nil
}
