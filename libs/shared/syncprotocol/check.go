package syncprotocol

import (
	"errors"
	"fmt"
	"slices"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	idSize        = 16
	hashSize      = 32
	signKeySize   = 32
	signatureSize = 64
)

var (
	// ErrInvalidRevision means a revision has a shape or a size the protocol does not allow.
	ErrInvalidRevision = errors.New("the revision is invalid")
	// ErrInvalidCatalogEntry means a catalog entry has a shape or a size the protocol does not allow.
	ErrInvalidCatalogEntry = errors.New("the catalog entry is invalid")
)

type sizedField struct {
	name  string
	value []byte
	sizes []int
}

func checkSizes(failure error, fields []sizedField) error {
	for _, field := range fields {
		if !slices.Contains(field.sizes, len(field.value)) {
			return fmt.Errorf("%w: %s has %d bytes", failure, field.name, len(field.value))
		}
	}
	return nil
}

// CheckRevision checks the shape and the sizes of a revision, not its signature or its place in the history.
// A tombstone has no blob and no size, and may have no meta.
func CheckRevision(revision *syncv1.Revision) error {
	if revision == nil {
		return fmt.Errorf("%w: it is missing", ErrInvalidRevision)
	}
	err := checkSizes(ErrInvalidRevision, []sizedField{
		{"vault_id", revision.GetVaultId(), []int{idSize}},
		{"file_id", revision.GetFileId(), []int{idSize}},
		{"parent_rev", revision.GetParentRev(), []int{0, hashSize}},
		{"path_hash", revision.GetPathHash(), []int{hashSize}},
		{"share_root_id", revision.GetShareRootId(), []int{0, idSize}},
		{"author_device_sign_key", revision.GetAuthorDeviceSignKey(), []int{signKeySize}},
		{"signature", revision.GetSignature(), []int{signatureSize}},
	})
	if err != nil {
		return err
	}
	if len(revision.GetEncMeta()) > MaxEncSize {
		return fmt.Errorf("%w: enc_meta has %d bytes", ErrInvalidRevision, len(revision.GetEncMeta()))
	}
	if revision.GetDeleted() {
		return checkTombstone(revision)
	}
	return checkBlobRevision(revision)
}

func checkTombstone(revision *syncv1.Revision) error {
	if len(revision.GetBlobId()) != 0 || len(revision.GetBlobHash()) != 0 || revision.GetBlobSize() != 0 {
		return fmt.Errorf("%w: a tombstone has a blob", ErrInvalidRevision)
	}
	return nil
}

func checkBlobRevision(revision *syncv1.Revision) error {
	err := checkSizes(ErrInvalidRevision, []sizedField{
		{"blob_id", revision.GetBlobId(), []int{idSize}},
		{"blob_hash", revision.GetBlobHash(), []int{hashSize}},
	})
	if err != nil {
		return err
	}
	if revision.GetBlobSize() == 0 || revision.GetBlobSize() > MaxMultipartSize {
		return fmt.Errorf("%w: blob_size is %d", ErrInvalidRevision, revision.GetBlobSize())
	}
	if len(revision.GetEncMeta()) == 0 {
		return fmt.Errorf("%w: enc_meta is empty", ErrInvalidRevision)
	}
	return nil
}

// CheckCatalogEntry checks the shape and the sizes of a catalog entry, not its signature or its place in the log.
func CheckCatalogEntry(entry *syncv1.CatalogEntry) error {
	if entry == nil {
		return fmt.Errorf("%w: it is missing", ErrInvalidCatalogEntry)
	}
	if entry.GetVersion() == 0 {
		return fmt.Errorf("%w: version is 0", ErrInvalidCatalogEntry)
	}
	previous := []int{hashSize}
	if entry.GetVersion() == 1 {
		previous = []int{0}
	}
	if len(entry.GetEncBody()) > MaxEncSize {
		return fmt.Errorf("%w: enc_body has %d bytes", ErrInvalidCatalogEntry, len(entry.GetEncBody()))
	}
	return checkSizes(ErrInvalidCatalogEntry, []sizedField{
		{"prev_hash", entry.GetPrevHash(), previous},
		{"vault_id", entry.GetVaultId(), []int{idSize}},
		{"author_device_sign_key", entry.GetAuthorDeviceSignKey(), []int{signKeySize}},
		{"signature", entry.GetSignature(), []int{signatureSize}},
	})
}
