package pull_test

import (
	"context"
	"testing"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

// serving is a server that answers the first change request with the revisions it was given as a page ahead of the
// head, and every other request as the head server does.
type serving struct {
	ports.Revisions
	page *syncv1.GetChangesResponse
	used bool
}

func (s *serving) GetChanges(ctx context.Context, vaultID []byte, since uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	if s.used {
		return s.Revisions.GetChanges(ctx, vaultID, since, limit)
	}
	s.used = true
	return s.page, nil
}

func (p *pair) pullServing(t *testing.T, changes ...*syncv1.StoredRevision) {
	t.Helper()
	head, err := p.a.Cloud.GetChanges(t.Context(), p.vault.ID, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	deps := p.b.PullDeps()
	deps.Revisions = &serving{
		Revisions: deps.Revisions,
		page: &syncv1.GetChangesResponse{
			Changes: changes, NextSeq: head.GetHeadSeq() + 1, HeadSeq: head.GetHeadSeq() + 1, RestoreEpochId: head.GetRestoreEpochId(),
		},
	}
	if _, err := pull.NewPullChanges(deps).Execute(t.Context(), ports.PullInput{VaultID: target, IsAuthor: trustAll}); err != nil {
		t.Fatal(err)
	}
}

func TestPullChanges_KeepsTheNewerTextWhenTheServerServesAnOlderRevisionAgain(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Passwords.md", "old secret", nil)
	older := p.a.ReadChanges(t, p.vault.ID)
	p.push(t, "Passwords.md", "new secret", &first)
	p.pull(t, trustAll)

	p.pullServing(t, older...)

	if text, _ := p.b.ReadNote(target, "Passwords.md"); text != "new secret" {
		t.Errorf("note = %q, want new secret kept", text)
	}
	if !p.b.HasItemKey(pull.KeyRollback) {
		t.Error("no health item names the older revision")
	}
}

func TestPullChanges_KeepsTheRemovalWhenTheServerServesTheRemovedNoteAgain(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Draft.md", "draft", nil)
	p.pull(t, trustAll)
	older := p.a.ReadChanges(t, p.vault.ID)
	p.a.CommitRemoval(t, p.vault, "Draft.md", first)
	p.pull(t, trustAll)
	if _, isThere := p.b.ReadNote(target, "Draft.md"); isThere {
		t.Fatal("the removed note is still in the folder")
	}

	p.pullServing(t, older...)

	if _, isThere := p.b.ReadNote(target, "Draft.md"); isThere {
		t.Error("the removed note came back")
	}
	if !p.b.HasItemKey(pull.KeyRollback) {
		t.Error("no health item names the revision of the removed note")
	}
}

func TestPullChanges_KeepsTheNoteWhenTheServerServesAnOlderRemovalAgain(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Plan.md", "tea", nil)
	p.a.CommitRemoval(t, p.vault, "Plan.md", first)
	removal := p.a.ReadChanges(t, p.vault.ID)
	p.pull(t, trustAll)
	revived := p.push(t, "Plan.md", "coffee", &transfer.Outcome{FileID: first.FileID, RevID: removal[0].GetRevId(), Record: transfer.FileRecord{Version: 2}})
	p.pull(t, trustAll)
	if text, _ := p.b.ReadNote(target, "Plan.md"); text != "coffee" || len(revived.RevID) == 0 {
		t.Fatalf("note = %q, want coffee", text)
	}

	p.pullServing(t, removal...)

	if text, _ := p.b.ReadNote(target, "Plan.md"); text != "coffee" {
		t.Errorf("note = %q, want coffee kept", text)
	}
}

func TestPullChanges_ContinuingRevisionApplied(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Shopping list.md", "milk", nil)
	p.pull(t, trustAll)
	p.push(t, "Shopping list.md", "bread", &first)

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Shopping list.md"); text != "bread" {
		t.Errorf("note = %q, want bread", text)
	}
	if p.b.HasItemKey(pull.KeyRollback) {
		t.Error("a revision that continues the file was reported as older")
	}
}
