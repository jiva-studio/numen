package note_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

func TestFileSnapshot_IsSettledWhenItsTimeIsOldEnoughOrItWasSeenLongEnoughWhileItsTimeIsInTheFuture(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	const settle = 10 * time.Second
	at := func(offset time.Duration) int64 { return now.Add(offset).UnixNano() }
	tests := []struct {
		name string
		file note.FileSnapshot
		want bool
	}{
		{"written a minute ago", note.FileSnapshot{ModifiedAt: at(-time.Minute)}, true},
		{"written a second ago", note.FileSnapshot{ModifiedAt: at(-time.Second)}, false},
		{"future time, never seen", note.FileSnapshot{ModifiedAt: at(time.Hour)}, false},
		{"future time, seen a second ago", note.FileSnapshot{ModifiedAt: at(time.Hour), SeenAt: at(-time.Second)}, false},
		{"future time, seen ten seconds ago", note.FileSnapshot{ModifiedAt: at(time.Hour), SeenAt: at(-settle)}, true},
	}
	for _, test := range tests {
		if got := test.file.IsSettled(now, settle); got != test.want {
			t.Errorf("%s: IsSettled = %v, want %v", test.name, got, test.want)
		}
	}
}
