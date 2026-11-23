package fakeserver

import (
	"google.golang.org/protobuf/proto"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// EncodeMessage returns the bytes a message takes on the wire, for a test that looks for plain text in what the
// server would hold.
func EncodeMessage(message proto.Message) ([]byte, error) { return proto.Marshal(message) }

// CloneCatalogEntry returns a copy of the entry that a test may change without changing the server's.
func CloneCatalogEntry(entry *syncv1.CatalogEntry) *syncv1.CatalogEntry {
	cloned, isEntry := proto.Clone(entry).(*syncv1.CatalogEntry)
	if !isEntry {
		return nil
	}
	return cloned
}
