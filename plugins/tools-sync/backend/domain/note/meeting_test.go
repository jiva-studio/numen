package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

func TestMeet_SortsAPairByTheSideThatHasTheNoteAndByItsText(t *testing.T) {
	milk, bread := note.Hash([]byte("milk")), note.Hash([]byte("bread"))
	tests := map[string]struct {
		local, head string
		want        note.Meeting
	}{
		"equal texts":               {local: milk, head: milk, want: note.Same},
		"a note of the folder":      {local: milk, head: "", want: note.LocalOnly},
		"a note of the vault":       {local: "", head: milk, want: note.RemoteOnly},
		"different texts":           {local: milk, head: bread, want: note.Differ},
		"an empty text on one side": {local: note.Hash(nil), head: milk, want: note.Differ},
		"an empty text on both":     {local: note.Hash(nil), head: note.Hash(nil), want: note.Same},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := note.Meet(tc.local, tc.head); got != tc.want {
				t.Errorf("Meet(%q, %q) = %v, want %v", tc.local, tc.head, got, tc.want)
			}
		})
	}
}
