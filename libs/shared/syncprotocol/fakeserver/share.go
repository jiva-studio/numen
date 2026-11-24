package fakeserver

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"maps"
	"slices"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver/internal/sharestate"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (s *Server) AppendShareLogEntry(_ context.Context, req *connect.Request[syncv1.AppendShareLogEntryRequest]) (*connect.Response[syncv1.AppendShareLogEntryResponse], error) {
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
	if err := syncprotocol.CheckShareLogEntry(entry); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if entry.GetKind() == syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE {
		return s.createShare(account, entry)
	}
	return s.extendShare(account, entry)
}

func (s *Server) createShare(account string, entry *syncv1.ShareLogEntry) (*connect.Response[syncv1.AppendShareLogEntryResponse], error) {
	if _, err := s.getOwnedVault(account, entry.GetVaultId()); err != nil {
		return nil, err
	}
	if _, exists := s.shares[string(entry.GetShareRootId())]; exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, errShareExists)
	}
	if err := s.checkShareAuthor(account, entry); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	share := sharestate.New(account, string(entry.GetVaultId()))
	share.Log = []*syncv1.ShareLogEntry{entry}
	s.shares[string(entry.GetShareRootId())] = share
	return connect.NewResponse(&syncv1.AppendShareLogEntryResponse{Committed: true, Head: entry}), nil
}

func (s *Server) extendShare(account string, entry *syncv1.ShareLogEntry) (*connect.Response[syncv1.AppendShareLogEntryResponse], error) {
	share, found := s.shares[string(entry.GetShareRootId())]
	if !found || share.Account != account || share.IsClosed {
		return nil, newNoShare()
	}
	if err := s.checkShareAuthor(account, entry); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	head := syncprotocol.GetShareLogEntryID(share.GetHead())
	if entry.GetVersion() != share.GetHead().GetVersion()+1 || !bytes.Equal(entry.GetPrevHash(), head[:]) {
		return connect.NewResponse(&syncv1.AppendShareLogEntryResponse{Head: share.GetHead()}), nil
	}
	if err := s.checkShareKind(share, entry); err != nil {
		return nil, err
	}
	applyShareEntry(share, entry)
	return connect.NewResponse(&syncv1.AppendShareLogEntryResponse{Committed: true, Head: entry}), nil
}

func (s *Server) checkShareAuthor(account string, entry *syncv1.ShareLogEntry) error {
	author := entry.GetAuthorDeviceSignKey()
	if !isDeviceOf(s.chains[account], author) {
		return fmt.Errorf("%w: the author is not a device of the account", errInvalidEntry)
	}
	message := syncprotocol.GetShareLogEntryMessage(entry)
	if !identity.VerifyForPlugin(author, syncprotocol.SignInPluginID, message, [64]byte(entry.GetSignature())) {
		return fmt.Errorf("%w: the signature does not match", errInvalidEntry)
	}
	return nil
}

func (s *Server) checkShareKind(share *sharestate.State, entry *syncv1.ShareLogEntry) error {
	var problem string
	switch entry.GetKind() {
	case syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER:
		return s.checkAddReader(share, entry)
	case syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER:
		problem = getRemoveReaderProblem(share, entry)
	case syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE:
		if entry.GetEpoch() != share.Epoch+1 || !hasSealsFor(entry, share.ListActiveMembers(), nil) {
			problem = "a rotation is the next epoch with a seal for each reader"
		}
	case syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE,
		syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE, syncv1.ShareLogKind_SHARE_LOG_KIND_UNSPECIFIED:
		if entry.GetEpoch() != share.Epoch {
			problem = "the epoch does not change"
		}
	}
	if problem != "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %s", errInvalidEntry, problem))
	}
	return nil
}

func getRemoveReaderProblem(share *sharestate.State, entry *syncv1.ShareLogEntry) string {
	subject := share.FindMemberByRoot(entry.GetSubjectRootSignKey())
	switch {
	case entry.GetEpoch() != share.Epoch+1:
		return "a removal is the next epoch"
	case subject == nil || !subject.IsActive():
		return "the subject is not a reader"
	case !hasSealsFor(entry, share.ListActiveMembers(), subject):
		return "a removal has a seal for each remaining reader"
	}
	return ""
}

func (s *Server) checkAddReader(share *sharestate.State, entry *syncv1.ShareLogEntry) error {
	root := entry.GetSubjectRootSignKey()
	account := identity.RecoveryPublicKey{Sign: root}.Fingerprint().String()
	_, isKnown := s.chains[account]
	member := share.FindMemberByRoot(root)
	var problem string
	switch {
	case !isKnown:
		problem = "the reader is not valid"
	case entry.GetEpoch() != share.Epoch:
		problem = "an addition keeps the epoch"
	case account == share.Account:
		problem = "the owner is not a reader"
	case member != nil && member.IsActive():
		problem = "the reader is a member"
	case len(share.ListActiveMembers()) >= syncprotocol.MaxShareSeals:
		problem = "the share has the most readers"
	}
	if problem != "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %s", errInvalidEntry, problem))
	}
	return nil
}

func hasSealsFor(entry *syncv1.ShareLogEntry, readers []*sharestate.Member, skipped *sharestate.Member) bool {
	wanted := map[string]bool{}
	for _, reader := range readers {
		if reader != skipped {
			wanted[string(reader.Root)] = true
		}
	}
	if len(entry.GetSeals()) != len(wanted) {
		return false
	}
	for _, seal := range entry.GetSeals() {
		if !wanted[string(seal.GetReaderRootSignKey())] {
			return false
		}
		delete(wanted, string(seal.GetReaderRootSignKey()))
	}
	return true
}

func applyShareEntry(share *sharestate.State, entry *syncv1.ShareLogEntry) {
	version := entry.GetVersion()
	share.Log = append(share.Log, entry)
	share.Epoch = entry.GetEpoch()
	switch entry.GetKind() {
	case syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER:
		root := entry.GetSubjectRootSignKey()
		account := identity.RecoveryPublicKey{Sign: root}.Fingerprint().String()
		if member := share.FindMemberByRoot(root); member != nil {
			member.Added, member.Removed = version, 0
			return
		}
		share.Members = append(share.Members, &sharestate.Member{Account: account, Root: bytes.Clone(root), Added: version})
	case syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER:
		share.FindMemberByRoot(entry.GetSubjectRootSignKey()).Removed = version
	case syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE:
		for _, member := range share.ListActiveMembers() {
			member.Removed = version
		}
		share.IsClosed = true
	case syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE,
		syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, syncv1.ShareLogKind_SHARE_LOG_KIND_UNSPECIFIED:
	}
}

func (s *Server) ListShares(_ context.Context, req *connect.Request[syncv1.ListSharesRequest]) (*connect.Response[syncv1.ListSharesResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var listed []*syncv1.ShareSummary
	for _, id := range slices.Sorted(maps.Keys(s.shares)) {
		share := s.shares[id]
		if share.Account != account {
			continue
		}
		listed = append(listed, &syncv1.ShareSummary{
			ShareRootId: []byte(id), VaultId: []byte(share.Vault), HeadVersion: share.GetHead().GetVersion(), Epoch: share.Epoch, IsClosed: share.IsClosed,
		})
	}
	return connect.NewResponse(&syncv1.ListSharesResponse{Shares: listed}), nil
}

func (s *Server) GetShareLog(_ context.Context, req *connect.Request[syncv1.GetShareLogRequest]) (*connect.Response[syncv1.GetShareLogResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	share, found := s.shares[string(req.Msg.GetShareRootId())]
	var member *sharestate.Member
	if found {
		member = share.FindMember(account)
	}
	if !found || (share.Account != account && member == nil) {
		return nil, newNoShare()
	}
	upTo := uint64(len(share.Log))
	if share.Account != account && !member.IsActive() {
		upTo = member.Removed
	}
	since := min(req.Msg.GetSinceVersion(), upTo)
	rest := share.Log[since:upTo]
	limit := getPageLimit(req.Msg.GetLimit())
	response := &syncv1.GetShareLogResponse{Entries: rest, OwnerChain: s.chains[share.Account].Bytes()}
	if len(rest) > limit {
		response.Entries, response.HasMore = rest[:limit], true
	}
	return connect.NewResponse(response), nil
}

func (s *Server) getReaderShare(account string, id []byte) (*sharestate.State, error) {
	share, found := s.shares[string(id)]
	if !found || share.Account == account {
		return nil, newNoShare()
	}
	member := share.FindMember(account)
	if member == nil {
		return nil, newNoShare()
	}
	if !member.IsActive() {
		return nil, newShareEnded(member.Removed)
	}
	return share, nil
}

func (s *Server) GetShareChanges(_ context.Context, req *connect.Request[syncv1.GetShareChangesRequest]) (*connect.Response[syncv1.GetShareChangesResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	share, err := s.getReaderShare(account, req.Msg.GetShareRootId())
	if err != nil {
		return nil, err
	}
	since := req.Msg.GetSinceSeq()
	rows := s.listShareChanges(share, req.Msg.GetShareRootId(), since)
	page := cutShareChanges(rows, getPageLimit(req.Msg.GetLimit()))
	next := since
	if len(page) > 0 {
		next = page[len(page)-1].GetChange().GetSeq()
	}
	return connect.NewResponse(&syncv1.GetShareChangesResponse{Changes: page, NextSeq: next, HasMore: len(page) < len(rows)}), nil
}

func (s *Server) listShareChanges(share *sharestate.State, id []byte, since uint64) []*syncv1.ShareChange {
	vault := s.vaults[share.Vault]
	if vault == nil {
		return nil
	}
	var rows []*syncv1.ShareChange
	for file, head := range vault.files {
		revision := head.GetRevision()
		metaHash := sha256.Sum256(revision.GetEncMeta())
		switch exitSeq, exited := share.Exits[file]; {
		case bytes.Equal(revision.GetShareRootId(), id) && head.GetSeq() > since:
			rows = append(rows, &syncv1.ShareChange{Change: head, EncMetaHash: metaHash[:]})
		case exited && exitSeq > since && !bytes.Equal(revision.GetShareRootId(), id):
			rows = append(rows, &syncv1.ShareChange{Change: getStrippedHead(head, exitSeq), EncMetaHash: metaHash[:], Left: true})
		}
	}
	slices.SortFunc(rows, func(a, b *syncv1.ShareChange) int {
		return cmp.Or(
			cmp.Compare(a.GetChange().GetSeq(), b.GetChange().GetSeq()),
			bytes.Compare(a.GetChange().GetRevision().GetFileId(), b.GetChange().GetRevision().GetFileId()),
		)
	})
	return rows
}

func getStrippedHead(head *syncv1.StoredRevision, seq uint64) *syncv1.StoredRevision {
	stripped, isStored := proto.Clone(head).(*syncv1.StoredRevision)
	if !isStored {
		return head
	}
	stripped.Revision.EncMeta, stripped.Seq = nil, seq
	return stripped
}

func cutShareChanges(rows []*syncv1.ShareChange, limit int) []*syncv1.ShareChange {
	if len(rows) <= limit {
		return rows
	}
	edge := rows[limit-1].GetChange().GetSeq()
	end := slices.IndexFunc(rows, func(row *syncv1.ShareChange) bool { return row.GetChange().GetSeq() > edge })
	if end < 0 {
		return rows
	}
	return rows[:end]
}

func (s *Server) GetShareBlobUrls(_ context.Context, req *connect.Request[syncv1.GetShareBlobUrlsRequest]) (*connect.Response[syncv1.GetShareBlobUrlsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	share, err := s.getReaderShare(account, req.Msg.GetShareRootId())
	if err != nil {
		return nil, err
	}
	ids := req.Msg.GetBlobIds()
	if len(ids) == 0 || len(ids) > syncprotocol.MaxBlobURLs {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %d ids", errInvalidRequest, len(ids)))
	}
	if err := s.countShareURLs(share, len(ids)); err != nil {
		return nil, err
	}
	shared := s.listSharedBlobs(share, req.Msg.GetShareRootId())
	downloads := make([]*syncv1.BlobDownload, 0, len(ids))
	for _, id := range ids {
		download := &syncv1.BlobDownload{BlobId: id}
		if row, found := s.blobs[string(id)]; found && shared[string(id)] && row.state == blobCommitted {
			download.Url, download.ExpiresAt, download.Found = s.getBlobURL(id), s.now().Add(blobLife).Unix(), true
		}
		downloads = append(downloads, download)
	}
	return connect.NewResponse(&syncv1.GetShareBlobUrlsResponse{Downloads: downloads}), nil
}

func (s *Server) countShareURLs(share *sharestate.State, count int) error {
	now := s.now()
	window := now.Truncate(shareWindow)
	if share.URLs[window.Unix()]+count > shareURLsPerHour {
		return newRateLimited(window.Add(shareWindow).Sub(now))
	}
	share.URLs[window.Unix()] += count
	return nil
}

func (s *Server) listSharedBlobs(share *sharestate.State, id []byte) map[string]bool {
	shared := map[string]bool{}
	if vault := s.vaults[share.Vault]; vault != nil {
		for _, head := range vault.files {
			if bytes.Equal(head.GetRevision().GetShareRootId(), id) && !head.GetRevision().GetDeleted() {
				shared[string(head.GetRevision().GetBlobId())] = true
			}
		}
	}
	return shared
}
