package download_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

const (
	bigSize       = 65 << 20
	downloadsFile = "downloads.json"
	chunkLog2     = 20
)

type world struct {
	p     *synctest.Pair
	vault transfer.Vault
	use   *download.DownloadFile
}

func newWorld(t *testing.T) *world {
	t.Helper()
	p := synctest.NewPair(t, fakeserver.Config{})
	return &world{p: p, vault: p.GetVault(t), use: syncwire.NewDownloads(p.B.WireDeps())}
}

func (w *world) publish(t *testing.T, file transfer.File) (*syncv1.StoredRevision, synctest.Meta) {
	t.Helper()
	outcomes, err := commit.NewCommitRevisions(w.p.A.CommitDeps()).Execute(t.Context(), transfer.CommitInput{VaultID: synctest.FirstVault, Vault: w.vault, Files: []transfer.File{file}})
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[0].Status != transfer.Committed {
		t.Fatalf("the commit of %s = %+v, want committed", file.Path, outcomes[0])
	}
	stored := w.p.A.ReadChanges(t, w.vault.ID)
	if len(stored) != 1 {
		t.Fatalf("the server holds %d heads, want 1", len(stored))
	}
	return stored[0], synctest.OpenMeta(t, w.vault.Key, stored[0].GetRevision())
}

func (w *world) publishBig(t *testing.T, path string, seed uint64) (*syncv1.StoredRevision, synctest.Meta, *synctest.Source) {
	t.Helper()
	source := synctest.NewSource(path, bigSize, seed, synctest.GetStart().Add(-time.Hour))
	w.p.A.Spy().Serve(source)
	stat := source.GetStat()
	stored, meta := w.publish(t, transfer.File{Path: path, ModifiedAt: stat.ModTime.UnixNano(), Stat: &stat})
	return stored, meta, source
}

func (w *world) input(stored *syncv1.StoredRevision, meta synctest.Meta) transfer.DownloadInput {
	rev := stored.GetRevision()
	return transfer.DownloadInput{
		Vault: synctest.SecondVault, RemoteID: rev.GetVaultId(), FileID: rev.GetFileId(), Path: meta.Path, BlobID: rev.GetBlobId(),
		BlobHash: rev.GetBlobHash(), BlobSize: rev.GetBlobSize(), Key: [32]byte(meta.Key), Hash: meta.Hash, Size: meta.Size,
		HeadRev: stored.GetRevId(), Baseline: nil,
	}
}

func (w *world) readEntries(t *testing.T) []transfer.DownloadEntry {
	t.Helper()
	raw, err := w.p.B.Host.Store(synctest.SecondVault).Read(t.Context(), downloadsFile)
	if errors.Is(err, host.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var kept struct {
		Downloads []transfer.DownloadEntry `json:"downloads"`
	}
	if err := json.Unmarshal(raw, &kept); err != nil {
		t.Fatal(err)
	}
	return kept.Downloads
}

func (w *world) getStagedSize(t *testing.T, name string) (int64, bool) {
	t.Helper()
	files, err := w.p.B.Host.Store(synctest.SecondVault).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Name == name {
			return file.Size, true
		}
	}
	return 0, false
}

func (w *world) getRanges(t *testing.T) []synctest.PartLogEntry {
	t.Helper()
	var ranges []synctest.PartLogEntry
	for _, one := range w.p.Server.GetPartLog(t) {
		if one.Direction == "get-range" {
			ranges = append(ranges, one)
		}
	}
	return ranges
}

func TestDownloadFile_StagesASmallFileOfAnyKindByItsNameAndDoesNotPlaceIt(t *testing.T) {
	w := newWorld(t)
	content := make([]byte, 300_000)
	if _, err := rand.Read(content); err != nil {
		t.Fatal(err)
	}
	stored, meta := w.publish(t, transfer.File{Path: "Papers/Report.pdf", Content: content})

	name, err := w.use.Execute(t.Context(), w.input(stored, meta))

	if err != nil {
		t.Fatal(err)
	}
	if want := note.StoredName(stored.GetRevision().GetFileId()); name != want {
		t.Errorf("staged name = %q, want %q", name, want)
	}
	if got := w.p.B.ReadStoreFile(t, synctest.SecondVault, name); !bytes.Equal(got, content) {
		t.Errorf("the staged file holds %d bytes, want the %d bytes of the file", len(got), len(content))
	}
	if _, isThere := w.p.B.ReadNote(synctest.SecondVault, "Papers/Report.pdf"); isThere {
		t.Error("the file is in the vault: staging does not place it")
	}
}

func TestDownloadFile_StagesAnEmptyFile(t *testing.T) {
	w := newWorld(t)
	stored, meta := w.publish(t, transfer.File{Path: "Papers/Empty.pdf", Content: []byte{}})

	name, err := w.use.Execute(t.Context(), w.input(stored, meta))

	if err != nil {
		t.Fatal(err)
	}
	if size, isThere := w.getStagedSize(t, name); !isThere || size != 0 {
		t.Errorf("staged size = %d (there: %v), want an empty staged file", size, isThere)
	}
}

func TestDownloadFile_RefusesASmallBlobThatIsNotTheOneItsRevisionNamesAndStagesNothing(t *testing.T) {
	w := newWorld(t)
	stored, meta := w.publish(t, transfer.File{Path: "Papers/Report.pdf", Content: []byte("pdf bytes")})
	input := w.input(stored, meta)
	input.BlobHash = bytes.Repeat([]byte{1}, 32)

	_, err := w.use.Execute(t.Context(), input)

	if !errors.Is(err, download.ErrIntegrity) {
		t.Errorf("error = %v, want ErrIntegrity", err)
	}
	if _, isThere := w.getStagedSize(t, note.StoredName(stored.GetRevision().GetFileId())); isThere {
		t.Error("a staged file is left after a blob that did not match its hash")
	}
}

func TestDownloadFile_RefusesASmallBlobWhoseFileIsNotTheOneItsMetadataNames(t *testing.T) {
	w := newWorld(t)
	stored, meta := w.publish(t, transfer.File{Path: "Papers/Report.pdf", Content: []byte("pdf bytes")})
	input := w.input(stored, meta)
	input.Hash = note.Hash([]byte("other bytes"))

	_, err := w.use.Execute(t.Context(), input)

	if !errors.Is(err, download.ErrIntegrity) {
		t.Errorf("error = %v, want ErrIntegrity for a file whose hash is not the one of its metadata", err)
	}
}

func TestDownloadFile_StagesABigFileInChunksAndItOpensToTheFile(t *testing.T) {
	w := newWorld(t)
	stored, meta, source := w.publishBig(t, "Videos/Trip.bin", 1)

	name, err := w.use.Execute(t.Context(), w.input(stored, meta))

	if err != nil {
		t.Fatal(err)
	}
	staged := w.p.B.ReadStoreFile(t, synctest.SecondVault, name)
	if int64(len(staged)) != bigSize || note.Hash(staged) != source.GetStat().Hash {
		t.Errorf("the staged file has %d bytes and hash %s, want the file of %d bytes and hash %s", len(staged), note.Hash(staged), bigSize, source.GetStat().Hash)
	}
	if ranges := w.getRanges(t); len(ranges) != 1 || ranges[0].From != 0 {
		t.Errorf("ranges read = %+v, want one read of the blob from its start", ranges)
	}
	if _, isThere := w.p.B.ReadNote(synctest.SecondVault, "Videos/Trip.bin"); isThere {
		t.Error("the file is in the vault: staging does not place it")
	}
}

func TestDownloadFile_ForgetDropsTheStateOfAPlacedFile(t *testing.T) {
	w := newWorld(t)
	stored, meta, _ := w.publishBig(t, "Videos/Trip.bin", 2)
	if _, err := w.use.Execute(t.Context(), w.input(stored, meta)); err != nil {
		t.Fatal(err)
	}
	if kept := w.readEntries(t); len(kept) != 1 || kept[0].NextChunk == 0 {
		t.Fatalf("entries after the download = %+v, want the position it reached", kept)
	}

	if err := w.use.Forget(t.Context(), synctest.SecondVault, stored.GetRevision().GetFileId()); err != nil {
		t.Fatal(err)
	}

	if kept := w.readEntries(t); len(kept) != 0 {
		t.Errorf("entries after Forget = %+v, want none", kept)
	}
}

func TestDownloadFile_ContinuesFromTheChunkItReachedAfterABreak(t *testing.T) {
	w := newWorld(t)
	stored, meta, source := w.publishBig(t, "Videos/Trip.bin", 3)
	input := w.input(stored, meta)
	w.p.Server.StopAfterDownloadBytes(t, 20<<20)
	name, err := w.use.Execute(t.Context(), input)
	if err == nil || errors.Is(err, download.ErrIntegrity) {
		t.Fatalf("error = %v, want a break that is not damage", err)
	}
	kept := w.readEntries(t)
	if len(kept) != 1 || kept[0].NextChunk == 0 {
		t.Fatalf("entries after the break = %+v, want the position of the last whole chunk", kept)
	}
	if size, _ := w.getStagedSize(t, name); size != int64(kept[0].NextChunk)<<chunkLog2-8 {
		t.Fatalf("staged size = %d, want the bytes of the %d chunks kept", size, kept[0].NextChunk)
	}
	w.p.Server.SetDown(false)

	if _, err := w.use.Execute(t.Context(), input); err != nil {
		t.Fatal(err)
	}

	staged := w.p.B.ReadStoreFile(t, synctest.SecondVault, name)
	if note.Hash(staged) != source.GetStat().Hash {
		t.Error("the file after the break is not the file")
	}
	ranges := w.getRanges(t)
	if len(ranges) != 2 || ranges[0].From != 0 || ranges[1].From != blobcipher.GetChunkOffset(chunkLog2, kept[0].NextChunk) {
		t.Errorf("ranges read = %+v, want the second from the offset of chunk %d and no byte of the chunks kept again", ranges, kept[0].NextChunk)
	}
}

func TestDownloadFile_StartsAgainWhenTheStagedFileIsNotTheSizeTheStateRecords(t *testing.T) {
	w := newWorld(t)
	stored, meta, source := w.publishBig(t, "Videos/Trip.bin", 4)
	input := w.input(stored, meta)
	w.p.Server.StopAfterDownloadBytes(t, 20<<20)
	name, _ := w.use.Execute(t.Context(), input)
	if err := w.p.B.Host.Store(synctest.SecondVault).Append(t.Context(), name, []byte("x")); err != nil {
		t.Fatal(err)
	}
	w.p.Server.SetDown(false)

	if _, err := w.use.Execute(t.Context(), input); err != nil {
		t.Fatal(err)
	}

	if note.Hash(w.p.B.ReadStoreFile(t, synctest.SecondVault, name)) != source.GetStat().Hash {
		t.Error("the file after a staged file of another size is not the file")
	}
	if ranges := w.getRanges(t); len(ranges) != 2 || ranges[1].From != 0 {
		t.Errorf("ranges read = %+v, want the second read from the start", ranges)
	}
}

func TestDownloadFile_RefusesABigBlobThatDoesNotOpenAndDropsWhatItStaged(t *testing.T) {
	w := newWorld(t)
	stored, meta, _ := w.publishBig(t, "Videos/Trip.bin", 5)
	w.p.Server.CorruptBlobs(true)

	_, err := w.use.Execute(t.Context(), w.input(stored, meta))

	if !errors.Is(err, download.ErrIntegrity) {
		t.Errorf("error = %v, want ErrIntegrity for a blob that is damaged", err)
	}
	if _, isThere := w.getStagedSize(t, note.StoredName(stored.GetRevision().GetFileId())); isThere {
		t.Error("a staged file is left after a damaged blob")
	}
	if kept := w.readEntries(t); len(kept) != 0 {
		t.Errorf("entries = %+v, want none after a damaged blob", kept)
	}
}

func TestDownloadFile_RefusesABigBlobWhoseHashIsNotTheOneOfItsRevision(t *testing.T) {
	w := newWorld(t)
	stored, meta, _ := w.publishBig(t, "Videos/Trip.bin", 6)
	input := w.input(stored, meta)
	input.BlobHash = bytes.Repeat([]byte{2}, 32)

	_, err := w.use.Execute(t.Context(), input)

	if !errors.Is(err, download.ErrIntegrity) {
		t.Errorf("error = %v, want ErrIntegrity", err)
	}
	if _, isThere := w.getStagedSize(t, note.StoredName(stored.GetRevision().GetFileId())); isThere {
		t.Error("a staged file is left after a blob whose hash differs from its revision")
	}
}

func TestDownloadFile_SweepDeletesTheStagedFilesNoDownloadNamesAndNothingElse(t *testing.T) {
	w := newWorld(t)
	stored, meta, _ := w.publishBig(t, "Videos/Trip.bin", 7)
	w.p.Server.StopAfterDownloadBytes(t, 20<<20)
	named, _ := w.use.Execute(t.Context(), w.input(stored, meta))
	store := w.p.B.Host.Store(synctest.SecondVault)
	if err := store.Write(t.Context(), "0badc0de.part", []byte("left behind")); err != nil {
		t.Fatal(err)
	}
	w.p.B.Vaults.Store().With(synctest.SecondVault).File("notes.json", []byte("{}"))

	if err := w.use.Sweep(t.Context(), synctest.SecondVault); err != nil {
		t.Fatal(err)
	}

	if _, isThere := w.getStagedSize(t, "0badc0de.part"); isThere {
		t.Error("a staged file that no download names is still there")
	}
	if _, isThere := w.getStagedSize(t, named); !isThere {
		t.Error("the staged file of a download that is kept was swept")
	}
	if _, isThere := w.getStagedSize(t, "notes.json"); !isThere {
		t.Error("a file of the plugin that is not a staged file was swept")
	}
}

func TestDownloadFile_DiscardDeletesTheStagedFileAndItsState(t *testing.T) {
	w := newWorld(t)
	stored, meta, _ := w.publishBig(t, "Videos/Trip.bin", 8)
	w.p.Server.StopAfterDownloadBytes(t, 20<<20)
	name, _ := w.use.Execute(t.Context(), w.input(stored, meta))

	if err := w.use.Discard(t.Context(), synctest.SecondVault, stored.GetRevision().GetFileId()); err != nil {
		t.Fatal(err)
	}

	if _, isThere := w.getStagedSize(t, name); isThere {
		t.Error("the staged file is still there after Discard")
	}
	if kept := w.readEntries(t); len(kept) != 0 {
		t.Errorf("entries = %+v, want none after Discard", kept)
	}
}
