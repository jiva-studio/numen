package restore_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

func id(b byte) []byte { return bytes.Repeat([]byte{b}, 16) }

func TestGetVerdict(t *testing.T) {
	t.Parallel()
	keyA, keyB := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	tests := []struct {
		name   string
		memory restore.Memory
		look   restore.Look
		want   restore.Verdict
	}{
		{"no epoch anywhere", restore.Memory{Cursor: 5}, restore.Look{HeadSeq: 5}, restore.Same},
		{"same epoch", restore.Memory{EpochID: id(1), Cursor: 5}, restore.Look{EpochID: id(1), HeadSeq: 9}, restore.Same},
		{"first epoch after a link with none", restore.Memory{Cursor: 5}, restore.Look{EpochID: id(1), HeadSeq: 5}, restore.Changed},
		{"another epoch", restore.Memory{EpochID: id(1), Cursor: 5}, restore.Look{EpochID: id(2), HeadSeq: 5}, restore.Changed},
		{"epoch announced and head lower", restore.Memory{EpochID: id(1), Cursor: 9}, restore.Look{EpochID: id(2), HeadSeq: 3}, restore.Changed},
		{"head below the cursor without an epoch", restore.Memory{EpochID: id(1), Cursor: 9}, restore.Look{EpochID: id(1), HeadSeq: 3}, restore.WentBack},
		{"head equal to the cursor", restore.Memory{Cursor: 9}, restore.Look{HeadSeq: 9}, restore.Same},
		{"key not pinned yet", restore.Memory{EpochID: id(1)}, restore.Look{EpochID: id(2), SignKey: keyA}, restore.Changed},
		{"same pinned key", restore.Memory{EpochID: id(1), PinnedKey: keyA}, restore.Look{EpochID: id(2), SignKey: keyA}, restore.Changed},
		{"another pinned key", restore.Memory{EpochID: id(1), PinnedKey: keyA}, restore.Look{EpochID: id(2), SignKey: keyB}, restore.KeyChanged},
		{"another key with the same epoch", restore.Memory{EpochID: id(1), PinnedKey: keyA}, restore.Look{EpochID: id(1), SignKey: keyB}, restore.KeyChanged},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := restore.GetVerdict(tc.memory, tc.look); got != tc.want {
				t.Errorf("GetVerdict = %v, want %v", got, tc.want)
			}
		})
	}
}

func journal(entries ...string) restore.Journal {
	out := restore.Journal{Kind: restore.KindCatalog}
	for _, entry := range entries {
		out.Entries = append(out.Entries, []byte(entry))
	}
	return out
}

func TestCompareJournals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		cache, server restore.Journal
		want          restore.Relation
	}{
		{"same entries", journal("a", "b"), journal("a", "b"), restore.Equal},
		{"both empty", journal(), journal(), restore.Equal},
		{"server holds a prefix", journal("a", "b", "c"), journal("a", "b"), restore.ServerBehind},
		{"server holds nothing", journal("a", "b"), journal(), restore.ServerBehind},
		{"server holds more", journal("a"), journal("a", "b"), restore.ServerAhead},
		{"empty copy", journal(), journal("a"), restore.ServerAhead},
		{"another entry in the middle", journal("a", "b", "c"), journal("a", "x"), restore.Diverged},
		{"another first entry", journal("a"), journal("x", "b"), restore.Diverged},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := restore.CompareJournals(tc.cache, tc.server); got != tc.want {
				t.Errorf("CompareJournals = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetReplay_ReturnsTheEntriesTheServerLacksInOrder(t *testing.T) {
	t.Parallel()

	got, err := restore.GetReplay(journal("a", "b", "c"), journal("a"))

	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || string(got[0]) != "b" || string(got[1]) != "c" {
		t.Errorf("replay = %q, want b then c", got)
	}
}

func TestGetReplay_ReturnsNothingWhenTheServerHasTheWholeCopy(t *testing.T) {
	t.Parallel()
	for _, server := range []restore.Journal{journal("a", "b"), journal("a", "b", "c")} {
		got, err := restore.GetReplay(journal("a", "b"), server)
		if err != nil || len(got) != 0 {
			t.Errorf("replay against %d entries = %q, %v, want nothing", len(server.Entries), got, err)
		}
	}
}

func TestGetReplay_RefusesALogThatDiffersAtAnEntryBothHold(t *testing.T) {
	t.Parallel()

	got, err := restore.GetReplay(journal("a", "b", "c"), journal("a", "x"))

	if !errors.Is(err, restore.ErrDiverged) || len(got) != 0 {
		t.Errorf("replay = %q, %v, want nothing and ErrDiverged", got, err)
	}
}

func record(file byte, path, base string, head byte) restore.Known {
	return restore.Known{FileID: id(file), Path: path, BaseHash: base, HeadRev: bytes.Repeat([]byte{head}, 32)}
}

func head(file byte, path string, rev byte) restore.Head {
	return restore.Head{FileID: id(file), Path: path, Rev: bytes.Repeat([]byte{rev}, 32)}
}

func heads(all ...restore.Head) map[string]restore.Head {
	out := map[string]restore.Head{}
	for _, one := range all {
		out[string(one.FileID)] = one
	}
	return out
}

func wantAction(t *testing.T, got []restore.Action, kind restore.ActionKind, file byte) restore.Action {
	t.Helper()
	for _, action := range got {
		if bytes.Equal(action.FileID, id(file)) {
			if action.Kind != kind {
				t.Fatalf("action for file %d = %v, want %v", file, action.Kind, kind)
			}
			return action
		}
	}
	if kind != restore.Keep {
		t.Fatalf("no action for file %d, want %v", file, kind)
	}
	return restore.Action{}
}

func TestPlanRecovery_RecreatesAKnownFileWhoseHeadIsMissingOnTheServer(t *testing.T) {
	t.Parallel()
	records := []restore.Known{record(1, "Shopping list.md", note.Hash([]byte("milk")), 7)}
	local := map[string]string{"Shopping list.md": note.Hash([]byte("milk"))}

	got := restore.PlanRecovery(records, heads(), local)

	action := wantAction(t, got, restore.Recreate, 1)
	if action.Path != "Shopping list.md" || len(action.ParentRev) != 0 {
		t.Errorf("action = %+v, want the path and no parent", action)
	}
}

func TestPlanRecovery_SendsTheLocalNoteOverAnOlderHeadEvenWhenLocalEqualsTheBase(t *testing.T) {
	t.Parallel()
	bread := note.Hash([]byte("bread"))
	records := []restore.Known{record(1, "Shopping list.md", bread, 8)}
	local := map[string]string{"Shopping list.md": bread}

	got := restore.PlanRecovery(records, heads(head(1, "Shopping list.md", 7)), local)

	action := wantAction(t, got, restore.Send, 1)
	if !bytes.Equal(action.ParentRev, bytes.Repeat([]byte{7}, 32)) {
		t.Errorf("parent = %x, want the head the server holds", action.ParentRev)
	}
}

func TestPlanRecovery_SendsTheLocalNoteWhenTheHeadHasNoBody(t *testing.T) {
	t.Parallel()
	milk := note.Hash([]byte("milk"))
	records := []restore.Known{record(1, "a.md", milk, 7)}
	lost := head(1, "a.md", 7)
	lost.IsBlobMissing = true

	got := restore.PlanRecovery(records, heads(lost), map[string]string{"a.md": milk})

	wantAction(t, got, restore.Send, 1)
}

func TestPlanRecovery_KeepsAFileWhoseHeadIsTheOneThisDeviceRecorded(t *testing.T) {
	t.Parallel()
	milk := note.Hash([]byte("milk"))
	records := []restore.Known{record(1, "a.md", milk, 7)}

	got := restore.PlanRecovery(records, heads(head(1, "a.md", 7)), map[string]string{"a.md": milk})

	wantAction(t, got, restore.Keep, 1)
}

func TestPlanRecovery_TombstonesAFileRemovedLocallyWhoseHeadIsLiveAndOlder(t *testing.T) {
	t.Parallel()
	records := []restore.Known{record(1, "a.md", note.Hash([]byte("milk")), 8)}

	got := restore.PlanRecovery(records, heads(head(1, "a.md", 7)), map[string]string{})

	action := wantAction(t, got, restore.Tombstone, 1)
	if !bytes.Equal(action.ParentRev, bytes.Repeat([]byte{7}, 32)) {
		t.Errorf("parent = %x, want the head the server holds", action.ParentRev)
	}
}

func TestPlanRecovery_KeepsAFileRemovedLocallyWhoseHeadIsAlreadyDeleted(t *testing.T) {
	t.Parallel()
	records := []restore.Known{record(1, "a.md", note.Hash([]byte("milk")), 8)}
	gone := head(1, "a.md", 7)
	gone.IsDeleted = true

	got := restore.PlanRecovery(records, heads(gone), map[string]string{})

	wantAction(t, got, restore.Keep, 1)
}

func TestPlanRecovery_KeepsAFileRemovedLocallyAndMissingOnTheServer(t *testing.T) {
	t.Parallel()
	records := []restore.Known{record(1, "a.md", note.Hash([]byte("milk")), 8)}

	got := restore.PlanRecovery(records, heads(), map[string]string{})

	wantAction(t, got, restore.Keep, 1)
}

func TestPlanRecovery_SendsTheLocalNoteOverADeletedHead(t *testing.T) {
	t.Parallel()
	milk := note.Hash([]byte("milk"))
	records := []restore.Known{record(1, "a.md", milk, 8)}
	gone := head(1, "a.md", 7)
	gone.IsDeleted = true

	got := restore.PlanRecovery(records, heads(gone), map[string]string{"a.md": milk})

	wantAction(t, got, restore.Send, 1)
}

func TestPlanRecovery_PullsAHeadThisDeviceNeverKnewWhenNoLocalNoteHasItsPath(t *testing.T) {
	t.Parallel()

	got := restore.PlanRecovery(nil, heads(head(2, "New.md", 5)), map[string]string{})

	action := wantAction(t, got, restore.Pull, 2)
	if action.Path != "New.md" {
		t.Errorf("path = %q, want New.md", action.Path)
	}
}

func TestPlanRecovery_NeverReplacesALocalNoteWithAHeadThisDeviceNeverKnew(t *testing.T) {
	t.Parallel()

	got := restore.PlanRecovery(nil, heads(head(2, "New.md", 5)), map[string]string{"New.md": note.Hash([]byte("mine"))})

	wantAction(t, got, restore.Keep, 2)
}

func TestPlanRecovery_PullsNothingForAnUnknownHeadThatIsDeletedOrHasNoBody(t *testing.T) {
	t.Parallel()
	deleted, lost := head(2, "One.md", 5), head(3, "Two.md", 5)
	deleted.IsDeleted, lost.IsBlobMissing = true, true

	got := restore.PlanRecovery(nil, heads(deleted, lost), map[string]string{})

	wantAction(t, got, restore.Keep, 2)
	wantAction(t, got, restore.Keep, 3)
}

func TestPlanRecovery_HasNoActionThatWritesAPlainServerContentOverALocalNote(t *testing.T) {
	t.Parallel()
	milk := note.Hash([]byte("milk"))
	records := []restore.Known{record(1, "a.md", milk, 8), record(2, "b.md", milk, 8)}
	all := heads(head(1, "a.md", 7), head(2, "b.md", 8))
	local := map[string]string{"a.md": note.Hash([]byte("mine")), "b.md": milk}

	for _, action := range restore.PlanRecovery(records, all, local) {
		if action.Kind == restore.Pull {
			t.Errorf("action %+v pulls a note this device knows", action)
		}
	}
}
