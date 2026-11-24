package syncprotocol_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func newRevision() *syncv1.Revision {
	return &syncv1.Revision{
		VaultId:             bytes.Repeat([]byte{1}, 16),
		FileId:              bytes.Repeat([]byte{2}, 16),
		ParentRev:           bytes.Repeat([]byte{3}, 32),
		BlobId:              bytes.Repeat([]byte{4}, 16),
		BlobHash:            bytes.Repeat([]byte{5}, 32),
		BlobSize:            4096,
		EncMeta:             []byte("meta"),
		PathHash:            bytes.Repeat([]byte{6}, 32),
		AuthorDeviceSignKey: bytes.Repeat([]byte{7}, 32),
		Signature:           bytes.Repeat([]byte{8}, 64),
	}
}

func newTombstone() *syncv1.Revision {
	return &syncv1.Revision{
		VaultId:             bytes.Repeat([]byte{1}, 16),
		FileId:              bytes.Repeat([]byte{2}, 16),
		ParentRev:           bytes.Repeat([]byte{3}, 32),
		PathHash:            bytes.Repeat([]byte{6}, 32),
		Deleted:             true,
		AuthorDeviceSignKey: bytes.Repeat([]byte{7}, 32),
		Signature:           bytes.Repeat([]byte{8}, 64),
	}
}

func newCatalogEntry() *syncv1.CatalogEntry {
	return &syncv1.CatalogEntry{
		Version:             3,
		PrevHash:            bytes.Repeat([]byte{9}, 32),
		VaultId:             bytes.Repeat([]byte{1}, 16),
		CreatesVault:        true,
		EncBody:             []byte("body"),
		AuthorDeviceSignKey: bytes.Repeat([]byte{7}, 32),
		Signature:           bytes.Repeat([]byte{8}, 64),
	}
}

func readHex(tb testing.TB, text string) []byte {
	tb.Helper()
	decoded, err := hex.DecodeString(text)
	if err != nil {
		tb.Fatal(err)
	}
	return decoded
}

func TestGetRevisionMessage_ProducesTheFrozenBytes(t *testing.T) {
	t.Parallel()
	want := readHex(t,
		"6e756d656e2d73796e632f7265766973696f6e2f76310010010101010101010101010101010101011002020202020202"+
			"020202020202020202200303030303030303030303030303030303030303030303030303030303030303100404040404"+
			"040404040404040404040420050505050505050505050505050505050505050505050505050505050505050520ea3bd7"+
			"3e2b506e00527232b3ed743c066da83a8e3066f62a71e75eb9b4aa1db620060606060606060606060606060606060606"+
			"060606060606060606060606060600200707070707070707070707070707070707070707070707070707070707070707"+
			"00000000000010000000000000")

	got := syncprotocol.GetRevisionMessage(newRevision())

	if !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
}

func TestGetRevisionID_IsTheSHA256OfTheMessage(t *testing.T) {
	t.Parallel()

	got := syncprotocol.GetRevisionID(newRevision())

	if want := "72ff3f6d63bcbb16ac98b68e02f06025b38a062a55b164837ac5cbd9073d70d9"; hex.EncodeToString(got[:]) != want {
		t.Errorf("id = %x, want %s", got, want)
	}
}

func TestGetRevisionMessage_ChangesWithEveryFieldItCovers(t *testing.T) {
	t.Parallel()
	base := syncprotocol.GetRevisionMessage(newRevision())

	for name, change := range map[string]func(*syncv1.Revision){
		"vault id":   func(r *syncv1.Revision) { r.GetVaultId()[0] ^= 1 },
		"file id":    func(r *syncv1.Revision) { r.GetFileId()[0] ^= 1 },
		"parent":     func(r *syncv1.Revision) { r.GetParentRev()[0] ^= 1 },
		"blob id":    func(r *syncv1.Revision) { r.GetBlobId()[0] ^= 1 },
		"blob hash":  func(r *syncv1.Revision) { r.GetBlobHash()[0] ^= 1 },
		"blob size":  func(r *syncv1.Revision) { r.BlobSize++ },
		"meta":       func(r *syncv1.Revision) { r.EncMeta = []byte("metb") },
		"path hash":  func(r *syncv1.Revision) { r.GetPathHash()[0] ^= 1 },
		"share root": func(r *syncv1.Revision) { r.ShareRootId = bytes.Repeat([]byte{8}, 16) },
		"deleted":    func(r *syncv1.Revision) { r.Deleted = true },
		"epoch":      func(r *syncv1.Revision) { r.Epoch = 1 },
		"author":     func(r *syncv1.Revision) { r.GetAuthorDeviceSignKey()[0] ^= 1 },
	} {
		changed := newRevision()
		change(changed)
		if bytes.Equal(syncprotocol.GetRevisionMessage(changed), base) {
			t.Errorf("a change of the %s leaves the message as it was", name)
		}
	}
}

func TestGetRevisionMessage_IgnoresTheSignature(t *testing.T) {
	t.Parallel()
	signed := newRevision()
	signed.Signature = bytes.Repeat([]byte{0xAA}, 64)

	if !bytes.Equal(syncprotocol.GetRevisionMessage(signed), syncprotocol.GetRevisionMessage(newRevision())) {
		t.Error("the signature is part of the signed bytes")
	}
}

func TestGetRevisionMessage_KeepsTheBoundaryBetweenNeighboringFields(t *testing.T) {
	t.Parallel()
	first := newRevision()
	first.ParentRev = bytes.Repeat([]byte{3}, 32)
	first.BlobId = nil
	second := newRevision()
	second.ParentRev = bytes.Repeat([]byte{3}, 16)
	second.BlobId = bytes.Repeat([]byte{3}, 16)

	if bytes.Equal(syncprotocol.GetRevisionMessage(first), syncprotocol.GetRevisionMessage(second)) {
		t.Error("two revisions that split the same bytes between two fields differently have one message")
	}
}

func TestGetCatalogEntryMessage_ProducesTheFrozenBytes(t *testing.T) {
	t.Parallel()
	want := readHex(t,
		"6e756d656e2d73796e632f636174616c6f672f7631000000000000000003200909090909090909090909090909090909"+
			"09090909090909090909090909090910010101010101010101010101010101010120230d8358dc8e8890b4c58deeb629"+
			"12ee2f20357ae92a5cc861b98e68fe31acb5200707070707070707070707070707070707070707070707070707070707"+
			"070707")

	got := syncprotocol.GetCatalogEntryMessage(newCatalogEntry())

	if !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
}

func TestGetCatalogEntryMessage_OfTheFirstEntryHasNoPreviousHash(t *testing.T) {
	t.Parallel()
	first := newCatalogEntry()
	first.Version, first.PrevHash, first.CreatesVault = 1, nil, false
	want := readHex(t,
		"6e756d656e2d73796e632f636174616c6f672f7631000000000000000001001001010101010101010101010101010101"+
			"0020230d8358dc8e8890b4c58deeb62912ee2f20357ae92a5cc861b98e68fe31acb52007070707070707070707070707"+
			"07070707070707070707070707070707070707")

	got := syncprotocol.GetCatalogEntryMessage(first)

	if !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
}

func TestGetCatalogEntryID_IsTheSHA256OfTheMessage(t *testing.T) {
	t.Parallel()
	got := syncprotocol.GetCatalogEntryID(newCatalogEntry())
	if want := "b443e8f7ff6c0fb4dbae46e919e5e930672505092daaf7d41594cee6d8bc6082"; hex.EncodeToString(got[:]) != want {
		t.Errorf("id = %x, want %s", got, want)
	}
	first := newCatalogEntry()
	first.Version, first.PrevHash, first.CreatesVault = 1, nil, false
	gotFirst := syncprotocol.GetCatalogEntryID(first)
	if want := "43e5d1b017622835ef752f1e6b07a9ea1417694a7b89400380475a339ca15090"; hex.EncodeToString(gotFirst[:]) != want {
		t.Errorf("first id = %x, want %s", gotFirst, want)
	}
}

func TestGetCatalogEntryMessage_ChangesWithEveryFieldItCovers(t *testing.T) {
	t.Parallel()
	base := syncprotocol.GetCatalogEntryMessage(newCatalogEntry())

	for name, change := range map[string]func(*syncv1.CatalogEntry){
		"version":       func(e *syncv1.CatalogEntry) { e.Version++ },
		"previous hash": func(e *syncv1.CatalogEntry) { e.GetPrevHash()[0] ^= 1 },
		"vault id":      func(e *syncv1.CatalogEntry) { e.GetVaultId()[0] ^= 1 },
		"creates vault": func(e *syncv1.CatalogEntry) { e.CreatesVault = false },
		"body":          func(e *syncv1.CatalogEntry) { e.EncBody = []byte("bodz") },
		"author":        func(e *syncv1.CatalogEntry) { e.GetAuthorDeviceSignKey()[0] ^= 1 },
	} {
		changed := newCatalogEntry()
		change(changed)
		if bytes.Equal(syncprotocol.GetCatalogEntryMessage(changed), base) {
			t.Errorf("a change of the %s leaves the message as it was", name)
		}
	}
	signed := newCatalogEntry()
	signed.Signature = bytes.Repeat([]byte{0xAA}, 64)
	if !bytes.Equal(syncprotocol.GetCatalogEntryMessage(signed), base) {
		t.Error("the signature is part of the signed bytes")
	}
}

func TestGetChainReadMessage_ProducesTheFrozenBytes(t *testing.T) {
	t.Parallel()
	challenge := &syncv1.GetChallengeResponse{
		ChallengeId: bytes.Repeat([]byte{1}, 16), Nonce: bytes.Repeat([]byte{2}, 32),
		ExpiresAt: 1_795_000_000, Audience: "https://sync.numen.md",
	}
	want := readHex(t,
		"6e756d656e2d73796e632f636861696e2d726561642f7631001568747470733a2f2f73796e632e6e756d656e2e6d6402"+
			"02020202020202020202020202020202020202020202020202020202020202000000006afd86c0030303030303030303"+
			"030303030303030303030303030303030303030303030300000001")

	got := syncprotocol.GetChainReadMessage(challenge, bytes.Repeat([]byte{3}, 32), 1)

	if !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
}

func TestGetChainReadMessage_DiffersFromTheSignInMessageOfTheSameInputs(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte{3}, 32)

	if bytes.Equal(syncprotocol.GetChainReadMessage(newChallenge(), key, 1), syncprotocol.GetSignInMessage(newChallenge(), key, 1)) {
		t.Error("a signature of a sign-in is also a signature of a chain read")
	}
}

func TestGetChainReadMessage_ChangesWithEveryFieldItCovers(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte{3}, 32)
	base := syncprotocol.GetChainReadMessage(newChallenge(), key, 1)

	for name, message := range map[string][]byte{
		"audience": syncprotocol.GetChainReadMessage(withChallenge(func(c *syncv1.GetChallengeResponse) { c.Audience = "https://other.example" }), key, 1),
		"nonce":    syncprotocol.GetChainReadMessage(withChallenge(func(c *syncv1.GetChallengeResponse) { c.Nonce[0] ^= 1 }), key, 1),
		"expiry":   syncprotocol.GetChainReadMessage(withChallenge(func(c *syncv1.GetChallengeResponse) { c.ExpiresAt++ }), key, 1),
		"key":      syncprotocol.GetChainReadMessage(newChallenge(), bytes.Repeat([]byte{4}, 32), 1),
		"version":  syncprotocol.GetChainReadMessage(newChallenge(), key, 2),
	} {
		if bytes.Equal(message, base) {
			t.Errorf("a change of the %s leaves the message as it was", name)
		}
	}
}

func TestCheckRevision_AcceptsAWellFormedRevisionAndATombstone(t *testing.T) {
	t.Parallel()

	if err := syncprotocol.CheckRevision(newRevision()); err != nil {
		t.Errorf("a revision: %v", err)
	}
	if err := syncprotocol.CheckRevision(newTombstone()); err != nil {
		t.Errorf("a tombstone: %v", err)
	}
	limit := newRevision()
	limit.EncMeta = bytes.Repeat([]byte{1}, syncprotocol.MaxEncSize)
	limit.BlobSize = syncprotocol.MaxMultipartSize
	limit.ShareRootId = bytes.Repeat([]byte{1}, 16)
	if err := syncprotocol.CheckRevision(limit); err != nil {
		t.Errorf("a revision at the limits: %v", err)
	}
}

func TestCheckRevision_RefusesAShapeThatIsNotTheProtocol(t *testing.T) {
	t.Parallel()

	for name, change := range map[string]func(*syncv1.Revision){
		"vault id of 15 bytes":             func(r *syncv1.Revision) { r.VaultId = r.GetVaultId()[:15] },
		"file id of 17 bytes":              func(r *syncv1.Revision) { r.FileId = append(r.GetFileId(), 0) },
		"no file id":                       func(r *syncv1.Revision) { r.FileId = nil },
		"parent of 5 bytes":                func(r *syncv1.Revision) { r.ParentRev = r.GetParentRev()[:5] },
		"no blob":                          func(r *syncv1.Revision) { r.BlobId = nil },
		"blob id of 15 bytes":              func(r *syncv1.Revision) { r.BlobId = r.GetBlobId()[:15] },
		"blob hash of 31 bytes":            func(r *syncv1.Revision) { r.BlobHash = r.GetBlobHash()[:31] },
		"no blob hash":                     func(r *syncv1.Revision) { r.BlobHash = nil },
		"blob size zero":                   func(r *syncv1.Revision) { r.BlobSize = 0 },
		"blob size over the limit":         func(r *syncv1.Revision) { r.BlobSize = syncprotocol.MaxMultipartSize + 1 },
		"no meta":                          func(r *syncv1.Revision) { r.EncMeta = nil },
		"meta over the limit":              func(r *syncv1.Revision) { r.EncMeta = make([]byte, syncprotocol.MaxEncSize+1) },
		"path hash of 31 bytes":            func(r *syncv1.Revision) { r.PathHash = r.GetPathHash()[:31] },
		"no path hash":                     func(r *syncv1.Revision) { r.PathHash = nil },
		"share root of 5 bytes":            func(r *syncv1.Revision) { r.ShareRootId = []byte{1, 2, 3, 4, 5} },
		"author of 31 bytes":               func(r *syncv1.Revision) { r.AuthorDeviceSignKey = r.GetAuthorDeviceSignKey()[:31] },
		"signature of 63 bytes":            func(r *syncv1.Revision) { r.Signature = r.GetSignature()[:63] },
		"no signature":                     func(r *syncv1.Revision) { r.Signature = nil },
		"tombstone with a blob id":         func(r *syncv1.Revision) { *r = *newTombstone(); r.BlobId = bytes.Repeat([]byte{4}, 16) },
		"tombstone with a blob hash":       func(r *syncv1.Revision) { *r = *newTombstone(); r.BlobHash = bytes.Repeat([]byte{5}, 32) },
		"tombstone with a blob size":       func(r *syncv1.Revision) { *r = *newTombstone(); r.BlobSize = 4096 },
		"tombstone with a meta over limit": func(r *syncv1.Revision) { *r = *newTombstone(); r.EncMeta = make([]byte, syncprotocol.MaxEncSize+1) },
	} {
		revision := newRevision()
		change(revision)
		if err := syncprotocol.CheckRevision(revision); !errors.Is(err, syncprotocol.ErrInvalidRevision) {
			t.Errorf("%s: error = %v, want ErrInvalidRevision", name, err)
		}
	}
	if err := syncprotocol.CheckRevision(nil); !errors.Is(err, syncprotocol.ErrInvalidRevision) {
		t.Errorf("nil revision: error = %v, want ErrInvalidRevision", err)
	}
}

func TestCheckCatalogEntry_AcceptsTheFirstEntryAndALaterOne(t *testing.T) {
	t.Parallel()
	first := newCatalogEntry()
	first.Version, first.PrevHash = 1, nil
	limit := newCatalogEntry()
	limit.EncBody = bytes.Repeat([]byte{1}, syncprotocol.MaxEncSize)

	for name, entry := range map[string]*syncv1.CatalogEntry{"first": first, "later": newCatalogEntry(), "at the limit": limit} {
		if err := syncprotocol.CheckCatalogEntry(entry); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestCheckCatalogEntry_RefusesAShapeThatIsNotTheProtocol(t *testing.T) {
	t.Parallel()

	for name, change := range map[string]func(*syncv1.CatalogEntry){
		"version zero":                func(e *syncv1.CatalogEntry) { e.Version = 0 },
		"first entry with a previous": func(e *syncv1.CatalogEntry) { e.Version = 1 },
		"later entry with none":       func(e *syncv1.CatalogEntry) { e.PrevHash = nil },
		"previous of 31 bytes":        func(e *syncv1.CatalogEntry) { e.PrevHash = e.GetPrevHash()[:31] },
		"vault id of 15 bytes":        func(e *syncv1.CatalogEntry) { e.VaultId = e.GetVaultId()[:15] },
		"no vault id":                 func(e *syncv1.CatalogEntry) { e.VaultId = nil },
		"body over the limit":         func(e *syncv1.CatalogEntry) { e.EncBody = make([]byte, syncprotocol.MaxEncSize+1) },
		"author of 31 bytes":          func(e *syncv1.CatalogEntry) { e.AuthorDeviceSignKey = e.GetAuthorDeviceSignKey()[:31] },
		"signature of 65 bytes":       func(e *syncv1.CatalogEntry) { e.Signature = append(e.GetSignature(), 0) },
		"no signature":                func(e *syncv1.CatalogEntry) { e.Signature = nil },
	} {
		entry := newCatalogEntry()
		change(entry)
		if err := syncprotocol.CheckCatalogEntry(entry); !errors.Is(err, syncprotocol.ErrInvalidCatalogEntry) {
			t.Errorf("%s: error = %v, want ErrInvalidCatalogEntry", name, err)
		}
	}
	if err := syncprotocol.CheckCatalogEntry(nil); !errors.Is(err, syncprotocol.ErrInvalidCatalogEntry) {
		t.Errorf("nil entry: error = %v, want ErrInvalidCatalogEntry", err)
	}
}

func TestLimits_HaveTheValuesOfTheProtocol(t *testing.T) {
	t.Parallel()

	for name, got := range map[string][2]int{
		"MaxBlobSize":       {syncprotocol.MaxBlobSize, 64 << 20},
		"MaxBatchRevisions": {syncprotocol.MaxBatchRevisions, 5000},
		"MaxReserveBlobs":   {syncprotocol.MaxReserveBlobs, 100},
		"MaxBlobURLs":       {syncprotocol.MaxBlobURLs, 100},
		"MaxEncSize":        {syncprotocol.MaxEncSize, 16 << 10},
		"DefaultPageSize":   {syncprotocol.DefaultPageSize, 200},
		"MaxPageSize":       {syncprotocol.MaxPageSize, 1000},
	} {
		if got[0] != got[1] {
			t.Errorf("%s = %d, want %d", name, got[0], got[1])
		}
	}
}
