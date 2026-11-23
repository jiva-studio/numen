package fakeserver

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	blobLife      = 15 * time.Minute
	bearerPrefix  = "Bearer "
	blobURLFolder = "/blobs/"
)

var (
	errNotSignedIn     = errors.New("not signed in")
	errNoVault         = errors.New("no such vault")
	errVaultExists     = errors.New("the vault exists")
	errInvalidRevision = errors.New("invalid revision")
	errInvalidEntry    = errors.New("invalid catalog entry")
	errInvalidRequest  = errors.New("invalid request")
	errBlobNotReady    = errors.New("a blob is not ready")
	errQuotaExceeded   = errors.New("the quota is exceeded")
	errNoUpload        = errors.New("no such upload")
	errIncomplete      = errors.New("the upload is incomplete")
	errRevokedAuthor   = errors.New("the author of the entry was revoked")
	errChainStale      = errors.New("the chain does not extend the stored one")
)

type vaultState struct {
	account   string
	seq       uint64
	prunedSeq uint64
	files     map[string]*syncv1.StoredRevision
	paths     map[string]string
}

func newVaultState(account string) *vaultState {
	return &vaultState{account: account, files: map[string]*syncv1.StoredRevision{}, paths: map[string]string{}}
}

type blobState int

const (
	blobReserved blobState = iota
	blobCommitted
	blobReplaced
	blobAbandoned
)

type blobRow struct {
	account    string
	vault      string
	size       uint64
	expiresAt  time.Time
	state      blobState
	body       []byte
	isUploaded bool
	multipart  *multipartState
}

func (s *Server) authenticate(header http.Header) (string, error) {
	session, err := s.authenticateSession(header)
	return session.AccountFingerprint, err
}

func (s *Server) authenticateSession(header http.Header) (Session, error) {
	authorization := header.Get("Authorization")
	if !strings.HasPrefix(authorization, bearerPrefix) {
		return Session{}, connect.NewError(connect.CodeUnauthenticated, errNotSignedIn)
	}
	session, isLive := s.findLiveSession(strings.TrimPrefix(authorization, bearerPrefix))
	if !isLive {
		return Session{}, connect.NewError(connect.CodeUnauthenticated, errNotSignedIn)
	}
	return session, nil
}

func (s *Server) findLiveSession(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	at := slices.IndexFunc(s.sessions, func(one Session) bool { return one.Token == token && one.ExpiresAt.After(s.now()) })
	if at < 0 {
		return Session{}, false
	}
	return s.sessions[at], true
}

func (s *Server) getOwnedVault(account string, id []byte) (*vaultState, error) {
	found, exists := s.vaults[string(id)]
	if !exists || found.account != account {
		return nil, connect.NewError(connect.CodeNotFound, errNoVault)
	}
	return found, nil
}

func getPageLimit(limit uint32) int {
	switch {
	case limit == 0:
		return syncprotocol.DefaultPageSize
	case limit > syncprotocol.MaxPageSize:
		return syncprotocol.MaxPageSize
	}
	return int(limit)
}
