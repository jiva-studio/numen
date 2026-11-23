package fakeserver

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"
	"slices"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

type batch struct {
	account string
	vaultID string
	vault   *vaultState
	revs    []*syncv1.Revision
	ids     [][32]byte
	heads   []*syncv1.StoredRevision
}

func (b *batch) isApplied(i int) bool {
	return b.heads[i] != nil && bytes.Equal(b.heads[i].GetRevId(), b.ids[i][:])
}

func (s *Server) CommitRevisions(_ context.Context, req *connect.Request[syncv1.CommitRevisionsRequest]) (*connect.Response[syncv1.CommitRevisionsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkWriteAllowed(account); err != nil {
		return nil, err
	}
	b, err := s.newBatch(account, req.Msg)
	if err != nil {
		return nil, err
	}
	if err := s.checkBlobs(b); err != nil {
		return nil, err
	}
	if conflicts := b.listConflicts(); len(conflicts) > 0 {
		return connect.NewResponse(&syncv1.CommitRevisionsResponse{Conflicts: conflicts}), nil
	}
	return connect.NewResponse(s.applyBatch(b)), nil
}

func (s *Server) newBatch(account string, req *syncv1.CommitRevisionsRequest) (*batch, error) {
	if err := checkBatchShape(req.GetVaultId(), req.GetRevisions()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	vault, err := s.getOwnedVault(account, req.GetVaultId())
	if err != nil {
		return nil, err
	}
	if err := s.checkAuthors(account, req.GetRevisions()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	b := &batch{account: account, vaultID: string(req.GetVaultId()), vault: vault, revs: req.GetRevisions()}
	for _, rev := range b.revs {
		b.ids = append(b.ids, syncprotocol.GetRevisionID(rev))
		b.heads = append(b.heads, vault.files[string(rev.GetFileId())])
	}
	return b, nil
}

func checkBatchShape(vaultID []byte, revs []*syncv1.Revision) error {
	if len(revs) == 0 || len(revs) > syncprotocol.MaxBatchRevisions {
		return fmt.Errorf("%w: a batch has %d revisions", errInvalidRevision, len(revs))
	}
	files, paths := map[string]bool{}, map[string]bool{}
	for _, rev := range revs {
		if err := checkRevisionOf(vaultID, rev); err != nil {
			return err
		}
		if !addOnce(files, rev.GetFileId()) || (!rev.GetDeleted() && !addOnce(paths, rev.GetPathHash())) {
			return fmt.Errorf("%w: two revisions share a file or a path", errInvalidRevision)
		}
	}
	return nil
}

func checkRevisionOf(vaultID []byte, rev *syncv1.Revision) error {
	if err := syncprotocol.CheckRevision(rev); err != nil {
		return err
	}
	if !bytes.Equal(rev.GetVaultId(), vaultID) {
		return fmt.Errorf("%w: a revision is of another vault", errInvalidRevision)
	}
	return nil
}

func addOnce(set map[string]bool, key []byte) bool {
	if set[string(key)] {
		return false
	}
	set[string(key)] = true
	return true
}

func (s *Server) checkAuthors(account string, revs []*syncv1.Revision) error {
	chain := s.chains[account]
	for _, rev := range revs {
		author := rev.GetAuthorDeviceSignKey()
		message := syncprotocol.GetRevisionMessage(rev)
		if !isDeviceOf(chain, author) || !identity.VerifyForPlugin(author, syncprotocol.SignInPluginID, message, [64]byte(rev.GetSignature())) {
			return fmt.Errorf("%w: the author or the signature is not valid", errInvalidRevision)
		}
	}
	return nil
}

func (s *Server) checkBlobs(b *batch) error {
	claimed := map[string]bool{}
	for i, rev := range b.revs {
		if rev.GetDeleted() || b.isApplied(i) || b.isMetadataOnly(i) || b.isStale(i) {
			continue
		}
		if !s.isBlobReady(b, rev) || !addOnce(claimed, rev.GetBlobId()) {
			return newBlobNotReady(uint32(i))
		}
	}
	return nil
}

// isStale says whether revision i names a parent that is not the head of its file: the batch is answered with that
// conflict, whatever the state of its blob.
func (b *batch) isStale(i int) bool {
	rev, head := b.revs[i], b.heads[i]
	return head != nil && !bytes.Equal(rev.GetParentRev(), head.GetRevId())
}

// isMetadataOnly says whether revision i names the blob of the head of its own file, with the same hash and size, and
// the head as its parent.
func (b *batch) isMetadataOnly(i int) bool {
	rev, head := b.revs[i], b.heads[i]
	if head == nil || head.GetRevision().GetDeleted() || rev.GetDeleted() {
		return false
	}
	old := head.GetRevision()
	return bytes.Equal(rev.GetParentRev(), head.GetRevId()) && bytes.Equal(rev.GetBlobId(), old.GetBlobId()) &&
		bytes.Equal(rev.GetBlobHash(), old.GetBlobHash()) && rev.GetBlobSize() == old.GetBlobSize()
}

func (s *Server) isBlobReady(b *batch, rev *syncv1.Revision) bool {
	row, found := s.blobs[string(rev.GetBlobId())]
	if !found || row.account != b.account || row.vault != b.vaultID || row.state != blobReserved {
		return false
	}
	if row.multipart != nil {
		return row.multipart.isCompleted && row.size == rev.GetBlobSize()
	}
	return row.expiresAt.After(s.now()) && row.isUploaded && row.size == rev.GetBlobSize() && uint64(len(row.body)) == row.size
}

func newBlobNotReady(index uint32) error {
	failure := connect.NewError(connect.CodeFailedPrecondition, errBlobNotReady)
	if detail, err := connect.NewErrorDetail(&syncv1.BlobNotReady{RevisionIndex: index}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}

func (b *batch) listConflicts() []*syncv1.Conflict {
	inBatch := map[string]bool{}
	for _, rev := range b.revs {
		inBatch[string(rev.GetFileId())] = true
	}
	var conflicts []*syncv1.Conflict
	for i := range b.revs {
		if conflict := b.getConflict(i, inBatch); conflict != nil && !b.isApplied(i) {
			conflicts = append(conflicts, conflict)
		}
	}
	return conflicts
}

func (b *batch) getConflict(i int, inBatch map[string]bool) *syncv1.Conflict {
	rev, head := b.revs[i], b.heads[i]
	if (head == nil && len(rev.GetParentRev()) != 0) || (head != nil && !bytes.Equal(rev.GetParentRev(), head.GetRevId())) {
		return &syncv1.Conflict{FileId: rev.GetFileId(), Kind: syncv1.ConflictKind_CONFLICT_KIND_PARENT_MISMATCH, Head: head}
	}
	if rev.GetDeleted() {
		return nil
	}
	holder, taken := b.vault.paths[string(rev.GetPathHash())]
	if !taken || holder == string(rev.GetFileId()) || inBatch[holder] {
		return nil
	}
	return &syncv1.Conflict{
		FileId: rev.GetFileId(), Kind: syncv1.ConflictKind_CONFLICT_KIND_PATH_TAKEN, Head: b.vault.files[holder], ExistingFileId: []byte(holder),
	}
}

func (s *Server) applyBatch(b *batch) *syncv1.CommitRevisionsResponse {
	stored := make([]*syncv1.StoredRevision, len(b.revs))
	var fresh []int
	for i := range b.revs {
		if b.isApplied(i) {
			stored[i] = b.heads[i]
			continue
		}
		fresh = append(fresh, i)
	}
	if len(fresh) == 0 {
		return &syncv1.CommitRevisionsResponse{Committed: true, Seq: getHighestSeq(stored), Revisions: stored}
	}
	b.vault.seq++
	for _, i := range fresh {
		s.releaseHead(b, i)
	}
	for _, i := range fresh {
		stored[i] = s.storeRevision(b, i)
	}
	return &syncv1.CommitRevisionsResponse{Committed: true, Seq: b.vault.seq, Revisions: stored}
}

func getHighestSeq(stored []*syncv1.StoredRevision) uint64 {
	var highest uint64
	for _, one := range stored {
		highest = max(highest, one.GetSeq())
	}
	return highest
}

func (s *Server) releaseHead(b *batch, i int) {
	old := b.heads[i]
	if old == nil {
		return
	}
	file := string(old.GetRevision().GetFileId())
	if b.vault.paths[string(old.GetRevision().GetPathHash())] == file {
		delete(b.vault.paths, string(old.GetRevision().GetPathHash()))
	}
	if b.isMetadataOnly(i) {
		return
	}
	if row, found := s.blobs[string(old.GetRevision().GetBlobId())]; found && row.state == blobCommitted {
		row.state = blobReplaced
		s.used[b.account] -= min(row.size, s.used[b.account])
	}
}

func (s *Server) storeRevision(b *batch, i int) *syncv1.StoredRevision {
	rev := b.revs[i]
	stored := &syncv1.StoredRevision{Revision: rev, RevId: b.ids[i][:], Seq: b.vault.seq}
	b.vault.files[string(rev.GetFileId())] = stored
	if !rev.GetDeleted() {
		b.vault.paths[string(rev.GetPathHash())] = string(rev.GetFileId())
		s.blobs[string(rev.GetBlobId())].state = blobCommitted
	}
	return stored
}

func (s *Server) GetChanges(_ context.Context, req *connect.Request[syncv1.GetChangesRequest]) (*connect.Response[syncv1.GetChangesResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, err := s.getOwnedVault(account, req.Msg.GetVaultId())
	if err != nil {
		return nil, err
	}
	since := req.Msg.GetSinceSeq()
	if since > 0 && since < vault.prunedSeq {
		return connect.NewResponse(&syncv1.GetChangesResponse{HeadSeq: vault.seq, Reset_: true}), nil
	}
	page := listChanges(vault, since, getPageLimit(req.Msg.GetLimit()))
	next := since
	if len(page) > 0 {
		next = page[len(page)-1].GetSeq()
	}
	return connect.NewResponse(&syncv1.GetChangesResponse{Changes: page, NextSeq: next, HasMore: next < vault.seq, HeadSeq: vault.seq}), nil
}

func listChanges(vault *vaultState, since uint64, limit int) []*syncv1.StoredRevision {
	var rows []*syncv1.StoredRevision
	for _, head := range vault.files {
		if head.GetSeq() > since {
			rows = append(rows, head)
		}
	}
	slices.SortFunc(rows, func(a, b *syncv1.StoredRevision) int {
		return cmp.Or(cmp.Compare(a.GetSeq(), b.GetSeq()), bytes.Compare(a.GetRevision().GetFileId(), b.GetRevision().GetFileId()))
	})
	if len(rows) <= limit {
		return rows
	}
	edge := rows[limit-1].GetSeq()
	end := slices.IndexFunc(rows, func(row *syncv1.StoredRevision) bool { return row.GetSeq() > edge })
	if end < 0 {
		return rows
	}
	return rows[:end]
}

// PruneTombstones drops the deletion records of every vault and keeps the highest seq dropped, so a cursor older
// than it gets reset. The real server does this for records older than 90 days.
func (s *Server) PruneTombstones() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	pruned := 0
	for _, vault := range s.vaults {
		for file, head := range vault.files {
			if head.GetRevision().GetDeleted() {
				vault.prunedSeq = max(vault.prunedSeq, head.GetSeq())
				delete(vault.files, file)
				pruned++
			}
		}
	}
	return pruned
}

func (s *Server) servePrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := fmt.Fprintln(w, s.PruneTombstones()); err != nil {
		return
	}
}
