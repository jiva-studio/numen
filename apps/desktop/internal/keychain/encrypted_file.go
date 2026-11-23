package keychain

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// ErrWrongPassphrase is returned when the passphrase does not open the credentials file, or the file was changed.
var ErrWrongPassphrase = errors.New("the passphrase does not open the credentials file")

var errBadEncryptedFile = errors.New("the credentials file is not a valid encrypted file")

var encryptedMagic = []byte("NMCRED")

const (
	encryptedVersion = 1
	saltSize         = 16
	headerSize       = 6 + 1 + 4 + 4 + 1 + saltSize
	maxKDFMemoryKiB  = 1 << 20
	maxKDFTime       = 16
	maxKDFThreads    = 16
	minKDFMemoryKiB  = 8
)

// kdfParams are the Argon2id cost parameters; they are written into the file header.
type kdfParams struct {
	memoryKiB uint32
	time      uint32
	threads   uint8
}

// defaultKDFParams follow RFC 9106 with at least 64 MiB of memory.
var defaultKDFParams = kdfParams{memoryKiB: 64 * 1024, time: 3, threads: 1}

func isEncryptedFile(raw []byte) bool {
	return bytes.HasPrefix(raw, encryptedMagic)
}

func encryptEntries(entries map[string]string, passphrase string, params kdfParams) ([]byte, error) {
	plain, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("encrypt the credentials: %w", err)
	}
	salt := make([]byte, saltSize)
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("encrypt the credentials: %w", err)
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("encrypt the credentials: %w", err)
	}
	header := writeHeader(params, salt)
	aead, err := chacha20poly1305.NewX(deriveKey(passphrase, salt, params))
	if err != nil {
		return nil, fmt.Errorf("encrypt the credentials: %w", err)
	}
	out := append(header, nonce...)
	return aead.Seal(out, nonce, plain, header), nil
}

func decryptEntries(raw []byte, passphrase string) (map[string]string, error) {
	if len(raw) < headerSize+chacha20poly1305.NonceSizeX+chacha20poly1305.Overhead {
		return nil, errBadEncryptedFile
	}
	header := raw[:headerSize]
	params, salt, err := readHeader(header)
	if err != nil {
		return nil, err
	}
	nonce := raw[headerSize : headerSize+chacha20poly1305.NonceSizeX]
	aead, err := chacha20poly1305.NewX(deriveKey(passphrase, salt, params))
	if err != nil {
		return nil, fmt.Errorf("decrypt the credentials: %w", err)
	}
	plain, err := aead.Open(nil, nonce, raw[headerSize+chacha20poly1305.NonceSizeX:], header)
	if err != nil {
		return nil, ErrWrongPassphrase
	}
	entries := map[string]string{}
	if err := json.Unmarshal(plain, &entries); err != nil {
		return nil, fmt.Errorf("%w: %s", errBadEncryptedFile, describeJSONSyntax(err))
	}
	return entries, nil
}

func deriveKey(passphrase string, salt []byte, params kdfParams) []byte {
	return argon2.IDKey([]byte(passphrase), salt, params.time, params.memoryKiB, params.threads, chacha20poly1305.KeySize)
}

func writeHeader(params kdfParams, salt []byte) []byte {
	header := make([]byte, 0, headerSize)
	header = append(header, encryptedMagic...)
	header = append(header, encryptedVersion)
	header = binary.BigEndian.AppendUint32(header, params.memoryKiB)
	header = binary.BigEndian.AppendUint32(header, params.time)
	header = append(header, params.threads)
	return append(header, salt...)
}

func readHeader(header []byte) (kdfParams, []byte, error) {
	if !isEncryptedFile(header) || header[len(encryptedMagic)] != encryptedVersion {
		return kdfParams{}, nil, fmt.Errorf("%w: unknown format", errBadEncryptedFile)
	}
	rest := header[len(encryptedMagic)+1:]
	params := kdfParams{
		memoryKiB: binary.BigEndian.Uint32(rest[0:4]),
		time:      binary.BigEndian.Uint32(rest[4:8]),
		threads:   rest[8],
	}
	if params.memoryKiB < minKDFMemoryKiB || params.memoryKiB > maxKDFMemoryKiB ||
		params.time < 1 || params.time > maxKDFTime ||
		params.threads < 1 || params.threads > maxKDFThreads {
		return kdfParams{}, nil, fmt.Errorf("%w: the key derivation parameters are out of range", errBadEncryptedFile)
	}
	return params, rest[9 : 9+saltSize], nil
}
