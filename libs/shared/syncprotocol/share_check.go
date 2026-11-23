package syncprotocol

import (
	"bytes"
	"errors"
	"fmt"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const maxSealedSize = 256

// ErrInvalidShareLogEntry means a share log entry has a shape or a size the protocol does not allow.
var ErrInvalidShareLogEntry = errors.New("the share log entry is invalid")

// CheckShareLogEntry checks the shape, the kind rules and the sizes of a share log entry, not its signature or its
// place in the log.
func CheckShareLogEntry(entry *syncv1.ShareLogEntry) error {
	if entry == nil {
		return fmt.Errorf("%w: it is missing", ErrInvalidShareLogEntry)
	}
	if entry.GetVersion() == 0 {
		return fmt.Errorf("%w: version is 0", ErrInvalidShareLogEntry)
	}
	if err := checkShareLogSizes(entry); err != nil {
		return err
	}
	if err := checkShareSeals(entry); err != nil {
		return err
	}
	return checkShareLogKind(entry)
}

func checkShareLogSizes(entry *syncv1.ShareLogEntry) error {
	previous := []int{hashSize}
	if entry.GetVersion() == 1 {
		previous = []int{0}
	}
	if len(entry.GetEncOwnerBody()) > MaxEncSize {
		return fmt.Errorf("%w: enc_owner_body has %d bytes", ErrInvalidShareLogEntry, len(entry.GetEncOwnerBody()))
	}
	if len(entry.GetEncShareBody()) > MaxEncShareBody {
		return fmt.Errorf("%w: enc_share_body has %d bytes", ErrInvalidShareLogEntry, len(entry.GetEncShareBody()))
	}
	return checkSizes(ErrInvalidShareLogEntry, []sizedField{
		{"share_root_id", entry.GetShareRootId(), []int{idSize}},
		{"prev_hash", entry.GetPrevHash(), previous},
		{"vault_id", entry.GetVaultId(), []int{0, idSize}},
		{"subject_root_sign_key", entry.GetSubjectRootSignKey(), []int{0, signKeySize}},
		{"author_device_sign_key", entry.GetAuthorDeviceSignKey(), []int{signKeySize}},
		{"signature", entry.GetSignature(), []int{signatureSize}},
	})
}

func checkShareSeals(entry *syncv1.ShareLogEntry) error {
	if len(entry.GetSeals()) > MaxShareSeals {
		return fmt.Errorf("%w: %d seals", ErrInvalidShareLogEntry, len(entry.GetSeals()))
	}
	for _, seal := range entry.GetSeals() {
		if len(seal.GetReaderRootSignKey()) != signKeySize {
			return fmt.Errorf("%w: reader_root_sign_key has %d bytes", ErrInvalidShareLogEntry, len(seal.GetReaderRootSignKey()))
		}
		if len(seal.GetSealed()) == 0 || len(seal.GetSealed()) > maxSealedSize {
			return fmt.Errorf("%w: sealed has %d bytes", ErrInvalidShareLogEntry, len(seal.GetSealed()))
		}
	}
	return nil
}

func checkShareLogKind(entry *syncv1.ShareLogEntry) error {
	kind := entry.GetKind()
	if kind < syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE || kind > syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE {
		return fmt.Errorf("%w: kind is %d", ErrInvalidShareLogEntry, kind)
	}
	isCreate := kind == syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE
	isAdd := kind == syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER
	isRemove := kind == syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER
	startsEpoch := isRemove || kind == syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE
	var problem string
	switch {
	case isCreate != (entry.GetVersion() == 1):
		problem = "only a create entry is version 1"
	case isCreate != (len(entry.GetVaultId()) != 0):
		problem = "only a create entry has vault_id"
	case (isAdd || isRemove) != (len(entry.GetSubjectRootSignKey()) != 0):
		problem = "only an add or remove entry has a subject"
	case startsEpoch != (len(entry.GetPrevKeySeal()) != 0):
		problem = "only an entry that starts an epoch has prev_key_seal"
	case (isCreate && entry.GetEpoch() != 0) || (startsEpoch && entry.GetEpoch() < 1):
		problem = "epoch does not fit the kind"
	default:
		problem = getShareSealProblem(entry, isAdd, startsEpoch)
	}
	if problem != "" {
		return fmt.Errorf("%w: %s", ErrInvalidShareLogEntry, problem)
	}
	return nil
}

func getShareSealProblem(entry *syncv1.ShareLogEntry, isAdd, startsEpoch bool) string {
	switch {
	case isAdd && len(entry.GetSeals()) != 1:
		return "an add entry has one seal"
	case isAdd && !bytes.Equal(entry.GetSeals()[0].GetReaderRootSignKey(), entry.GetSubjectRootSignKey()):
		return "the seal of an add entry is for another reader"
	case !isAdd && !startsEpoch && len(entry.GetSeals()) != 0:
		return "this entry has no seal"
	}
	return ""
}

// CheckShareRevision is CheckRevision for a change served to a reader: enc_meta may be empty when the file left, and
// the hash of enc_meta is required.
func CheckShareRevision(change *syncv1.ShareChange) error {
	if change == nil || change.GetChange() == nil {
		return fmt.Errorf("%w: it is missing", ErrInvalidRevision)
	}
	if len(change.GetEncMetaHash()) != hashSize {
		return fmt.Errorf("%w: enc_meta_hash has %d bytes", ErrInvalidRevision, len(change.GetEncMetaHash()))
	}
	return checkRevision(change.GetChange().GetRevision(), change.GetLeft())
}
