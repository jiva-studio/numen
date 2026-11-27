package syncprotocol_test

import (
	"bytes"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func newCodecEntry() *syncv1.CatalogEntry {
	return &syncv1.CatalogEntry{
		Version: 1, VaultId: bytes.Repeat([]byte{1}, 16), CreatesVault: true, EncBody: []byte("sealed"),
		AuthorDeviceSignKey: bytes.Repeat([]byte{2}, 32), Signature: bytes.Repeat([]byte{3}, 64),
	}
}

func TestDecodeCatalogEntry_ReturnsTheEntryEncodeCatalogEntryMade(t *testing.T) {
	t.Parallel()
	entry := newCodecEntry()

	raw, err := syncprotocol.EncodeCatalogEntry(entry)
	if err != nil {
		t.Fatal(err)
	}
	got, err := syncprotocol.DecodeCatalogEntry(raw)

	if err != nil || !proto.Equal(got, entry) {
		t.Errorf("DecodeCatalogEntry = %v, %v, want the entry", got, err)
	}
}

func TestCatalogCodec_KeepsTheBytesOfAStoredEntry(t *testing.T) {
	t.Parallel()
	entry := newCatalogEntry()
	want := readHex(t,
		"0803122009090909090909090909090909090909090909090909090909090909090909091a100101010101010101"+
			"010101010101010120012a04626f6479322007070707070707070707070707070707070707070707070707070707"+
			"070707073a4008080808080808080808080808080808080808080808080808080808080808080808080808080808"+
			"080808080808080808080808080808080808080808080808")

	got, err := syncprotocol.EncodeCatalogEntry(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("stored entry = %x\nwant           %x", got, want)
	}

	decoded, err := syncprotocol.DecodeCatalogEntry(want)
	if err != nil || !proto.Equal(decoded, entry) {
		t.Errorf("DecodeCatalogEntry = %v, %v, want the entry", decoded, err)
	}
}

func TestDecodeCatalogEntry_RefusesBytesThatAreNotAnEntryAndAnEntryOfAWrongSize(t *testing.T) {
	t.Parallel()
	short := newCodecEntry()
	short.VaultId = short.GetVaultId()[:15]
	raw, err := syncprotocol.EncodeCatalogEntry(short)
	if err != nil {
		t.Fatal(err)
	}

	for name, input := range map[string][]byte{"noise": {0xff, 0xff, 0xff}, "short vault id": raw} {
		if _, err := syncprotocol.DecodeCatalogEntry(input); !errors.Is(err, syncprotocol.ErrInvalidCatalogEntry) {
			t.Errorf("%s: DecodeCatalogEntry = %v, want ErrInvalidCatalogEntry", name, err)
		}
	}
}
