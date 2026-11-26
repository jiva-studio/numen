// Package wiretest is test support: what a test needs to look at the protocol messages as the server holds them.
package wiretest

import (
	"connectrpc.com/connect"
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

// DecodeMessage fills the message from the bytes it takes on the wire.
func DecodeMessage(data []byte, message proto.Message) error { return proto.Unmarshal(data, message) }

// IsResourceExhausted reports whether the server refused the call for a limit it keeps.
func IsResourceExhausted(err error) bool { return connect.CodeOf(err) == connect.CodeResourceExhausted }
