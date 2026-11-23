package fakeserver

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

var errNotImplemented = errors.New("not implemented")

// IssueRestoreEpoch announces a restore: from now on every answer of GetChanges carries the id of the new
// epoch and GetRestoreEpoch returns it signed. It returns the id.
func (s *Server) IssueRestoreEpoch() ([]byte, error) {
	return nil, errNotImplemented
}

// DeleteAccountData removes the catalogs, vaults, files and blobs of every account, as the retention deletion
// of a real server does. The accounts, their device chains and their subscriptions stay.
func (s *Server) DeleteAccountData() error {
	return errNotImplemented
}

// DropBlob removes the body of a blob from the store and keeps the row, so a head that names the blob has no body.
func (s *Server) DropBlob(id []byte) error {
	return errNotImplemented
}

func (s *Server) GetRestoreEpoch(_ context.Context, _ *connect.Request[syncv1.GetRestoreEpochRequest]) (*connect.Response[syncv1.GetRestoreEpochResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotImplemented)
}
