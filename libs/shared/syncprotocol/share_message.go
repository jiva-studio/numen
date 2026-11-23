package syncprotocol

import (
	"crypto/sha256"
	"encoding/binary"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const shareLogLabel = "numen-sync/share-log/v1"

// GetShareLogEntryMessage returns the bytes a device signs to append a share log entry: the label
// numen-sync/share-log/v1, a zero byte, share_root_id, vault_id, prev_hash, subject_root_sign_key and
// author_device_sign_key as a uvarint length and bytes, version (8 bytes), epoch (4 bytes) and kind (1 byte), the
// SHA-256 of enc_owner_body, enc_share_body and prev_key_seal as a length and bytes, the count of seals (4 bytes)
// and for each seal reader_root_sign_key and the SHA-256 of sealed as a length and bytes. The signature is never part of it.
func GetShareLogEntryMessage(entry *syncv1.ShareLogEntry) []byte {
	message := append([]byte(shareLogLabel), 0)
	for _, field := range [][]byte{
		entry.GetShareRootId(), entry.GetVaultId(), entry.GetPrevHash(), entry.GetSubjectRootSignKey(), entry.GetAuthorDeviceSignKey(),
	} {
		message = appendField(message, field)
	}
	message = binary.BigEndian.AppendUint64(message, entry.GetVersion())
	message = binary.BigEndian.AppendUint32(message, entry.GetEpoch())
	message = append(message, byte(entry.GetKind()))
	for _, body := range [][]byte{entry.GetEncOwnerBody(), entry.GetEncShareBody(), entry.GetPrevKeySeal()} {
		sum := sha256.Sum256(body)
		message = appendField(message, sum[:])
	}
	message = binary.BigEndian.AppendUint32(message, uint32(len(entry.GetSeals())))
	for _, seal := range entry.GetSeals() {
		sum := sha256.Sum256(seal.GetSealed())
		message = appendField(message, seal.GetReaderRootSignKey())
		message = appendField(message, sum[:])
	}
	return message
}

// GetShareLogEntryID returns the id of a share log entry: the SHA-256 of GetShareLogEntryMessage. The prev_hash of
// the next entry is this value.
func GetShareLogEntryID(entry *syncv1.ShareLogEntry) [32]byte {
	return sha256.Sum256(GetShareLogEntryMessage(entry))
}
