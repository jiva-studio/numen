package index

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// The column is an INTEGER of nanoseconds since the epoch, and it holds that
// number and no other.
func TestTheSourceRowContainsNanosecondsSinceTheEpoch(t *testing.T) {
	pos := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	const nanos = 1788433445123456789

	db := openDB(t)
	if err := saveCut(t.Context(), db, first.ID, []domain.Note{{
		FileStat: domain.FileStat{
			Path: "notes/Compost.md", Kind: domain.KindNote, Size: 12, ModTime: pos,
		},
		Title: "Compost",
		Body:  "Leaves and peelings.",
	}}); err != nil {
		t.Fatal(err)
	}

	var target int64
	if err := db.read.QueryRowContext(t.Context(),
		`SELECT modified_at FROM sources WHERE path = ?`, "notes/Compost.md").Scan(&target); err != nil {
		t.Fatal(err)
	}
	if target != nanos {
		t.Errorf("the row holds modified_at %d, want %d", target, nanos)
	}

	found, err := db.NoteQueries().GetFileStats(t.Context(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if back := found["notes/Compost.md"]; !back.ModTime.Equal(pos) {
		t.Errorf("the index read the file back at %v, and it changed at %v", back.ModTime, pos)
	}
}

// A stamp a walk took carries a monotonic reading and a zone, and the column
// carries neither. A file read back off the index is still the file that was
// walked, or every scan indexes the whole vault again.
func TestAFileStampedByAClockIsUnchangedThroughTheIndex(t *testing.T) {
	pos := time.Now()
	db := openDB(t)

	ref := domain.FileStat{
		Path: "notes/Leaves.md", Kind: domain.KindNote, Size: 12, ModTime: pos,
	}
	if err := saveCut(t.Context(), db, first.ID, []domain.Note{{
		FileStat: ref, Title: "Leaves", Body: "Turned and rotted down.",
	}}); err != nil {
		t.Fatal(err)
	}

	found, err := db.NoteQueries().GetFileStats(t.Context(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if back := found["notes/Leaves.md"]; !back.IsUnchanged(ref) {
		t.Errorf("the index read the file back as %+v, and it is %+v", back, ref)
	}
}

// Nothing said about a file is not the epoch, either way round.
func TestNoStampIsNoTimeAndNotTheEpoch(t *testing.T) {
	if target := chunk.ConvertTimeToNanos(time.Time{}); target != 0 {
		t.Errorf("no stamp goes into the column as %d", target)
	}
	if targetVal := chunk.ConvertNanosToTime(0); !targetVal.IsZero() {
		t.Errorf("a column naming no time came back as %v", targetVal)
	}
}
