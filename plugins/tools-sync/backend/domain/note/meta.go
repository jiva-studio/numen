package note

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	metaVersion  = 0x01
	metaContext  = "numen-sync/meta/v1"
	KindMarkdown = "markdown"
)

// errCannotOpenMeta means the metadata of a revision does not open with the vault key.
var errCannotOpenMeta = errors.New("the metadata of the revision cannot be opened")

// Meta is what a revision says of its note, sealed with the vault key.
type Meta struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	Hash  string `json:"hash"`
	Key   []byte `json:"key"`
	Kind  string `json:"kind"`
}

func getMetaData(vaultID, fileID []byte) []byte {
	return bytes.Join([][]byte{[]byte(metaContext), vaultID, fileID}, nil)
}

const metaPadStep = 64

// SealMeta returns the sealed metadata: one byte of version, a random nonce and the JSON sealed under the vault
// key, bound to the vault and the file. The JSON is padded with spaces to a multiple of 64 bytes.
func SealMeta(random io.Reader, vaultKey [32]byte, vaultID, fileID []byte, meta Meta) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(vaultKey[:])
	if err != nil {
		return nil, fmt.Errorf("make the metadata cipher: %w", err)
	}
	plain, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("encode the metadata: %w", err)
	}
	plain = append(plain, bytes.Repeat([]byte{' '}, (metaPadStep-len(plain)%metaPadStep)%metaPadStep)...)
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(random, nonce); err != nil {
		return nil, fmt.Errorf("read a nonce: %w", err)
	}
	sealed := append([]byte{metaVersion}, nonce...)
	return aead.Seal(sealed, nonce, plain, getMetaData(vaultID, fileID)), nil
}

func OpenMeta(vaultKey [32]byte, vaultID, fileID, sealed []byte) (Meta, error) {
	if len(sealed) < 1+chacha20poly1305.NonceSizeX || sealed[0] != metaVersion {
		return Meta{}, errCannotOpenMeta
	}
	aead, err := chacha20poly1305.NewX(vaultKey[:])
	if err != nil {
		return Meta{}, fmt.Errorf("make the metadata cipher: %w", err)
	}
	nonce, body := sealed[1:1+chacha20poly1305.NonceSizeX], sealed[1+chacha20poly1305.NonceSizeX:]
	plain, err := aead.Open(nil, nonce, body, getMetaData(vaultID, fileID))
	if err != nil {
		return Meta{}, errCannotOpenMeta
	}
	var meta Meta
	if err := json.Unmarshal(plain, &meta); err != nil {
		return Meta{}, errCannotOpenMeta
	}
	return meta, nil
}

func NewID(random io.Reader, n int) ([]byte, error) {
	id := make([]byte, n)
	if _, err := io.ReadFull(random, id); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return id, nil
}
