package index

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// The column is an INTEGER of nanoseconds since the epoch, and it holds that
// number and no other.
func TestTheSourceRowHoldsNanosecondsSinceTheEpoch(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	const nanos = 1788433445123456789

	db := openDB(t)
	if err := db.Notes().Save(t.Context(), first.ID, []domain.Note{{
		Fingerprint: domain.Fingerprint{
			Path: "notes/Compost.md", Kind: domain.KindNote, Size: 12, ModTime: at,
		},
		Title: "Compost",
		Body:  "Leaves and peelings.",
	}}); err != nil {
		t.Fatal(err)
	}

	var held int64
	if err := db.read.QueryRowContext(t.Context(),
		`SELECT modified_at FROM sources WHERE path = ?`, "notes/Compost.md").Scan(&held); err != nil {
		t.Fatal(err)
	}
	if held != nanos {
		t.Errorf("the row holds modified_at %d, want %d", held, nanos)
	}

	found, err := db.NoteQueries().Fingerprints(t.Context(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if back := found["notes/Compost.md"]; !back.ModTime.Equal(at) {
		t.Errorf("the index read the file back at %v, and it changed at %v", back.ModTime, at)
	}
}

// A stamp a walk took carries a monotonic reading and a zone, and the column
// carries neither. A file read back off the index is still the file that was
// walked, or every scan indexes the whole vault again.
func TestAFileStampedByAClockIsUnchangedThroughTheIndex(t *testing.T) {
	at := time.Now()
	db := openDB(t)

	ref := domain.Fingerprint{
		Path: "notes/Leaves.md", Kind: domain.KindNote, Size: 12, ModTime: at,
	}
	if err := db.Notes().Save(t.Context(), first.ID, []domain.Note{{
		Fingerprint: ref, Title: "Leaves", Body: "Turned and rotted down.",
	}}); err != nil {
		t.Fatal(err)
	}

	found, err := db.NoteQueries().Fingerprints(t.Context(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if back := found["notes/Leaves.md"]; !back.Unchanged(ref) {
		t.Errorf("the index read the file back as %+v, and it is %+v", back, ref)
	}
}

// Nothing said about a file is not the epoch, either way round.
func TestNoStampIsNoTimeAndNotTheEpoch(t *testing.T) {
	if held := chunk.Stamp(time.Time{}); held != 0 {
		t.Errorf("no stamp goes into the column as %d", held)
	}
	if held := chunk.Instant(0); !held.IsZero() {
		t.Errorf("a column naming no time came back as %v", held)
	}
}
