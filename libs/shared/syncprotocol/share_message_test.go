package syncprotocol_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	entry.Kind = syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE
	entry.VaultId, entry.SubjectRootSignKey = nil, nil
	entry.Seals = []*syncv1.ShareSeal{
		{ReaderRootSignKey: bytes.Repeat([]byte{8}, 32), Sealed: []byte("two")},
		{ReaderRootSignKey: bytes.Repeat([]byte{7}, 32), Sealed: []byte("one")},
	}
	want := readHex(t,
		"6e756d656e2d73796e632f73686172652d6c6f672f76310010010101010101010101010101010101010020020202"+
			"02020202020202020202020202020202020202020202020202020202020020050505050505050505050505050505"+
			"05050505050505050505050505050505050000000000000003000000020420c09e34fb3a1ae9d0deda8af4cc7d88"+
			"352eff848e8c61e884f112f1c7fb8600b6207e24b666ca3cb7f00150e8d74dc42958df54f0ae8c834a41244281fb"+
			"01caa340206cd6465450478b08e9822c6fff4653c0d5b21c8aae379f5f021f57e9ec51b479000000022008080808"+
			"08080808080808080808080808080808080808080808080808080808203fc4ccfe745870e2c0d99f71f30ff0656c"+
			"8dedd41cc1d7d3d376b0dbe685e2f320070707070707070707070707070707070707070707070707070707070707"+
			"0707207692c3ad3540bb803c020b3aee66cd8887123234ea0c6e7143c0add73ff431ed")

	got := syncprotocol.GetShareLogEntryMessage(entry)

	if !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
	id := syncprotocol.GetShareLogEntryID(entry)
	if wantID := "dee0301fc809ba2a0c651dc9514a44a298359ecc302a691e291ed5ba12674054"; hex.EncodeToString(id[:]) != wantID {
		t.Errorf("id = %x, want %s", id, wantID)
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

func TestGetShareLogEntryMessage_ProducesTheFrozenLayoutForAnAddReader(t *testing.T) {
	t.Parallel()
	entry := newShareLogEntry()
	want := readHex(t,
		"6e756d656e2d73796e632f73686172652d6c6f672f76310010010101010101010101010101010101011003030303"+
			"03030303030303030303030320020202020202020202020202020202020202020202020202020202020202020220"+
			"04040404040404040404040404040404040404040404040404040404040404042005050505050505050505050505"+
			"050505050505050505050505050505050505050000000000000003000000020220c09e34fb3a1ae9d0deda8af4cc"+
			"7d88352eff848e8c61e884f112f1c7fb8600b6207e24b666ca3cb7f00150e8d74dc42958df54f0ae8c834a412442"+
			"81fb01caa340206cd6465450478b08e9822c6fff4653c0d5b21c8aae379f5f021f57e9ec51b47900000001200404"+
			"0404040404040404040404040404040404040404040404040404040404042032d8542e1cfe183be6459d2fd0eeb3"+
			"e36d38209908a6b218393b31d015247cf4")

	if got := syncprotocol.GetShareLogEntryMessage(entry); !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
	id := syncprotocol.GetShareLogEntryID(entry)
	if wantID := "e82de1311c32d1807d9d46cfea9b3426e7f37457dff350130b5ada6c9849f15b"; hex.EncodeToString(id[:]) != wantID {
		t.Errorf("id = %x, want %s", id, wantID)
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
	full.ShareRootId = bytes.Repeat([]byte{9}, 16)
	full.FileVersion = 0x0102030405060708
	full.Epoch = 0x11223344
	want := readHex(t,
		"6e756d656e2d73796e632f7265766973696f6e2f7631001001010101010101010101010101010101100202020202"+
			"02020202020202020202022003030303030303030303030303030303030303030303030303030303030303031004"+
			"04040404040404040404040404040420050505050505050505050505050505050505050505050505050505050505"+
			"050520ea3bd73e2b506e00527232b3ed743c066da83a8e3066f62a71e75eb9b4aa1db62006060606060606060606"+
			"06060606060606060606060606060606060606060606100909090909090909090909090909090920070707070707"+
			"07070707070707070707070707070707070707070707070707070000000000001000010203040506070811223344"+
			"00")
	const wantID = "970126b767024dbb6a3f0a04456f8c1999191a286eca685ce8bc764c98af45fc"
	metaHash := [32]byte(readHex(t, "ea3bd73e2b506e00527232b3ed743c066da83a8e3066f62a71e75eb9b4aa1db6"))

	if got := syncprotocol.GetRevisionMessage(full); !bytes.Equal(got, want) {
		t.Errorf("full message = %x\nwant           %x", got, want)
	}
	if id := syncprotocol.GetRevisionID(full); hex.EncodeToString(id[:]) != wantID {
		t.Errorf("full id = %x, want %s", id, wantID)
	}

	full.EncMeta = nil
	if got := syncprotocol.GetRevisionMessageWithMetaHash(full, metaHash); !bytes.Equal(got, want) {
		t.Errorf("message with withheld metadata = %x\nwant                             %x", got, want)
	}
	if id := syncprotocol.GetRevisionIDWithMetaHash(full, metaHash); hex.EncodeToString(id[:]) != wantID {
		t.Errorf("id with withheld metadata = %x, want %s", id, wantID)
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
