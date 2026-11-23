package syncprotocol_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func newShareLogEntry() *syncv1.ShareLogEntry {
	return &syncv1.ShareLogEntry{
		ShareRootId:         bytes.Repeat([]byte{1}, 16),
		Version:             3,
		PrevHash:            bytes.Repeat([]byte{2}, 32),
		Kind:                syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER,
		VaultId:             bytes.Repeat([]byte{3}, 16),
		Epoch:               2,
		SubjectRootSignKey:  bytes.Repeat([]byte{4}, 32),
		EncOwnerBody:        []byte("owner body"),
		EncShareBody:        []byte("share body"),
		Seals:               []*syncv1.ShareSeal{{ReaderRootSignKey: bytes.Repeat([]byte{4}, 32), Sealed: []byte("sealed to the reader")}},
		PrevKeySeal:         []byte("previous key"),
		AuthorDeviceSignKey: bytes.Repeat([]byte{5}, 32),
		Signature:           bytes.Repeat([]byte{6}, 64),
	}
}

func appendUvarintField(message, field []byte) []byte {
	return append(binary.AppendUvarint(message, uint64(len(field))), field...)
}

func wantShareLogEntryMessage(entry *syncv1.ShareLogEntry) []byte {
	message := append([]byte("numen-sync/share-log/v1"), 0)
	for _, field := range [][]byte{
		entry.GetShareRootId(), entry.GetVaultId(), entry.GetPrevHash(), entry.GetSubjectRootSignKey(), entry.GetAuthorDeviceSignKey(),
	} {
		message = appendUvarintField(message, field)
	}
	message = binary.BigEndian.AppendUint64(message, entry.GetVersion())
	message = binary.BigEndian.AppendUint32(message, entry.GetEpoch())
	message = append(message, byte(entry.GetKind()))
	for _, body := range [][]byte{entry.GetEncOwnerBody(), entry.GetEncShareBody(), entry.GetPrevKeySeal()} {
		sum := sha256.Sum256(body)
		message = appendUvarintField(message, sum[:])
	}
	message = binary.BigEndian.AppendUint32(message, uint32(len(entry.GetSeals())))
	for _, seal := range entry.GetSeals() {
		sum := sha256.Sum256(seal.GetSealed())
		message = appendUvarintField(message, seal.GetReaderRootSignKey())
		message = appendUvarintField(message, sum[:])
	}
	return message
}

func TestGetShareLogEntryMessage_ProducesTheFrozenLayout(t *testing.T) {
	t.Parallel()
	entry := newShareLogEntry()

	got := syncprotocol.GetShareLogEntryMessage(entry)

	if want := wantShareLogEntryMessage(entry); !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
}

func TestGetShareLogEntryMessage_OfAnEntryWithoutSealsEndsWithAZeroCount(t *testing.T) {
	t.Parallel()
	entry := newShareLogEntry()
	entry.Kind, entry.Seals, entry.SubjectRootSignKey, entry.PrevKeySeal = syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, nil, nil, nil

	got := syncprotocol.GetShareLogEntryMessage(entry)

	if want := wantShareLogEntryMessage(entry); !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
	if !bytes.HasSuffix(got, []byte{0, 0, 0, 0}) {
		t.Errorf("message ends with %x, want a zero count of seals", got[len(got)-4:])
	}
}

func TestGetShareLogEntryMessage_ChangesWithEveryFieldItCovers(t *testing.T) {
	t.Parallel()
	base := syncprotocol.GetShareLogEntryMessage(newShareLogEntry())

	for name, change := range map[string]func(*syncv1.ShareLogEntry){
		"share root":    func(e *syncv1.ShareLogEntry) { e.GetShareRootId()[0] ^= 1 },
		"version":       func(e *syncv1.ShareLogEntry) { e.Version++ },
		"previous hash": func(e *syncv1.ShareLogEntry) { e.GetPrevHash()[0] ^= 1 },
		"kind":          func(e *syncv1.ShareLogEntry) { e.Kind = syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER },
		"vault":         func(e *syncv1.ShareLogEntry) { e.GetVaultId()[0] ^= 1 },
		"epoch":         func(e *syncv1.ShareLogEntry) { e.Epoch++ },
		"subject":       func(e *syncv1.ShareLogEntry) { e.GetSubjectRootSignKey()[0] ^= 1 },
		"owner body":    func(e *syncv1.ShareLogEntry) { e.EncOwnerBody = []byte("owner bodz") },
		"share body":    func(e *syncv1.ShareLogEntry) { e.EncShareBody = []byte("share bodz") },
		"previous key":  func(e *syncv1.ShareLogEntry) { e.PrevKeySeal = []byte("previous kez") },
		"author":        func(e *syncv1.ShareLogEntry) { e.GetAuthorDeviceSignKey()[0] ^= 1 },
		"seal reader":   func(e *syncv1.ShareLogEntry) { e.GetSeals()[0].ReaderRootSignKey[0] ^= 1 },
		"seal bytes":    func(e *syncv1.ShareLogEntry) { e.GetSeals()[0].Sealed = []byte("sealed to the readeq") },
		"extra seal": func(e *syncv1.ShareLogEntry) {
			e.Seals = append(e.Seals, &syncv1.ShareSeal{ReaderRootSignKey: bytes.Repeat([]byte{9}, 32), Sealed: []byte("x")})
		},
	} {
		changed := newShareLogEntry()
		change(changed)
		if bytes.Equal(syncprotocol.GetShareLogEntryMessage(changed), base) {
			t.Errorf("a change of the %s leaves the message as it was", name)
		}
	}
}

func TestGetShareLogEntryMessage_IgnoresTheSignature(t *testing.T) {
	t.Parallel()
	signed := newShareLogEntry()
	signed.Signature = bytes.Repeat([]byte{0xAA}, 64)

	if !bytes.Equal(syncprotocol.GetShareLogEntryMessage(signed), syncprotocol.GetShareLogEntryMessage(newShareLogEntry())) {
		t.Error("the signature is part of the signed bytes")
	}
}

func TestGetShareLogEntryMessage_KeepsTheOrderOfTheSeals(t *testing.T) {
	t.Parallel()
	first := &syncv1.ShareSeal{ReaderRootSignKey: bytes.Repeat([]byte{7}, 32), Sealed: []byte("one")}
	second := &syncv1.ShareSeal{ReaderRootSignKey: bytes.Repeat([]byte{8}, 32), Sealed: []byte("two")}
	forward, backward := newShareLogEntry(), newShareLogEntry()
	forward.Seals = []*syncv1.ShareSeal{first, second}
	backward.Seals = []*syncv1.ShareSeal{second, first}

	if bytes.Equal(syncprotocol.GetShareLogEntryMessage(forward), syncprotocol.GetShareLogEntryMessage(backward)) {
		t.Error("two orders of the seals have one message")
	}
}

func TestGetShareLogEntryID_IsTheSHA256OfTheMessage(t *testing.T) {
	t.Parallel()
	entry := newShareLogEntry()

	got := syncprotocol.GetShareLogEntryID(entry)

	if want := sha256.Sum256(wantShareLogEntryMessage(entry)); got != want {
		t.Errorf("id = %x, want %x", got, want)
	}
}

func TestGetRevisionMessageWithMetaHash_IsTheRevisionMessageForTheHashOfItsMeta(t *testing.T) {
	t.Parallel()
	revision := newRevision()
	metaHash := sha256.Sum256(revision.GetEncMeta())

	got := syncprotocol.GetRevisionMessageWithMetaHash(revision, metaHash)

	if !bytes.Equal(got, syncprotocol.GetRevisionMessage(revision)) {
		t.Error("the message with the hash of the meta differs from GetRevisionMessage")
	}
	if id, want := syncprotocol.GetRevisionIDWithMetaHash(revision, metaHash), syncprotocol.GetRevisionID(revision); id != want {
		t.Errorf("id = %x, want %x", id, want)
	}
}

func TestGetRevisionMessageWithMetaHash_VerifiesARevisionWhoseMetaWasWithheld(t *testing.T) {
	t.Parallel()
	full := newRevision()
	metaHash := sha256.Sum256(full.GetEncMeta())
	stripped := newRevision()
	stripped.EncMeta = nil

	got := syncprotocol.GetRevisionMessageWithMetaHash(stripped, metaHash)

	if !bytes.Equal(got, syncprotocol.GetRevisionMessage(full)) {
		t.Error("a stripped revision with the hash does not give the message the owner signed")
	}
	if id, want := syncprotocol.GetRevisionIDWithMetaHash(stripped, metaHash), syncprotocol.GetRevisionID(full); id != want {
		t.Errorf("id = %x, want %x", id, want)
	}
}

func TestGetRevisionMessageWithMetaHash_ChangesWithTheHash(t *testing.T) {
	t.Parallel()
	revision := newRevision()
	first := sha256.Sum256([]byte("one"))
	second := sha256.Sum256([]byte("two"))

	if bytes.Equal(syncprotocol.GetRevisionMessageWithMetaHash(revision, first), syncprotocol.GetRevisionMessageWithMetaHash(revision, second)) {
		t.Error("two hashes of the meta give one message")
	}
}
