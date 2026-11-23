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

func TestEncodeCatalogEntry_GivesTheSameBytesForTheSameEntry(t *testing.T) {
	t.Parallel()

	one, errOne := syncprotocol.EncodeCatalogEntry(newCodecEntry())
	other, errOther := syncprotocol.EncodeCatalogEntry(newCodecEntry())

	if errOne != nil || errOther != nil || !bytes.Equal(one, other) {
		t.Errorf("encodings differ: %x, %x, %v, %v", one, other, errOne, errOther)
	}
}

func TestDecodeCatalogEntry_RefusesBytesThatAreNotAnEntryAndAnEntryOfAWrongSize(t *testing.T) {
	t.Parallel()
	short := newCodecEntry()
	short.VaultId = short.VaultId[:15]
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
