package syncprotocol

import (
	"crypto/sha256"
	"encoding/binary"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	revisionLabel = "numen-sync/revision/v1"
	catalogLabel  = "numen-sync/catalog/v1"
)

// GetRevisionMessage returns the bytes a device signs to commit a revision: the label of the protocol, a
// zero byte, the fields vault_id, file_id, parent_rev, blob_id, blob_hash, SHA-256 of enc_meta, path_hash,
// share_root_id and author_device_sign_key, each as a uvarint length and its bytes, then blob_size as 8 bytes,
// file_version as 8 bytes, epoch as 4 bytes (all big endian) and deleted as one byte. The signature is not part of it.
func GetRevisionMessage(revision *syncv1.Revision) []byte {
	return GetRevisionMessageWithMetaHash(revision, sha256.Sum256(revision.GetEncMeta()))
}

// GetRevisionMessageWithMetaHash is GetRevisionMessage with metaHash in place of the SHA-256 of enc_meta, so a reader
// can verify a revision whose enc_meta the server withheld.
func GetRevisionMessageWithMetaHash(revision *syncv1.Revision, metaHash [32]byte) []byte {
	message := append([]byte(revisionLabel), 0)
	for _, field := range [][]byte{
		revision.GetVaultId(), revision.GetFileId(), revision.GetParentRev(), revision.GetBlobId(), revision.GetBlobHash(),
		metaHash[:], revision.GetPathHash(), revision.GetShareRootId(), revision.GetAuthorDeviceSignKey(),
	} {
		message = appendField(message, field)
	}
	message = binary.BigEndian.AppendUint64(message, revision.GetBlobSize())
	message = binary.BigEndian.AppendUint64(message, revision.GetFileVersion())
	message = binary.BigEndian.AppendUint32(message, revision.GetEpoch())
	return append(message, encodeBool(revision.GetDeleted()))
}

// GetRevisionID returns the id of a revision: the SHA-256 of GetRevisionMessage. The parent_rev of the next
// revision of the file is this value.
func GetRevisionID(revision *syncv1.Revision) [32]byte {
	return sha256.Sum256(GetRevisionMessage(revision))
}

// GetRevisionIDWithMetaHash is the SHA-256 of GetRevisionMessageWithMetaHash.
func GetRevisionIDWithMetaHash(revision *syncv1.Revision, metaHash [32]byte) [32]byte {
	return sha256.Sum256(GetRevisionMessageWithMetaHash(revision, metaHash))
}

// GetCatalogEntryMessage returns the bytes a device signs to append a catalog entry: the label of the
// protocol, a zero byte, version as 8 bytes big endian, then prev_hash and vault_id as uvarint length and
// bytes, creates_vault as one byte, SHA-256 of enc_body and author_device_sign_key as uvarint length and bytes.
func GetCatalogEntryMessage(entry *syncv1.CatalogEntry) []byte {
	bodyHash := sha256.Sum256(entry.GetEncBody())
	message := append([]byte(catalogLabel), 0)
	message = binary.BigEndian.AppendUint64(message, entry.GetVersion())
	message = appendField(message, entry.GetPrevHash())
	message = appendField(message, entry.GetVaultId())
	message = append(message, encodeBool(entry.GetCreatesVault()))
	message = appendField(message, bodyHash[:])
	return appendField(message, entry.GetAuthorDeviceSignKey())
}

// GetCatalogEntryID returns the id of a catalog entry: the SHA-256 of GetCatalogEntryMessage. The prev_hash
// of the next entry is this value.
func GetCatalogEntryID(entry *syncv1.CatalogEntry) [32]byte {
	return sha256.Sum256(GetCatalogEntryMessage(entry))
}

func appendField(message, field []byte) []byte {
	return append(binary.AppendUvarint(message, uint64(len(field))), field...)
}

func encodeBool(value bool) byte {
	if value {
		return 1
	}
	return 0
}
