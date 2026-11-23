package fakeserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	multipartLife   = 7 * 24 * time.Hour
	partURLLife     = time.Hour
	partFolder      = "/parts/"
	maxListedNumber = 100
)

type multipartState struct {
	partSize    uint64
	partCount   uint32
	parts       map[uint32]uint64
	isCompleted bool
	isAborted   bool
}

func (m *multipartState) getPartSize(number uint32, size uint64) uint64 {
	if number < m.partCount {
		return m.partSize
	}
	return size - uint64(m.partCount-1)*m.partSize
}

type partLogEntry struct {
	Blob      string `json:"blob"`
	Part      uint32 `json:"part"`
	Direction string `json:"direction"`
	From      int64  `json:"from"`
	Bytes     int64  `json:"bytes"`
}

func getPartName(id string, number uint32) string {
	return hex.EncodeToString([]byte(id)) + "/part-" + strconv.FormatUint(uint64(number), 10)
}

func getObjectName(id string) string {
	return hex.EncodeToString([]byte(id)) + "/object"
}

func (s *Server) ReserveMultipartBlob(_ context.Context, req *connect.Request[syncv1.ReserveMultipartBlobRequest]) (*connect.Response[syncv1.ReserveMultipartBlobResponse], error) {
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
	size, partSize := req.Msg.GetSize(), req.Msg.GetPartSize()
	if err := syncprotocol.CheckReserveMultipart(size, partSize); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	s.abandonExpired(account)
	if err := s.checkQuota(account, []uint64{size}); err != nil {
		return nil, err
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	expiresAt := s.now().Add(multipartLife)
	count := uint32((size + partSize - 1) / partSize)
	s.blobs[string(id)] = &blobRow{
		account: account, vault: string(req.Msg.GetVaultId()), size: size, expiresAt: expiresAt,
		multipart: &multipartState{partSize: partSize, partCount: count, parts: map[uint32]uint64{}},
	}
	s.used[account] += size
	return connect.NewResponse(&syncv1.ReserveMultipartBlobResponse{BlobId: id, PartSize: partSize, PartCount: count, ExpiresAt: expiresAt.Unix()}), nil
}

func (s *Server) getOpenUpload(account string, vaultID, blobID []byte) (*blobRow, error) {
	row, found := s.blobs[string(blobID)]
	if !found || row.multipart == nil || row.account != account || row.vault != string(vaultID) || row.state != blobReserved ||
		row.multipart.isCompleted || row.multipart.isAborted || !row.expiresAt.After(s.now()) {
		return nil, connect.NewError(connect.CodeNotFound, errNoUpload)
	}
	return row, nil
}

func (s *Server) GetPartUrls(_ context.Context, req *connect.Request[syncv1.GetPartUrlsRequest]) (*connect.Response[syncv1.GetPartUrlsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkWriteAllowed(account); err != nil {
		return nil, err
	}
	row, err := s.getOpenUpload(account, req.Msg.GetVaultId(), req.Msg.GetBlobId())
	if err != nil {
		return nil, err
	}
	numbers := req.Msg.GetPartNumbers()
	if len(numbers) == 0 || len(numbers) > syncprotocol.MaxPartURLs {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %d part numbers", errInvalidRequest, len(numbers)))
	}
	uploads := make([]*syncv1.PartUpload, 0, len(numbers))
	for _, number := range numbers {
		if number < 1 || number > row.multipart.partCount {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: part %d", errInvalidRequest, number))
		}
		uploads = append(uploads, &syncv1.PartUpload{
			PartNumber: number, Url: s.getBlobURL(req.Msg.GetBlobId()) + partFolder + strconv.FormatUint(uint64(number), 10),
			ExpiresAt: s.now().Add(partURLLife).Unix(), Size: row.multipart.getPartSize(number, row.size),
		})
	}
	return connect.NewResponse(&syncv1.GetPartUrlsResponse{Uploads: uploads}), nil
}

func (s *Server) ListParts(_ context.Context, req *connect.Request[syncv1.ListPartsRequest]) (*connect.Response[syncv1.ListPartsResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.getOpenUpload(account, req.Msg.GetVaultId(), req.Msg.GetBlobId())
	if err != nil {
		return nil, err
	}
	parts := make([]*syncv1.PartInfo, 0, len(row.multipart.parts))
	for number, size := range row.multipart.parts {
		parts = append(parts, &syncv1.PartInfo{PartNumber: number, Size: size})
	}
	slices.SortFunc(parts, func(a, b *syncv1.PartInfo) int { return int(a.GetPartNumber()) - int(b.GetPartNumber()) })
	return connect.NewResponse(&syncv1.ListPartsResponse{Parts: parts}), nil
}

func (s *Server) CompleteMultipartBlob(_ context.Context, req *connect.Request[syncv1.CompleteMultipartBlobRequest]) (*connect.Response[syncv1.CompleteMultipartBlobResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkWriteAllowed(account); err != nil {
		return nil, err
	}
	id := string(req.Msg.GetBlobId())
	if row, found := s.blobs[id]; found && row.multipart != nil && row.multipart.isCompleted && row.account == account && row.vault == string(req.Msg.GetVaultId()) {
		return connect.NewResponse(&syncv1.CompleteMultipartBlobResponse{}), nil
	}
	row, err := s.getOpenUpload(account, req.Msg.GetVaultId(), req.Msg.GetBlobId())
	if err != nil {
		return nil, err
	}
	if incomplete := listIncomplete(row); incomplete != nil {
		failure := connect.NewError(connect.CodeFailedPrecondition, errIncomplete)
		if detail, err := connect.NewErrorDetail(incomplete); err == nil {
			failure.AddDetail(detail)
		}
		return nil, failure
	}
	if err := s.joinParts(id, row); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&syncv1.CompleteMultipartBlobResponse{}), nil
}

func listIncomplete(row *blobRow) *syncv1.MultipartIncomplete {
	incomplete := &syncv1.MultipartIncomplete{}
	for number := uint32(1); number <= row.multipart.partCount; number++ {
		size, held := row.multipart.parts[number]
		switch {
		case !held && len(incomplete.Missing) < maxListedNumber:
			incomplete.Missing = append(incomplete.Missing, number)
		case !held:
		case size != row.multipart.getPartSize(number, row.size) && len(incomplete.WrongSize) < maxListedNumber:
			incomplete.WrongSize = append(incomplete.WrongSize, number)
		}
	}
	if len(incomplete.Missing) == 0 && len(incomplete.WrongSize) == 0 {
		return nil
	}
	return incomplete
}

func (s *Server) joinParts(id string, row *blobRow) error {
	object := make([]byte, 0, row.size)
	for number := uint32(1); number <= row.multipart.partCount; number++ {
		part, found := s.store.read(getPartName(id, number))
		if !found {
			return fmt.Errorf("%w: part %d is not stored", errIncomplete, number)
		}
		object = append(object, part...)
	}
	if err := s.store.write(getObjectName(id), object); err != nil {
		return err
	}
	for number := uint32(1); number <= row.multipart.partCount; number++ {
		s.store.remove(getPartName(id, number))
	}
	row.multipart.isCompleted, row.isUploaded = true, true
	return nil
}

func (s *Server) AbortMultipartBlob(_ context.Context, req *connect.Request[syncv1.AbortMultipartBlobRequest]) (*connect.Response[syncv1.AbortMultipartBlobResponse], error) {
	account, err := s.authenticate(req.Header())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkWriteAllowed(account); err != nil {
		return nil, err
	}
	id := string(req.Msg.GetBlobId())
	row, found := s.blobs[id]
	if !found || row.multipart == nil || row.account != account || row.vault != string(req.Msg.GetVaultId()) || row.multipart.isCompleted {
		return nil, connect.NewError(connect.CodeNotFound, errNoUpload)
	}
	if row.multipart.isAborted {
		return connect.NewResponse(&syncv1.AbortMultipartBlobResponse{}), nil
	}
	for number := range row.multipart.parts {
		s.store.remove(getPartName(id, number))
	}
	row.multipart.isAborted, row.multipart.parts, row.state = true, map[uint32]uint64{}, blobAbandoned
	s.used[account] -= min(row.size, s.used[account])
	return connect.NewResponse(&syncv1.AbortMultipartBlobResponse{}), nil
}

// storePart takes the PUT of one part to /blobs/<id>/parts/<number>.
func (s *Server) storePart(w http.ResponseWriter, r *http.Request, id string, number uint32) {
	s.mu.Lock()
	row, found := s.blobs[id]
	var expected uint64
	if found && row.multipart != nil && number >= 1 && number <= row.multipart.partCount {
		expected = row.multipart.getPartSize(number, row.size)
	}
	s.mu.Unlock()
	if expected == 0 {
		http.NotFound(w, r)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(expected)+1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case row.state != blobReserved || row.multipart.isCompleted || row.multipart.isAborted:
		http.NotFound(w, r)
	case !row.expiresAt.After(s.now()):
		http.Error(w, "the upload URL expired", http.StatusForbidden)
	case r.ContentLength != int64(len(body)) || uint64(len(body)) != expected:
		http.Error(w, "the body is not of the size of the part", http.StatusBadRequest)
	default:
		if err := s.store.write(getPartName(id, number), body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		row.multipart.parts[number] = expected
		s.partLog = append(s.partLog, partLogEntry{Blob: hex.EncodeToString([]byte(id)), Part: number, Direction: "put", Bytes: int64(len(body))})
		s.countReceivedBytes(int64(len(body)))
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) servePartLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	logged := slices.Clone(s.partLog)
	s.mu.Unlock()
	if logged == nil {
		logged = []partLogEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logged); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// getPartNumber reads the number of a part from the rest of a blob path, "<id hex>/parts/<n>".
func getPartNumber(rest string) (id string, number uint32, ok bool) {
	idText, tail, found := strings.Cut(rest, partFolder)
	if !found {
		return "", 0, false
	}
	raw, err := hex.DecodeString(idText)
	parsed, parseErr := strconv.ParseUint(tail, 10, 32)
	if err != nil || parseErr != nil {
		return "", 0, false
	}
	return string(raw), uint32(parsed), true
}
