package pull_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

func (p *pair) getHead(t *testing.T) uint64 {
	t.Helper()
	var head uint64
	for _, stored := range p.a.ReadChanges(t, p.vault.ID) {
		head = max(head, stored.GetSeq())
	}
	return head
}

func TestPullChanges_RecordsTheHighestHeadItSaw(t *testing.T) {
	p := newPair(t)
	p.push(t, "Shopping list.md", "milk", nil)
	p.push(t, "Plan.md", "tea", nil)

	p.pull(t, trustAll)

	if link := p.b.Link(t, target); link.HighSeq != p.getHead(t) || link.HighSeq == 0 {
		t.Errorf("highest head = %d, want the head of the server %d", link.HighSeq, p.getHead(t))
	}
}

func TestPullChanges_ReportsSameWhenTheServerGrowsBetweenPulls(t *testing.T) {
	p := newPair(t)
	for _, text := range []string{"milk", "bread", "eggs"} {
		p.push(t, text+".md", text, nil)
		p.push(t, "Plan "+text+".md", text, nil)

		result := p.pull(t, trustAll)

		if result.Verdict != restore.Same {
			t.Fatalf("verdict after %s = %v, want Same", text, result.Verdict)
		}
	}
	if link := p.b.Link(t, target); link.HighSeq != p.getHead(t) {
		t.Errorf("highest head = %d, want the head %d", link.HighSeq, p.getHead(t))
	}
}

func TestPullChanges_ReportsSameForANewDeviceOnAVaultWithAHistory(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Shopping list.md", "milk", nil)
	p.push(t, "Shopping list.md", "bread", &first)
	p.push(t, "Plan.md", "tea", nil)

	result := p.pull(t, trustAll)

	if result.Verdict != restore.Same || len(result.Applied) != 2 {
		t.Errorf("result = %+v, want Same and both notes applied", result)
	}
}

func TestPullChanges_KeepsTheHighestHeadForAFreshUseCaseAndStaysSameOnTheSameServer(t *testing.T) {
	p := newPair(t)
	p.push(t, "Shopping list.md", "milk", nil)
	p.pull(t, trustAll)
	before := p.b.Link(t, target)

	result := p.pull(t, trustAll)

	if result.Verdict != restore.Same {
		t.Errorf("verdict = %v, want Same", result.Verdict)
	}
	if after := p.b.Link(t, target); after.HighSeq != before.HighSeq || before.HighSeq == 0 {
		t.Errorf("highest head = %d after a restart, want %d kept", after.HighSeq, before.HighSeq)
	}
}
