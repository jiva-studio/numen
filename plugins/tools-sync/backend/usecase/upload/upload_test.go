package upload_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/upload"
)

const (
	vaultID    domain.VaultID = "Test"
	bigSize                   = 65 << 20
	partSize                  = 16 << 20
	uploadsKey                = "uploads.json"
)

type world struct {
	c      *synctest.Computer
	server *synctest.Server
	spy    *synctest.SpyHost
	remote []byte
	use    *upload.UploadFile
}

func newWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	c := synctest.NewFirstComputer(t, server, clock, string(vaultID), synctest.NewWords(t))
	c.Connect(t, vaultID)
	spy := c.Spy()
	use := upload.NewUploadFile(upload.Deps{Uploads: c.WireDeps().Uploads, Entries: c.State, Crypto: c.Account, Host: c.Host, Random: rand.Reader, Digests: c.WireDeps().Digests})
	return &world{c: c, server: server, spy: spy, remote: c.GetRemoteID(t, vaultID), use: use}
}

func newFileID(t *testing.T) []byte {
	t.Helper()
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (w *world) input(path string, stat domain.FileStat, fileID []byte) transfer.UploadInput {
	return transfer.UploadInput{Vault: vaultID, RemoteID: w.remote, Path: path, Stat: stat, FileID: fileID, ParentRev: nil}
}

func (w *world) writeFile(t *testing.T, path string, content []byte) domain.FileStat {
	t.Helper()
	w.c.WriteNote(vaultID, path, string(content))
	stat, err := w.c.Vaults.With(vaultID).StatFile(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	return stat
}

func (w *world) readEntries(t *testing.T) []transfer.UploadEntry {
	t.Helper()
	var kept struct {
		Uploads []transfer.UploadEntry `json:"uploads"`
	}
	raw, err := w.c.Host.Store(vaultID).Read(t.Context(), uploadsKey)
	if errors.Is(err, host.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &kept); err != nil {
		t.Fatal(err)
	}
	return kept.Uploads
}

func (w *world) putParts(t *testing.T) []synctest.PartLogEntry {
	t.Helper()
	var parts []synctest.PartLogEntry
	for _, one := range w.server.GetPartLog(t) {
		if one.Direction == "put" {
			parts = append(parts, one)
		}
	}
	return parts
}

// sourceAt reads a source as the sealer reads a plain file.
type sourceAt struct{ source *synctest.Source }

func (s sourceAt) ReadAt(p []byte, off int64) (int, error) {
	got := s.source.GetBytes(off, int64(len(p)))
	copy(p, got)
	if len(got) < len(p) {
		return len(got), io.EOF
	}
	return len(got), nil
}

func getStoredHash(t *testing.T, source *synctest.Source, key [32]byte, fileID, salt []byte) [32]byte {
	t.Helper()
	sealer, err := blobcipher.NewSealer(key, [16]byte(fileID), [16]byte(salt), source.GetStat().Size)
	if err != nil {
		t.Fatal(err)
	}
	hasher := sha256.New()
	buffer := make([]byte, 1<<20)
	for off := int64(0); off < sealer.StoredSize(); off += int64(len(buffer)) {
		n, err := sealer.ReadStoredAt(buffer, off, sourceAt{source: source})
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
		hasher.Write(buffer[:n])
	}
	return [32]byte(hasher.Sum(nil))
}

func (w *world) newSource(path string, seed uint64) *synctest.Source {
	source := synctest.NewSource(path, bigSize, seed, synctest.GetStart().Add(-time.Hour))
	w.spy.Serve(source)
	return source
}

func (w *world) countCalls(procedure string) int { return w.server.Fake.CountCalls(procedure) }

func TestUploadFile_SendsASmallFileInOneBodyThatOpensToTheFile(t *testing.T) {
	w := newWorld(t)
	content := make([]byte, 100_000)
	if _, err := rand.Read(content); err != nil {
		t.Fatal(err)
	}
	stat := w.writeFile(t, "Papers/Report.pdf", content)
	fileID := newFileID(t)

	result, err := w.use.Execute(t.Context(), w.input("Papers/Report.pdf", stat, fileID))

	if err != nil {
		t.Fatal(err)
	}
	if got := w.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != 1 {
		t.Errorf("reservations = %d, want 1", got)
	}
	if got := w.countCalls(syncv1connect.BlobServiceReserveMultipartBlobProcedure); got != 0 {
		t.Errorf("multipart reservations = %d, want none for a small file", got)
	}
	if kept := w.readEntries(t); len(kept) != 0 {
		t.Errorf("uploads.json holds %d entries, want none for a body sent in one request", len(kept))
	}
	if result.BlobSize == 0 || len(result.BlobID) == 0 {
		t.Errorf("result = %+v, want the blob and its size", result)
	}
	if blob := w.readUploadedBody(t, result.BlobID); sha256.Sum256(blob) != result.BlobHash || uint64(len(blob)) != result.BlobSize {
		t.Error("the blob on the server is not the one the result names")
	} else if plain, err := blobcipher.Open(result.Key, [16]byte(fileID), blob); err != nil || !bytes.Equal(plain, content) {
		t.Errorf("the blob opens to %d bytes (%v), want the file", len(plain), err)
	}
}

func (w *world) readUploadedBody(t *testing.T, blobID []byte) []byte {
	t.Helper()
	dump := w.server.GetStoredBlobs(t)
	if !bytes.HasPrefix(dump, blobID) {
		t.Fatal("the server holds something other than the blob")
	}
	return dump[len(blobID):]
}

func TestUploadFile_StoresATextFileSmallerThanABinaryFileOfTheSameBytes(t *testing.T) {
	w := newWorld(t)
	content := bytes.Repeat([]byte("the same line of text\n"), 20_000)
	text := w.writeFile(t, "Notes/log.txt", content)
	binary := w.writeFile(t, "Notes/log.bin", content)

	asText, err := w.use.Execute(t.Context(), w.input("Notes/log.txt", text, newFileID(t)))
	if err != nil {
		t.Fatal(err)
	}
	asBinary, err := w.use.Execute(t.Context(), w.input("Notes/log.bin", binary, newFileID(t)))
	if err != nil {
		t.Fatal(err)
	}

	if asText.BlobSize*4 > asBinary.BlobSize {
		t.Errorf("stored sizes = %d for text and %d for binary, want text compressed and binary not", asText.BlobSize, asBinary.BlobSize)
	}
	if asBinary.BlobSize < uint64(len(content)) {
		t.Errorf("binary stored size = %d, want at least the %d bytes of the file", asBinary.BlobSize, len(content))
	}
}

func TestUploadFile_ReadsAFileOfSeveralMiBInRangesOfAtMostFourMiB(t *testing.T) {
	w := newWorld(t)
	content := make([]byte, 9<<20)
	if _, err := rand.Read(content); err != nil {
		t.Fatal(err)
	}
	stat := w.writeFile(t, "Videos/Clip.bin", content)
	fileID := newFileID(t)

	result, err := w.use.Execute(t.Context(), w.input("Videos/Clip.bin", stat, fileID))

	if err != nil {
		t.Fatal(err)
	}
	if got := w.spy.GetMaxRead(); got > host.MaxReadBytes {
		t.Errorf("the largest read asked for %d bytes, want at most %d", got, host.MaxReadBytes)
	}
	blob := w.readUploadedBody(t, result.BlobID)
	if plain, err := blobcipher.Open(result.Key, [16]byte(fileID), blob); err != nil || !bytes.Equal(plain, content) {
		t.Errorf("the blob opens to %d bytes (%v), want the file of %d bytes", len(plain), err, len(content))
	}
}

func TestUploadFile_FailsWithErrFileChangedWhenTheFileChangesWhileItIsRead(t *testing.T) {
	w := newWorld(t)
	content := make([]byte, 9<<20)
	stat := w.writeFile(t, "Videos/Clip.bin", content)
	w.c.WriteNote(vaultID, "Videos/Clip.bin", string(content)+"more")

	_, err := w.use.Execute(t.Context(), w.input("Videos/Clip.bin", stat, newFileID(t)))

	if !errors.Is(err, upload.ErrFileChanged) {
		t.Errorf("error = %v, want ErrFileChanged for a file that is not the one the stat names", err)
	}
}

func TestUploadFile_SendsABigFileInPartsOfSixteenMiBAndKeepsItsKeySealed(t *testing.T) {
	w := newWorld(t)
	source := w.newSource("Videos/Trip.bin", 1)
	fileID := newFileID(t)

	result, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID))

	if err != nil {
		t.Fatal(err)
	}
	if got := w.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != 0 {
		t.Errorf("single reservations = %d, want none for a big file", got)
	}
	if got := w.countCalls(syncv1connect.BlobServiceReserveMultipartBlobProcedure); got != 1 {
		t.Errorf("multipart reservations = %d, want 1", got)
	}
	if got := w.countCalls(syncv1connect.BlobServiceCompleteMultipartBlobProcedure); got != 1 {
		t.Errorf("completions = %d, want 1", got)
	}
	parts := w.putParts(t)
	var sent int64
	for i, part := range parts {
		sent += part.Bytes
		if part.Part != uint32(i+1) {
			t.Errorf("part %d of the log is part number %d, want them in order", i+1, part.Part)
		}
	}
	if len(parts) != 5 || sent != int64(result.BlobSize) || result.BlobSize <= partSize*4 {
		t.Errorf("%d parts of %d bytes, stored size %d, want 5 parts that add up to it", len(parts), sent, result.BlobSize)
	}
	if got := w.spy.GetMaxRead(); got > host.MaxReadBytes {
		t.Errorf("the largest read asked for %d bytes, want at most %d", got, host.MaxReadBytes)
	}
	entries := w.readEntries(t)
	if len(entries) != 1 {
		t.Fatalf("uploads.json holds %d entries, want the one of the upload until its revision is committed", len(entries))
	}
	if want := getStoredHash(t, source, result.Key, fileID, entries[0].Salt); want != result.BlobHash {
		t.Error("the blob hash is not the hash of the stored bytes of the file under its key and salt")
	}
	if entries[0].PartsDone != 5 || entries[0].PartCount != 5 {
		t.Errorf("entry = %d of %d parts done, want 5 of 5", entries[0].PartsDone, entries[0].PartCount)
	}
}

func TestUploadFile_KeepsNoPlainKeyInTheStateOfAnUpload(t *testing.T) {
	w := newWorld(t)
	source := w.newSource("Videos/Trip.bin", 2)
	fileID := newFileID(t)

	result, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID))

	if err != nil {
		t.Fatal(err)
	}
	raw, err := w.c.Host.Store(vaultID).Read(t.Context(), uploadsKey)
	if err != nil {
		t.Fatal(err)
	}
	for name, plain := range map[string][]byte{"raw": result.Key[:], "base64": []byte(base64.StdEncoding.EncodeToString(result.Key[:]))} {
		if bytes.Contains(raw, plain) {
			t.Errorf("uploads.json holds the key of the file as %s", name)
		}
	}
	entry := w.readEntries(t)[0]
	keyContext := append([]byte("numen-sync/upload-key/v1"), w.remote...)
	opened, err := w.c.Client.OpenData(t.Context(), append(keyContext, fileID...), entry.SealedKey)
	if err != nil || !bytes.Equal(opened, result.Key[:]) {
		t.Errorf("the sealed key opens to %x (%v), want the key of the file", opened, err)
	}
}

func TestUploadFile_ForgetDropsTheStateOfACommittedUpload(t *testing.T) {
	w := newWorld(t)
	source := w.newSource("Videos/Trip.bin", 3)
	fileID := newFileID(t)
	if _, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID)); err != nil {
		t.Fatal(err)
	}

	if err := w.use.Forget(t.Context(), vaultID, fileID); err != nil {
		t.Fatal(err)
	}

	if kept := w.readEntries(t); len(kept) != 0 {
		t.Errorf("uploads.json holds %d entries after Forget, want none", len(kept))
	}
}

func TestUploadFile_ContinuesAfterABreakAndSendsNoPartTwice(t *testing.T) {
	w := newWorld(t)
	source := w.newSource("Videos/Trip.bin", 4)
	fileID := newFileID(t)
	w.server.StopAfterUploadBytes(t, 20<<20)
	if _, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID)); err == nil {
		t.Fatal("the upload went through a server that stopped")
	}
	kept := w.readEntries(t)
	if len(kept) != 1 || kept[0].PartsDone != 2 {
		t.Fatalf("entries after the break = %+v, want the upload with 2 parts done", kept)
	}
	w.server.SetDown(false)

	result, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID))

	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint32]int{}
	for _, part := range w.putParts(t) {
		seen[part.Part]++
	}
	if len(seen) != 5 {
		t.Errorf("parts received = %v, want parts 1 to 5", seen)
	}
	for number, times := range seen {
		if times != 1 {
			t.Errorf("part %d was received %d times, want once", number, times)
		}
	}
	entry := w.readEntries(t)[0]
	if want := getStoredHash(t, source, result.Key, fileID, entry.Salt); want != result.BlobHash {
		t.Error("the hash after the break is not the hash of a file sent whole")
	}
	if got := w.countCalls(syncv1connect.BlobServiceReserveMultipartBlobProcedure); got != 1 {
		t.Errorf("multipart reservations = %d, want the one reservation kept over the break", got)
	}
}

func TestUploadFile_BuildsTheHashAgainWhenTheServerHasFewerPartsThanTheState(t *testing.T) {
	w := newWorld(t)
	source := w.newSource("Videos/Trip.bin", 5)
	fileID := newFileID(t)
	w.server.StopAfterUploadBytes(t, 20<<20)
	if _, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID)); err == nil {
		t.Fatal("the upload went through a server that stopped")
	}
	kept := w.readEntries(t)
	kept[0].PartsDone, kept[0].StoredHashState = 3, []byte("a state of parts that the server does not hold")
	w.writeEntries(t, kept)
	w.server.SetDown(false)

	result, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID))

	if err != nil {
		t.Fatal(err)
	}
	if want := getStoredHash(t, source, result.Key, fileID, kept[0].Salt); want != result.BlobHash {
		t.Error("the hash is not the hash of a file sent whole after the state ran ahead of the server")
	}
	if parts := w.putParts(t); len(parts) != 5 {
		t.Errorf("%d parts received, want 5: the upload goes on from the parts the server holds", len(parts))
	}
}

func (w *world) writeEntries(t *testing.T, entries []transfer.UploadEntry) {
	t.Helper()
	raw, err := json.Marshal(struct {
		Uploads []transfer.UploadEntry `json:"uploads"`
	}{Uploads: entries})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.c.Host.Store(vaultID).Write(t.Context(), uploadsKey, raw); err != nil {
		t.Fatal(err)
	}
}

func TestUploadFile_SendsAgainThePartsTheServerHasBeyondTheState(t *testing.T) {
	w := newWorld(t)
	source := w.newSource("Videos/Trip.bin", 6)
	fileID := newFileID(t)
	w.server.StopAfterUploadBytes(t, 1)
	if _, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID)); err == nil {
		t.Fatal("the upload went through a server that stopped")
	}
	afterOne := w.readEntries(t)
	if len(afterOne) != 1 || afterOne[0].PartsDone != 1 {
		t.Fatalf("entries after the first break = %+v, want 1 part done", afterOne)
	}
	w.server.SetDown(false)
	w.server.StopAfterUploadBytes(t, 1)
	if _, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID)); err == nil {
		t.Fatal("the upload went through a server that stopped")
	}
	w.writeEntries(t, afterOne)
	w.server.SetDown(false)

	result, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID))

	if err != nil {
		t.Fatal(err)
	}
	if want := getStoredHash(t, source, result.Key, fileID, afterOne[0].Salt); want != result.BlobHash {
		t.Error("the hash is not the hash of a file sent whole after a part was sent again")
	}
	times := 0
	for _, part := range w.putParts(t) {
		if part.Part == 2 {
			times++
		}
	}
	if times != 2 {
		t.Errorf("part 2 was received %d times, want twice: once before the state knew it and once again", times)
	}
}

func TestUploadFile_StartsAgainWithANewKeyWhenTheFileChangedAfterTheBreak(t *testing.T) {
	w := newWorld(t)
	first := w.newSource("Videos/Trip.bin", 7)
	fileID := newFileID(t)
	w.server.StopAfterUploadBytes(t, 1)
	if _, err := w.use.Execute(t.Context(), w.input(first.GetPath(), first.GetStat(), fileID)); err == nil {
		t.Fatal("the upload went through a server that stopped")
	}
	before := w.readEntries(t)[0]
	w.server.SetDown(false)
	changed := synctest.NewSource(first.GetPath(), bigSize, 8, synctest.GetStart())
	w.spy.Serve(changed)

	result, err := w.use.Execute(t.Context(), w.input(changed.GetPath(), changed.GetStat(), fileID))

	if err != nil {
		t.Fatal(err)
	}
	after := w.readEntries(t)
	if len(after) != 1 || bytes.Equal(after[0].BlobID, before.BlobID) || bytes.Equal(after[0].SealedKey, before.SealedKey) {
		t.Fatalf("entries after the change = %+v, want one entry with a new reservation and a new key", after)
	}
	if got := w.countCalls(syncv1connect.BlobServiceAbortMultipartBlobProcedure); got != 1 {
		t.Errorf("aborted reservations = %d, want the old one", got)
	}
	if want := getStoredHash(t, changed, result.Key, fileID, after[0].Salt); want != result.BlobHash {
		t.Error("the hash is not the hash of the file as it is now")
	}
}

func TestUploadFile_StartsAgainWhenTheServerForgotTheReservation(t *testing.T) {
	w := newWorld(t)
	source := w.newSource("Videos/Trip.bin", 9)
	fileID := newFileID(t)
	w.server.StopAfterUploadBytes(t, 1)
	if _, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID)); err == nil {
		t.Fatal("the upload went through a server that stopped")
	}
	before := w.readEntries(t)[0]
	w.server.SetDown(false)
	w.c.Clock.Advance(8 * 24 * time.Hour)

	result, err := w.use.Execute(t.Context(), w.input(source.GetPath(), source.GetStat(), fileID))

	if err != nil {
		t.Fatal(err)
	}
	after := w.readEntries(t)
	if len(after) != 1 || bytes.Equal(after[0].BlobID, before.BlobID) {
		t.Fatalf("entries = %+v, want the upload started again under a new reservation", after)
	}
	if got := w.countCalls(syncv1connect.BlobServiceReserveMultipartBlobProcedure); got != 2 {
		t.Errorf("multipart reservations = %d, want 2", got)
	}
	if want := getStoredHash(t, source, result.Key, fileID, after[0].Salt); want != result.BlobHash {
		t.Error("the hash is not the hash of the file sent whole under the new reservation")
	}
}
