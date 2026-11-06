package index

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

const (
	hashVault  domain.VaultID = "01HASHVAULT"
	riversPath                = "library/rivers.pdf"
	// riversHash and otherHash are two SHA-256 sums in hex.
	riversHash = "4f2c8e0b9d7a61e3c5b2a09f8d7e6c5b4a3928170f6e5d4c3b2a1908f7e6d5c4"
	otherHash  = "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
)

// getRiversTime is a modification time with nanoseconds, as a file system gives it.
func getRiversTime() time.Time { return time.Date(2026, 6, 2, 14, 30, 5, 123456789, time.UTC) }

func openHashIndex(t *testing.T) *DB {
	t.Helper()
	db, err := Open(t.Context(), filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	if err := db.Vaults().Register(t.Context(), hashVault); err != nil {
		t.Fatal(err)
	}
	return db
}

// scan writes the row a scan of the vault writes for a file.
func scan(t *testing.T, db *DB, path string, size int64, modTime time.Time) {
	t.Helper()
	if err := db.Sources().SaveSource(t.Context(), hashVault, domain.Source{
		FileStat: domain.FileStat{Path: path, Kind: domain.KindFile, Size: size, ModTime: modTime},
	}); err != nil {
		t.Fatal(err)
	}
}

func getRiversRecord() textport.ContentHash {
	return textport.ContentHash{
		Path: riversPath, Hash: riversHash, Size: 2048, ModTime: getRiversTime(), FileID: 4242,
		HashTime: getRiversTime().Add(90*time.Second + 7),
	}
}

func saveContentHash(t *testing.T, db *DB, record textport.ContentHash) {
	t.Helper()
	if err := db.TextVersions().SaveContentHash(t.Context(), hashVault, record); err != nil {
		t.Fatal(err)
	}
}

func findContentHash(t *testing.T, db *DB, path string) (textport.ContentHash, bool) {
	t.Helper()
	got, ok, err := db.TextVersions().FindContentHash(t.Context(), hashVault, path)
	if err != nil {
		t.Fatal(err)
	}
	return got, ok
}

func isSameContentHash(a, b textport.ContentHash) bool {
	return a.Path == b.Path && a.Hash == b.Hash && a.Size == b.Size && a.ModTime.Equal(b.ModTime) &&
		a.FileID == b.FileID && a.HashTime.Equal(b.HashTime)
}

func TestASavedContentHashIsFoundWithItsNanosecondStatFileIDAndHashTime(t *testing.T) {
	db := openHashIndex(t)
	scan(t, db, riversPath, 2048, getRiversTime())
	saveContentHash(t, db, getRiversRecord())

	got, ok := findContentHash(t, db, riversPath)
	if !ok || !isSameContentHash(got, getRiversRecord()) {
		t.Fatalf("found %+v (%v), want %+v", got, ok, getRiversRecord())
	}
}

func TestAScannedRowHasNoHashNoFileIDAndNoHashTime(t *testing.T) {
	db := openHashIndex(t)
	scan(t, db, riversPath, 2048, getRiversTime())

	got, ok := findContentHash(t, db, riversPath)
	want := textport.ContentHash{Path: riversPath, Size: 2048, ModTime: getRiversTime()}
	if !ok || !isSameContentHash(got, want) {
		t.Fatalf("found %+v (%v), want %+v", got, ok, want)
	}
}

func TestAContentHashOfAnotherStatAndAnotherHashIsNotSaved(t *testing.T) {
	db := openHashIndex(t)
	scan(t, db, riversPath, 2048, getRiversTime())
	saveContentHash(t, db, getRiversRecord())

	changed := getRiversRecord()
	changed.Hash, changed.Size, changed.ModTime = otherHash, 4096, getRiversTime().Add(time.Minute)
	changed.FileID, changed.HashTime = 4243, getRiversTime().Add(2*time.Minute)
	saveContentHash(t, db, changed)

	if got, _ := findContentHash(t, db, riversPath); !isSameContentHash(got, getRiversRecord()) {
		t.Fatalf("the row became %+v, want it kept as %+v", got, getRiversRecord())
	}
}

func TestAContentHashOfTheSameHashTakesTheNewStat(t *testing.T) {
	db := openHashIndex(t)
	scan(t, db, riversPath, 2048, getRiversTime())
	saveContentHash(t, db, getRiversRecord())

	touched := getRiversRecord()
	touched.ModTime, touched.FileID, touched.HashTime = getRiversTime().Add(time.Minute+3), 4243, getRiversTime().Add(2*time.Minute)
	saveContentHash(t, db, touched)

	if got, _ := findContentHash(t, db, riversPath); !isSameContentHash(got, touched) {
		t.Fatalf("the row became %+v, want %+v", got, touched)
	}
}

func TestAContentHashOfAFileWithNoRowIsNotSaved(t *testing.T) {
	db := openHashIndex(t)
	record := getRiversRecord()
	record.Path = "library/unscanned.pdf"
	saveContentHash(t, db, record)

	if got, ok := findContentHash(t, db, record.Path); ok {
		t.Fatalf("a file with no row has %+v", got)
	}
	var rows int
	if err := db.read.QueryRowContext(t.Context(), `SELECT count(*) FROM sources`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("saving the hash of a file with no row made %d rows", rows)
	}
}

func TestAContentHashOfAnotherVaultIsNotFound(t *testing.T) {
	db := openHashIndex(t)
	scan(t, db, riversPath, 2048, getRiversTime())
	saveContentHash(t, db, getRiversRecord())
	if err := db.Vaults().Register(t.Context(), "01OTHERVAULT"); err != nil {
		t.Fatal(err)
	}

	if got, ok, err := db.TextVersions().FindContentHash(t.Context(), "01OTHERVAULT", riversPath); err != nil || ok {
		t.Fatalf("another vault has %+v (%v, %v)", got, ok, err)
	}
}

// readHashColumns is the file id and the hash time of the row, NULL as invalid.
func readHashColumns(t *testing.T, db *DB, path string) (sql.NullInt64, sql.NullInt64) {
	t.Helper()
	var fileID, hashTime sql.NullInt64
	if err := db.read.QueryRowContext(t.Context(),
		`SELECT file_id, content_hash_time FROM sources WHERE path = ?`, path).Scan(&fileID, &hashTime); err != nil {
		t.Fatal(err)
	}
	return fileID, hashTime
}

func TestAScanOfTheFileClearsItsFileIDAndHashTime(t *testing.T) {
	db := openHashIndex(t)
	scan(t, db, riversPath, 2048, getRiversTime())
	saveContentHash(t, db, getRiversRecord())
	if fileID, hashTime := readHashColumns(t, db, riversPath); !fileID.Valid || !hashTime.Valid {
		t.Fatalf("the saved row has file id %v and hash time %v", fileID, hashTime)
	}

	scan(t, db, riversPath, 2048, getRiversTime())

	if fileID, hashTime := readHashColumns(t, db, riversPath); fileID.Valid || hashTime.Valid {
		t.Fatalf("after a scan the row has file id %v and hash time %v, want both NULL", fileID, hashTime)
	}
	got, _ := findContentHash(t, db, riversPath)
	if got.Hash != "" || got.FileID != 0 || !got.HashTime.IsZero() {
		t.Fatalf("after a scan the row is %+v, want no hash, file id or hash time", got)
	}
}

func TestASaveOfTheNoteClearsItsFileIDAndHashTime(t *testing.T) {
	db := openHashIndex(t)
	const path = "notes/rivers.md"
	note := noteAt(path, "Rivers", "deltas and estuaries")
	saveNote := func() {
		t.Helper()
		if err := db.Notes().Save(t.Context(), hashVault, []domain.NoteChunks{{Note: note, Recipe: "cut"}}); err != nil {
			t.Fatal(err)
		}
	}
	saveNote()
	saveContentHash(t, db, textport.ContentHash{
		Path: path, Hash: riversHash, Size: note.FileStat.Size, ModTime: note.FileStat.ModTime, FileID: 51,
		HashTime: note.FileStat.ModTime.Add(time.Hour),
	})
	if fileID, hashTime := readHashColumns(t, db, path); !fileID.Valid || !hashTime.Valid {
		t.Fatalf("the saved note row has file id %v and hash time %v", fileID, hashTime)
	}

	saveNote()

	if fileID, hashTime := readHashColumns(t, db, path); fileID.Valid || hashTime.Valid {
		t.Fatalf("after the note is saved again its row has file id %v and hash time %v, want both NULL", fileID, hashTime)
	}
}
