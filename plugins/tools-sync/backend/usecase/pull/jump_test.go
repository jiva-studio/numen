package pull_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

func TestPullChanges_AppliesARevisionThatContinuesARemovedFile(t *testing.T) {
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
