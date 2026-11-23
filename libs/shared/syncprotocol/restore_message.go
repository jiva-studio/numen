package syncprotocol

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const restoreEpochLabel = "numen-sync/restore-epoch/v1"

// ErrInvalidRestoreEpoch means a restore epoch has a size the protocol does not allow or a signature that does
// not verify.
var ErrInvalidRestoreEpoch = errors.New("the restore epoch is invalid")

// GetRestoreEpochMessage returns the bytes the server signs to announce a restore: the label of the protocol, a
// zero byte, the 16 bytes of id, then issuedAt as 8 bytes big endian.
func GetRestoreEpochMessage(id []byte, issuedAt int64) []byte {
	message := append([]byte(restoreEpochLabel), 0)
	message = append(message, id...)
	return binary.BigEndian.AppendUint64(message, uint64(issuedAt))
}

// CheckRestoreEpoch checks the sizes of the epoch and its signature against the key of the server.
func CheckRestoreEpoch(key []byte, epoch *syncv1.RestoreEpoch) error {
	if epoch == nil {
		return fmt.Errorf("%w: it is missing", ErrInvalidRestoreEpoch)
	}
	err := checkSizes(ErrInvalidRestoreEpoch, []sizedField{
		{"epoch_sign_key", key, []int{signKeySize}},
		{"id", epoch.GetId(), []int{idSize}},
		{"signature", epoch.GetSignature(), []int{signatureSize}},
	})
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, GetRestoreEpochMessage(epoch.GetId(), epoch.GetIssuedAt()), epoch.GetSignature()) {
		return fmt.Errorf("%w: the signature does not verify", ErrInvalidRestoreEpoch)
	}
	return nil
}
