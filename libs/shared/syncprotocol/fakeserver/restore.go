package fakeserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

var errNoBlob = errors.New("no such blob")

// restoreState is what the server keeps about restores: the key that signs epochs, the current epoch and the
// backup the admin paths put back.
type restoreState struct {
	key    ed25519.PrivateKey
	epoch  *syncv1.RestoreEpoch
	backup []byte
}

// getKey returns the key of the epochs, made on first use. The caller holds the lock.
func (r *restoreState) getKey() (ed25519.PrivateKey, error) {
	if r.key == nil {
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("make the key of the epochs: %w", err)
		}
		r.key = key
	}
	return r.key, nil
}

// getEpochID returns the id of the current epoch, nil before the first one. The caller holds the lock.
func (s *Server) getEpochID() []byte {
	if s.restore.epoch == nil {
		return nil
	}
	return slices.Clone(s.restore.epoch.GetId())
}

// IssueRestoreEpoch announces a restore: from now on every answer of GetChanges carries the id of the new
// epoch and GetRestoreEpoch returns it signed. It returns the id.
func (s *Server) IssueRestoreEpoch() ([]byte, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("make an epoch id: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key, err := s.restore.getKey()
	if err != nil {
		return nil, err
	}
	issuedAt := s.now().Unix()
	s.restore.epoch = &syncv1.RestoreEpoch{
		Id: id, IssuedAt: issuedAt, Signature: ed25519.Sign(key, syncprotocol.GetRestoreEpochMessage(id, issuedAt)),
	}
	return slices.Clone(id), nil
}

// DeleteAccountData removes the catalogs, vaults, files and blobs of every account, as the retention deletion
// of a real server does. The accounts, their device chains and their subscriptions stay.
func (s *Server) DeleteAccountData() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.catalogs = map[string][]*syncv1.CatalogEntry{}
	s.vaults = map[string]*vaultState{}
	s.blobs = map[string]*blobRow{}
	s.used = map[string]uint64{}
	return nil
}

// DropBlob removes the body of a blob from the store and keeps the row, so a head that names the blob has no body.
func (s *Server) DropBlob(id []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, found := s.blobs[string(id)]
	if !found {
		return errNoBlob
	}
	row.body, row.isUploaded = nil, false
	return nil
}

func (s *Server) GetRestoreEpoch(_ context.Context, req *connect.Request[syncv1.GetRestoreEpochRequest]) (*connect.Response[syncv1.GetRestoreEpochResponse], error) {
	if _, err := s.authenticate(req.Header()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key, err := s.restore.getKey()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	public, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("the key of the epochs is not an Ed25519 key"))
	}
	return connect.NewResponse(&syncv1.GetRestoreEpochResponse{EpochSignKey: public, Epoch: s.restore.epoch}), nil
}

func newMissingBlobFailure(ids [][]byte) error {
	failure := connect.NewError(connect.CodeNotFound, errors.New("the body of a blob is gone"))
	if detail, err := connect.NewErrorDetail(&syncv1.BlobMissing{BlobIds: ids}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}

func (s *Server) serveBackup(w http.ResponseWriter, r *http.Request) {
	s.serveAdminPost(w, r, func() error {
		kept, err := s.Snapshot()
		if err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.restore.backup = kept
		return nil
	})
}

func (s *Server) serveRestore(w http.ResponseWriter, r *http.Request) {
	s.serveAdminPost(w, r, func() error {
		s.mu.Lock()
		kept := s.restore.backup
		s.mu.Unlock()
		if kept == nil {
			return errors.New("there is no backup")
		}
		return s.Restore(kept)
	})
}

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	s.serveAdminPost(w, r, func() error {
		_, err := s.IssueRestoreEpoch()
		return err
	})
}

func (s *Server) serveRetention(w http.ResponseWriter, r *http.Request) {
	s.serveAdminPost(w, r, s.DeleteAccountData)
}

func (s *Server) serveAdminPost(w http.ResponseWriter, r *http.Request, run func() error) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
