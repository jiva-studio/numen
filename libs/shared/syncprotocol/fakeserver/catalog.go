package fakeserver

import (
	"bytes"
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (s *Server) AppendCatalogEntry(_ context.Context, req *connect.Request[syncv1.AppendCatalogEntryRequest]) (*connect.Response[syncv1.AppendCatalogEntryResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	entry := req.Msg.GetEntry()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkWriteAllowed(account); err != nil {
		return nil, err
	}
	if err := s.checkEntry(account, entry); err != nil {
		return nil, wrapInvalidArgument(err)
	}
	log := s.catalogs[account]
	if !isNextEntry(log, entry) {
		return connect.NewResponse(&syncv1.AppendCatalogEntryResponse{Head: getLastEntry(log)}), nil
	}
	if err := s.registerVault(account, entry); err != nil {
		return nil, err
	}
	s.catalogs[account] = append(log, entry)
	return connect.NewResponse(&syncv1.AppendCatalogEntryResponse{Committed: true, Head: entry}), nil
}

func (s *Server) checkEntry(account string, entry *syncv1.CatalogEntry) error {
	if err := syncprotocol.CheckCatalogEntry(entry); err != nil {
		return err
	}
	author := entry.GetAuthorDeviceSignKey()
	if isRevoked(s.chains[account], author) {
		return connect.NewError(connect.CodePermissionDenied, errRevokedAuthor)
	}
	if !isDeviceOf(s.chains[account], author) {
		return fmt.Errorf("%w: the author is not a device of the account", errInvalidEntry)
	}
	message := syncprotocol.GetCatalogEntryMessage(entry)
	if !identity.VerifyForPlugin(author, syncprotocol.SignInPluginID, message, [64]byte(entry.GetSignature())) {
		return fmt.Errorf("%w: the signature does not match", errInvalidEntry)
	}
	return nil
}

func getLastEntry(log []*syncv1.CatalogEntry) *syncv1.CatalogEntry {
	if len(log) == 0 {
		return nil
	}
	return log[len(log)-1]
}

func isNextEntry(log []*syncv1.CatalogEntry, entry *syncv1.CatalogEntry) bool {
	if entry.GetVersion() != uint64(len(log))+1 {
		return false
	}
	if len(log) == 0 {
		return len(entry.GetPrevHash()) == 0
	}
	head := syncprotocol.GetCatalogEntryID(log[len(log)-1])
	return bytes.Equal(entry.GetPrevHash(), head[:])
}

func (s *Server) registerVault(account string, entry *syncv1.CatalogEntry) error {
	id := string(entry.GetVaultId())
	_, exists := s.vaults[id]
	if !entry.GetCreatesVault() {
		_, err := s.getOwnedVault(account, entry.GetVaultId())
		return err
	}
	if exists {
		return connect.NewError(connect.CodeAlreadyExists, errVaultExists)
	}
	s.vaults[id] = newVaultState(account)
	return nil
}

func (s *Server) GetCatalog(_ context.Context, req *connect.Request[syncv1.GetCatalogRequest]) (*connect.Response[syncv1.GetCatalogResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	log := s.catalogs[account]
	since := min(req.Msg.GetSinceVersion(), uint64(len(log)))
	rest := log[since:]
	limit := getPageLimit(req.Msg.GetLimit())
	if len(rest) <= limit {
		return connect.NewResponse(&syncv1.GetCatalogResponse{Entries: rest}), nil
	}
	return connect.NewResponse(&syncv1.GetCatalogResponse{Entries: rest[:limit], HasMore: true}), nil
}
