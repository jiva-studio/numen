package syncprotocol

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// EncodeCatalogEntry returns the bytes of an entry as a device keeps a copy of a catalog: the same bytes for the
// same entry.
func EncodeCatalogEntry(entry *syncv1.CatalogEntry) ([]byte, error) {
	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("encode a catalog entry: %w", err)
	}
	return raw, nil
}

// DecodeCatalogEntry returns the entry EncodeCatalogEntry made the bytes of, or an error for bytes that are not one
// or that hold an entry the protocol does not allow.
func DecodeCatalogEntry(raw []byte) (*syncv1.CatalogEntry, error) {
	entry := &syncv1.CatalogEntry{}
	if err := proto.Unmarshal(raw, entry); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidCatalogEntry, err)
	}
	if err := CheckCatalogEntry(entry); err != nil {
		return nil, err
	}
	return entry, nil
}
