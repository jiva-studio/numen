package syncprotocol

import (
	"errors"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// ErrInvalidRestoreEpoch means a restore epoch has a size the protocol does not allow or a signature that does
// not verify.
var ErrInvalidRestoreEpoch = errors.New("the restore epoch is invalid")

// GetRestoreEpochMessage returns the bytes the server signs to announce a restore: the label of the protocol, a
// zero byte, the 16 bytes of id, then issuedAt as 8 bytes big endian.
func GetRestoreEpochMessage(id []byte, issuedAt int64) []byte {
	return nil
}

// CheckRestoreEpoch checks the sizes of the epoch and its signature against the key of the server.
func CheckRestoreEpoch(key []byte, epoch *syncv1.RestoreEpoch) error {
	return errors.New("not implemented")
}
