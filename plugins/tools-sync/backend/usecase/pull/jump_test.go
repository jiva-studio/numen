package pull_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

const hugeVersion = 1 << 62

func atVersion(outcome transfer.Outcome, version uint64) *transfer.Outcome {
	outcome.Record.Version = version
	return &outcome
}

func TestPullChanges_RefusesARevisionWhoseVersionTheHistoryDoesNotExplain(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Plan.md", "tea", nil)
	p.pull(t, trustAll)
	p.push(t, "Plan.md", "coffee", atVersion(first, hugeVersion))

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Plan.md"); text != "tea" {
		t.Errorf("note = %q, want tea kept: the revision jumps to a version nothing explains", text)
	}
	if !p.b.HasItemKey(healthkey.Create(pull.KeyVersionJump, target)) {
		t.Error("no health item names the revision with the version that jumps")
	}
}

func TestPullChanges_StillAppliesTheOtherFilesWhenOneFileJumpsInVersion(t *testing.T) {
	p := newPair(t)
	plan := p.push(t, "Plan.md", "tea", nil)
	milk := p.push(t, "Milk.md", "one", nil)
	p.pull(t, trustAll)
	p.push(t, "Plan.md", "coffee", atVersion(plan, hugeVersion))
	p.push(t, "Milk.md", "two", &milk)

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Milk.md"); text != "two" {
		t.Errorf("note = %q, want two: the jump of another file must not stop the pull", text)
	}
	if text, _ := p.b.ReadNote(target, "Plan.md"); text != "tea" {
		t.Errorf("note = %q, want tea kept", text)
	}
}

func TestPullChanges_ReportsTheJumpAgainOnTheNextPullWhileTheRevisionIsStillThere(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Plan.md", "tea", nil)
	p.pull(t, trustAll)
	p.push(t, "Plan.md", "coffee", atVersion(first, hugeVersion))
	p.pull(t, trustAll)

	p.pull(t, trustAll)

	if !p.b.HasItemKey(healthkey.Create(pull.KeyVersionJump, target)) {
		t.Error("the health item went away on the next pull although the revision is still refused")
	}
}

func TestPullChanges_RefusesARemovalWhoseVersionJumps(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Plan.md", "tea", nil)
	p.pull(t, trustAll)
	p.a.CommitRemoval(t, p.vault, "Plan.md", *atVersion(first, hugeVersion))

	p.pull(t, trustAll)

	if text, isThere := p.b.ReadNote(target, "Plan.md"); !isThere || text != "tea" {
		t.Errorf("note = %q (there: %v), want tea kept: the removal jumps in version", text, isThere)
	}
	if !p.b.HasItemKey(healthkey.Create(pull.KeyVersionJump, target)) {
		t.Error("no health item names the removal")
	}
}

func TestPullChanges_PlantsNoFloorFromARemovalOfAFileItNeverSawWhenTheVersionJumps(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Plan.md", "tea", nil)
	p.a.CommitRemoval(t, p.vault, "Plan.md", *atVersion(first, hugeVersion))
	p.pull(t, trustAll)

	link := p.b.Link(t, target)
	if floor := link.GetFloor(first.FileID); floor != 0 {
		t.Errorf("floor = %d, want none: a version nothing explains must not freeze the file", floor)
	}
}

func TestPullChanges_AcceptsAFileOfADeepHistoryOnADeviceThatStartsInTheMiddle(t *testing.T) {
	p := newPair(t)
	p.push(t, "Log.md", "entries", atVersion(transfer.Outcome{FileID: []byte("0123456789abcdef"), Record: transfer.FileRecord{}}, 4999))

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Log.md"); text != "entries" {
		t.Errorf("note = %q, want entries: version 5000 of a file is an old file, not a jump", text)
	}
	if p.b.HasItemKey(healthkey.Create(pull.KeyVersionJump, target)) {
		t.Error("a file with a long history was reported as a jump")
	}
}

func TestPullChanges_AcceptsAGapOfAFewThousandVersionsAfterARestoredServer(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Plan.md", "tea", nil)
	p.pull(t, trustAll)
	p.push(t, "Plan.md", "coffee", atVersion(first, first.Record.Version+3000))

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Plan.md"); text != "coffee" {
		t.Errorf("note = %q, want coffee: a restored server loses edits and the author continues above them", text)
	}
}

func TestPullChanges_AcceptsTheRevisionOfAFileThatWasRemovedAndWrittenAgain(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Plan.md", "tea", nil)
	p.pull(t, trustAll)
	p.a.CommitRemoval(t, p.vault, "Plan.md", first)
	removal := p.a.ReadChanges(t, p.vault.ID)
	p.pull(t, trustAll)
	p.push(t, "Plan.md", "coffee", &transfer.Outcome{FileID: first.FileID, RevID: removal[0].GetRevId(), Record: transfer.FileRecord{Version: 2}})

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Plan.md"); text != "coffee" {
		t.Errorf("note = %q, want coffee", text)
	}
	if p.b.HasItemKey(healthkey.Create(pull.KeyVersionJump, target)) {
		t.Error("a file written again after a removal was reported as a jump")
	}
}
