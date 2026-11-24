package fakeserver

import (
	"bytes"
	"context"
	"slices"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (s *Server) GetVaultUsage(_ context.Context, req *connect.Request[syncv1.GetVaultUsageRequest]) (*connect.Response[syncv1.GetVaultUsageResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usage := []*syncv1.VaultUsage{}
	for id, vault := range s.vaults {
		if vault.account == account {
			usage = append(usage, &syncv1.VaultUsage{VaultId: []byte(id), UsedBytes: s.getVaultBytes(account, id)})
		}
	}
	slices.SortFunc(usage, func(a, b *syncv1.VaultUsage) int { return bytes.Compare(a.GetVaultId(), b.GetVaultId()) })
	return connect.NewResponse(&syncv1.GetVaultUsageResponse{Vaults: usage}), nil
}

// getVaultBytes counts the reserved and committed blobs, as used does.
func (s *Server) getVaultBytes(account, vault string) uint64 {
	var total uint64
	for _, row := range s.blobs {
		if row.account == account && row.vault == vault && isCounted(row) {
			total += row.size
		}
	}
	return total
}

func isCounted(row *blobRow) bool {
	return row.state == blobReserved || row.state == blobCommitted
}

func (s *Server) DeleteVault(_ context.Context, req *connect.Request[syncv1.DeleteVaultRequest]) (*connect.Response[syncv1.DeleteVaultResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if phase, _ := s.getPhase(account); phase == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED {
		return nil, connect.NewError(connect.CodePermissionDenied, errNeedsSubscription)
	}
	id := string(req.Msg.GetVaultId())
	if vault, found := s.vaults[id]; !found || vault.account != account {
		return connect.NewResponse(&syncv1.DeleteVaultResponse{}), nil
	}
	freed := s.removeVault(account, id)
	return connect.NewResponse(&syncv1.DeleteVaultResponse{Deleted: true, FreedBytes: freed}), nil
}

func (s *Server) removeVault(account, id string) uint64 {
	freed := s.getVaultBytes(account, id)
	for blob, row := range s.blobs {
		if row.account == account && row.vault == id {
			s.removeBlobObjects(blob, row)
			delete(s.blobs, blob)
		}
	}
	for share, state := range s.shares {
		if state.Vault == id {
			delete(s.shares, share)
		}
	}
	delete(s.vaults, id)
	s.used[account] -= min(freed, s.used[account])
	return freed
}

func (s *Server) removeBlobObjects(blob string, row *blobRow) {
	s.store.Remove(getObjectName(blob))
	if row.multipart == nil {
		return
	}
	for number := range row.multipart.parts {
		s.store.Remove(getPartName(blob, number))
	}
}
