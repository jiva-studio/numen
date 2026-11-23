package syncprotocol

import (
	"errors"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// GetShareLogEntryMessage returns the bytes a device signs to append a share log entry: the label
// numen-sync/share-log/v1, a zero byte, the fields of the entry in the frozen order. The signature is never part of it.
func GetShareLogEntryMessage(_ *syncv1.ShareLogEntry) []byte {
	return nil
}

// GetShareLogEntryID returns the id of a share log entry: the SHA-256 of GetShareLogEntryMessage. The prev_hash of
// the next entry is this value.
func GetShareLogEntryID(_ *syncv1.ShareLogEntry) [32]byte {
	return [32]byte{}
}

// GetRevisionMessageWithMetaHash is GetRevisionMessage with metaHash in place of the SHA-256 of enc_meta.
func GetRevisionMessageWithMetaHash(_ *syncv1.Revision, _ [32]byte) []byte {
	return nil
}

// GetRevisionIDWithMetaHash is the SHA-256 of GetRevisionMessageWithMetaHash.
func GetRevisionIDWithMetaHash(_ *syncv1.Revision, _ [32]byte) [32]byte {
	return [32]byte{}
}

// ErrInvalidShareLogEntry means a share log entry has a shape or a size the protocol does not allow.
var ErrInvalidShareLogEntry = errors.New("the share log entry is invalid")

// CheckShareLogEntry checks the shape, the kind rules and the sizes of a share log entry, not its signature or its
// place in the log.
func CheckShareLogEntry(_ *syncv1.ShareLogEntry) error {
	return errors.New("not implemented")
}

// CheckShareRevision is CheckRevision for a change served to a reader: enc_meta may be empty when the file left, and
// the hash of enc_meta is required.
func CheckShareRevision(_ *syncv1.ShareChange) error {
	return errors.New("not implemented")
}
