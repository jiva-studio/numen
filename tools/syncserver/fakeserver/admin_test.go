package fakeserver_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

type partLogRow struct {
	Blob      string `json:"blob"`
	Part      uint32 `json:"part"`
	Direction string `json:"direction"`
	From      int64  `json:"from"`
	Bytes     int64  `json:"bytes"`
}

func newStoppingEnv(tb testing.TB) (*env, *atomic.Int32) {
	tb.Helper()
	var stops atomic.Int32
	server := fakeserver.New(fakeserver.Config{Stop: func() { stops.Add(1) }})
	served := httptest.NewServer(server)
	tb.Cleanup(served.Close)
	server.SetAudience(served.URL)
	account := newOwnedAccount(tb)
	device := account.addDevice(tb, "First")
	return newEnvOn(tb, served.URL, &account, device), &stops
}

func (e *env) post(path string) int {
	e.tb.Helper()
	request, err := http.NewRequestWithContext(e.tb.Context(), http.MethodPost, e.address+path, http.NoBody)
	if err != nil {
		e.tb.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		e.tb.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		e.tb.Fatal(err)
	}
	return response.StatusCode
}

func (e *env) getAdmin(path string) []byte {
	e.tb.Helper()
	request, err := http.NewRequestWithContext(e.tb.Context(), http.MethodGet, e.address+path, http.NoBody)
	if err != nil {
		e.tb.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		e.tb.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			e.tb.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		e.tb.Fatal(err)
	}
	return body
}

func (e *env) readPartLog() []partLogRow {
	e.tb.Helper()
	var rows []partLogRow
	if err := json.Unmarshal(e.getAdmin("/admin/part-log"), &rows); err != nil {
		e.tb.Fatal(err)
	}
	return rows
}

func (e *env) commitBigFile(vault []byte, name string) (blob, body []byte) {
	e.tb.Helper()
	body = newBigBody()
	reserved, err := e.blob.ReserveMultipartBlob(e.tb.Context(), vault, uint64(len(body)), partSize)
	if err != nil {
		e.tb.Fatal(err)
	}
	for number := uint32(1); number <= reserved.GetPartCount(); number++ {
		e.putParts(vault, reserved.GetBlobId(), body, number)
	}
	if err := e.blob.CompleteMultipartBlob(e.tb.Context(), vault, reserved.GetBlobId()); err != nil {
		e.tb.Fatal(err)
	}
	e.commit(vault, e.newRevisionOf(vault, newID(16), nil, name, &syncv1.BlobUpload{BlobId: reserved.GetBlobId()}, body))
	return reserved.GetBlobId(), body
}

func TestAdmin_PartLogListsEachPartOnceAndEmptyIsAList(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	if got := strings.TrimSpace(string(e.getAdmin("/admin/part-log"))); got != "[]" {
		t.Fatalf("part log = %s, want an empty list", got)
	}
	reserved, err := e.blob.ReserveMultipartBlob(t.Context(), vault, bigSize, partSize)
	if err != nil {
		t.Fatal(err)
	}

	e.putParts(vault, reserved.GetBlobId(), newBigBody(), 2, 1)

	rows := e.readPartLog()
	if len(rows) != 2 || rows[0].Direction != "put" || rows[0].Part != 2 || rows[1].Part != 1 || rows[0].Bytes != partSize {
		t.Errorf("part log = %+v, want a put of part 2 then of part 1, each of %d bytes", rows, partSize)
	}
}

func TestAdmin_StopAfterBytesAsksTheHostToStopOnceTheBudgetIsReceived(t *testing.T) {
	t.Parallel()
	e, stops := newStoppingEnv(t)
	vault := e.newVault()
	reserved, err := e.blob.ReserveMultipartBlob(t.Context(), vault, bigSize, partSize)
	if err != nil {
		t.Fatal(err)
	}
	if status := e.post("/admin/stop-after-bytes?n=7000000"); status != http.StatusNoContent {
		t.Fatalf("arm status = %d, want 204", status)
	}

	e.putParts(vault, reserved.GetBlobId(), newBigBody(), 1)
	if stops.Load() != 0 {
		t.Fatalf("stopped after %d bytes, want it to wait for 7000000", partSize)
	}
	e.putParts(vault, reserved.GetBlobId(), newBigBody(), 2)
	e.putParts(vault, reserved.GetBlobId(), newBigBody(), 3)

	if stops.Load() != 1 {
		t.Errorf("stops = %d, want one request once the budget ended", stops.Load())
	}
}

func TestAdmin_StopAfterBytesRefusesWhatIsNotACount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	if status := e.post("/admin/stop-after-bytes?n=many"); status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
	if status := e.post("/admin/stop-after-download-bytes?n=-1"); status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestAdmin_StopAfterDownloadBytesEndsTheBodyAndLogsWhatWasSent(t *testing.T) {
	t.Parallel()
	e, stops := newStoppingEnv(t)
	vault := e.newVault()
	blob, body := e.commitBigFile(vault, "big.bin")
	downloads, err := e.blob.GetBlobUrls(t.Context(), vault, [][]byte{blob})
	if err != nil || !downloads[0].GetFound() {
		t.Fatalf("downloads = %v, error = %v", downloads, err)
	}
	const budget = 1000
	e.post("/admin/stop-after-download-bytes?n=1000")

	stream, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, downloads[0].GetUrl(), 10)
	if err == nil {
		got, readErr := io.ReadAll(stream)
		t.Logf("the stream ended as planned: %v", readErr)
		if err := stream.Close(); err != nil {
			t.Logf("closing the cut stream: %v", err)
		}
		if len(got) > budget || !bytes.Equal(got, body[10:10+len(got)]) {
			t.Errorf("read %d bytes, want at most %d of the right bytes", len(got), budget)
		}
	}

	if stops.Load() != 1 {
		t.Errorf("stops = %d, want 1", stops.Load())
	}
	var seen bool
	for _, row := range e.readPartLog() {
		if row.Direction == "get-range" && row.From == 10 && row.Bytes == budget {
			seen = true
		}
	}
	if !seen {
		t.Errorf("part log = %+v, want a get-range from 10 of %d bytes", e.readPartLog(), budget)
	}
}

func TestAdmin_BlobsHoldTheBodiesOfCompletedMultipartObjects(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	_, body := e.commitBigFile(vault, "big.bin")

	if all := e.getAdmin("/admin/blobs"); !bytes.Contains(all, body[:4096]) {
		t.Errorf("/admin/blobs of %d bytes does not hold the object", len(all))
	}
}

func TestGetChanges_ResetsACursorOlderThanTheFloorOfPrunedDeletions(t *testing.T) {
	t.Parallel()
	server, address := newServer(t)
	account := newOwnedAccount(t)
	e := newEnvOn(t, address, &account, account.addDevice(t, "First"))
	vault := e.newVault()
	file := newID(16)
	first := e.commit(vault, e.newFile(vault, file, nil, "gone.md", []byte("gone")))
	e.commit(vault, e.signRevision(&syncv1.Revision{
		VaultId: vault, FileId: file, ParentRev: first.GetRevisions()[0].GetRevId(), Deleted: true,
		PathHash: hashOf([]byte("gone.md")), AuthorDeviceSignKey: e.device.Public().Sign,
	}))
	e.commit(vault, e.newFile(vault, newID(16), nil, "kept.md", []byte("kept")))
	if pruned := server.PruneTombstones(); pruned != 1 {
		t.Fatalf("pruned = %d, want 1", pruned)
	}

	old, err := e.revision.GetChanges(t.Context(), vault, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := e.revision.GetChanges(t.Context(), vault, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	current, err := e.revision.GetChanges(t.Context(), vault, 3, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !old.GetReset_() || len(old.GetChanges()) != 0 || old.GetHeadSeq() != 3 {
		t.Errorf("old cursor answer = %v, want reset with no changes and head 3", old)
	}
	if fresh.GetReset_() || len(fresh.GetChanges()) != 1 {
		t.Errorf("answer from the start = %v, want no reset and the one live file", fresh)
	}
	if current.GetReset_() {
		t.Errorf("answer at the floor = %v, want no reset", current)
	}
}

func TestAdminPulls_CountsTheReadsOfChangesOfEachDevice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	for range 3 {
		if _, err := e.revision.GetChanges(t.Context(), vault, 0, 0); err != nil {
			t.Fatal(err)
		}
	}

	var counts map[string]int
	if err := json.Unmarshal(e.getAdmin("/admin/pulls"), &counts); err != nil {
		t.Fatal(err)
	}

	if want := map[string]int{hex.EncodeToString(e.device.Public().Sign): 3}; !maps.Equal(counts, want) {
		t.Errorf("pulls = %v, want %v", counts, want)
	}
}
