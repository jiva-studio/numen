package note

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
)

// ErrCorruptBlob means a blob the server sent is not the one its revision names, or does not open.
var ErrCorruptBlob = errors.New("the blob of a revision is damaged")

// OpenBlob returns the plain content of the stored blob of a revision: the blob must have the hash the revision
// names and open under the key of the revision to content with the hash of the metadata.
func OpenBlob(meta Meta, fileID, blobHash, blob []byte) ([]byte, error) {
	sum := sha256.Sum256(blob)
	if !bytes.Equal(sum[:], blobHash) {
		return nil, fmt.Errorf("%w: its hash differs", ErrCorruptBlob)
	}
	plain, err := blobcipher.Open([32]byte(meta.Key), [16]byte(fileID), blob)
	if err != nil || Hash(plain) != meta.Hash {
		return nil, fmt.Errorf("%w: it does not open to the note", ErrCorruptBlob)
	}
	return plain, nil
}
