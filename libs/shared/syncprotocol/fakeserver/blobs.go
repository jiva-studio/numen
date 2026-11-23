package fakeserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (s *Server) ReserveBlobs(_ context.Context, req *connect.Request[syncv1.ReserveBlobsRequest]) (*connect.Response[syncv1.ReserveBlobsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkWriteAllowed(account); err != nil {
		return nil, err
	}
	if _, err := s.getOwnedVault(account, req.Msg.GetVaultId()); err != nil {
		return nil, err
	}
	sizes := req.Msg.GetSizes()
	if err := checkReserveSizes(sizes); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	s.abandonExpired(account)
	if err := s.checkQuota(account, sizes); err != nil {
		return nil, err
	}
	uploads, err := s.addReservations(account, string(req.Msg.GetVaultId()), sizes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&syncv1.ReserveBlobsResponse{Uploads: uploads}), nil
}

func checkReserveSizes(sizes []uint64) error {
	if len(sizes) == 0 || len(sizes) > syncprotocol.MaxReserveBlobs {
		return fmt.Errorf("%w: %d sizes", errInvalidRequest, len(sizes))
	}
	for _, size := range sizes {
		if size == 0 || size > syncprotocol.MaxBlobSize {
			return fmt.Errorf("%w: a size of %d bytes", errInvalidRequest, size)
		}
	}
	return nil
}

func (s *Server) abandonExpired(account string) {
	for _, row := range s.blobs {
		if row.account == account && row.state == blobReserved && !row.expiresAt.After(s.now()) {
			row.state = blobAbandoned
			row.body = nil
			s.used[account] -= min(row.size, s.used[account])
		}
	}
}

func (s *Server) checkQuota(account string, sizes []uint64) error {
	var total uint64
	for _, size := range sizes {
		total += size
	}
	limit := s.getQuota(account)
	if s.used[account]+total <= limit {
		return nil
	}
	failure := connect.NewError(connect.CodeResourceExhausted, errQuotaExceeded)
	if detail, err := connect.NewErrorDetail(&syncv1.QuotaExceeded{UsedBytes: s.used[account], LimitBytes: limit}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}

func (s *Server) addReservations(account, vault string, sizes []uint64) ([]*syncv1.BlobUpload, error) {
	expiresAt := s.now().Add(blobLife)
	uploads := make([]*syncv1.BlobUpload, 0, len(sizes))
	for _, size := range sizes {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			return nil, err
		}
		s.blobs[string(id)] = &blobRow{account: account, vault: vault, size: size, expiresAt: expiresAt}
		s.used[account] += size
		uploads = append(uploads, &syncv1.BlobUpload{BlobId: id, Url: s.getBlobURL(id), ExpiresAt: expiresAt.Unix(), Size: size})
	}
	return uploads, nil
}

func (s *Server) getBlobURL(id []byte) string {
	return strings.TrimSuffix(s.audience, "/") + blobURLFolder + hex.EncodeToString(id)
}

func (s *Server) GetBlobUrls(_ context.Context, req *connect.Request[syncv1.GetBlobUrlsRequest]) (*connect.Response[syncv1.GetBlobUrlsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.getOwnedVault(account, req.Msg.GetVaultId()); err != nil {
		return nil, err
	}
	ids := req.Msg.GetBlobIds()
	if len(ids) == 0 || len(ids) > syncprotocol.MaxBlobURLs {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %d ids", errInvalidRequest, len(ids)))
	}
	downloads := make([]*syncv1.BlobDownload, 0, len(ids))
	var missing [][]byte
	for _, id := range ids {
		downloads = append(downloads, s.getDownload(account, string(req.Msg.GetVaultId()), id))
		if row, found := s.blobs[string(id)]; found && row.account == account && row.state == blobCommitted && !row.isUploaded {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return nil, newMissingBlobFailure(missing)
	}
	return connect.NewResponse(&syncv1.GetBlobUrlsResponse{Downloads: downloads}), nil
}

func (s *Server) getDownload(account, vault string, id []byte) *syncv1.BlobDownload {
	row, found := s.blobs[string(id)]
	if !found || row.account != account || row.vault != vault || row.state != blobCommitted {
		return &syncv1.BlobDownload{BlobId: id}
	}
	return &syncv1.BlobDownload{BlobId: id, Url: s.getBlobURL(id), ExpiresAt: s.now().Add(blobLife).Unix(), Found: true}
}

func (s *Server) serveBlob(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, blobURLFolder)
	if partID, number, isPart := getPartNumber(rest); isPart {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.storePart(w, r, partID, number)
		return
	}
	id, err := hex.DecodeString(rest)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		s.storeBlobBody(w, r, string(id))
	case http.MethodGet:
		s.sendBlobBody(w, r, string(id))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) storeBlobBody(w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, syncprotocol.MaxBlobSize+1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, found := s.blobs[id]
	switch {
	case !found || row.state != blobReserved:
		http.NotFound(w, r)
	case !row.expiresAt.After(s.now()):
		http.Error(w, "the upload URL expired", http.StatusForbidden)
	case r.ContentLength != int64(len(body)) || uint64(len(body)) != row.size:
		http.Error(w, "the body is not of the reserved size", http.StatusBadRequest)
	default:
		row.body, row.isUploaded = body, true
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) sendBlobBody(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	row, found := s.blobs[id]
	var body []byte
	if found && row.state == blobCommitted {
		body = s.getBody(id, row)
	}
	s.mu.Unlock()
	if body == nil {
		http.NotFound(w, r)
		return
	}
	s.sendBody(w, r, id, body)
}

func (s *Server) getBody(id string, row *blobRow) []byte {
	if row.multipart == nil {
		return slices.Clone(row.body)
	}
	body, _ := s.store.read(getObjectName(id))
	return body
}

func (s *Server) logDownload(id string, from, sent int64) {
	if row, found := s.blobs[id]; found && row.multipart != nil {
		s.partLog = append(s.partLog, partLogEntry{Blob: hex.EncodeToString([]byte(id)), Direction: "get-range", From: from, Bytes: sent})
	}
}

func (s *Server) serveStoredBlobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := w.Write(s.listStoredBlobs()); err != nil {
		return
	}
}

func (s *Server) listStoredBlobs() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.blobs))
	for id, row := range s.blobs {
		if row.isUploaded && (row.body != nil || row.multipart != nil) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	var all bytes.Buffer
	for _, id := range ids {
		all.WriteString(id)
		all.Write(s.getBody(id, s.blobs[id]))
	}
	return all.Bytes()
}
