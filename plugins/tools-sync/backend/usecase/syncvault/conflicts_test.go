package syncvault_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
)

const (
	list     = "Shopping list.md"
	listCopy = "Shopping list (conflict 2026-11-23).md"
)

func (p *pair) conflict(t *testing.T, path, first, second string) {
	t.Helper()
	p.a.WriteNote(source, path, "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.WriteNote(source, path, first)
	p.b.WriteNote(target, path, second)
	sync(t, p.a, source)
}

func TestSyncVault_ANoteChangedOnTwoComputersKeepsTheHeadAndTheOtherTextInACopyOnBoth(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.conflict(t, list, "bread", "eggs")

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.b, target, list, "bread")
	wantNote(t, p.b, target, listCopy, "eggs")
	wantNote(t, p.a, source, list, "bread")
	wantNote(t, p.a, source, listCopy, "eggs")
}

func TestSyncVault_TheConflictCopyIsPushedOnceAndNothingElseIs(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.conflict(t, list, "bread", "eggs")

	sync(t, p.b, target)
	sync(t, p.b, target)
	sync(t, p.a, source)

	if got := len(p.heads(t)); got != 2 {
		t.Errorf("server holds %d files, want the note and its copy", got)
	}
}

func TestSyncVault_WarnsOfTheNotesResolvedWithACopyAndClearsTheWarningOnTheNextPass(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.conflict(t, list, "bread", "eggs")

	sync(t, p.b, target)

	if !p.b.HasItem(syncvault.GetConflictText(1)) {
		t.Fatalf("health items = %+v, want the warning of 1 note changed on two computers", p.b.Health.List())
	}
	sync(t, p.b, target)
	if p.b.HasItemKey("conflicts") {
		t.Errorf("health items = %+v, want the warning cleared by a pass with no conflict", p.b.Health.List())
	}
}

func TestSyncVault_TheComputerThatReachedTheServerFirstHasNoWarning(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.conflict(t, list, "bread", "eggs")

	sync(t, p.a, source)

	if p.a.HasItemKey("conflicts") {
		t.Errorf("health items = %+v, want no warning on the computer whose text is the head", p.a.Health.List())
	}
}

func TestSyncVault_CountsOneConflictForEachNoteResolvedWithACopy(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	for _, path := range []string{"One.md", "Two.md"} {
		p.a.WriteNote(source, path, "milk")
	}
	sync(t, p.a, source)
	sync(t, p.b, target)
	for _, path := range []string{"One.md", "Two.md"} {
		p.a.WriteNote(source, path, "bread")
		p.b.WriteNote(target, path, "eggs")
	}
	sync(t, p.a, source)

	sync(t, p.b, target)

	if !p.b.HasItem("2 notes changed on two computers") {
		t.Errorf("health items = %+v, want 2 notes changed on two computers", p.b.Health.List())
	}
}

func TestSyncVault_TheSameTextOnBothComputersMakesNoCopyAndNoWarning(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.conflict(t, list, "bread", "bread")

	sync(t, p.b, target)
	sync(t, p.a, source)

	if _, ok := p.b.ReadNote(target, listCopy); ok {
		t.Error("a conflict copy of equal notes was made")
	}
	if p.b.HasItemKey("conflicts") || len(p.heads(t)) != 1 {
		t.Errorf("health items = %+v with %d files on the server, want no warning and the one note", p.b.Health.List(), len(p.heads(t)))
	}
}

func TestSyncVault_ANoteChangedOnOneComputerOnlyMakesNoCopy(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, list, "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.WriteNote(source, list, "bread")
	sync(t, p.a, source)

	sync(t, p.b, target)

	wantNote(t, p.b, target, list, "bread")
	if _, ok := p.b.ReadNote(target, listCopy); ok || len(p.heads(t)) != 1 {
		t.Errorf("copy there: %v with %d files on the server, want no copy", ok, len(p.heads(t)))
	}
}

func TestSyncVault_ASecondConflictKeepsTheFirstCopyAndMakesAnotherUnderANumber(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.conflict(t, list, "bread", "eggs")
	sync(t, p.b, target)
	sync(t, p.a, source)
	p.a.WriteNote(source, list, "rice")
	p.b.WriteNote(target, list, "tea")
	sync(t, p.a, source)

	sync(t, p.b, target)

	wantNote(t, p.b, target, list, "rice")
	wantNote(t, p.b, target, listCopy, "eggs")
	wantNote(t, p.b, target, "Shopping list (conflict 2026-11-23 2).md", "tea")
}

func TestSyncVault_AFileChangedOnTwoComputersKeepsTheHeadAndMakesNoCopyAndNoWarning(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.conflict(t, pdf, pdfText+" there", pdfText+" here")

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.b, target, pdf, pdfText+" there")
	wantNote(t, p.a, source, pdf, pdfText+" there")
	if p.b.HasItemKey("conflicts") || len(p.heads(t)) != 1 {
		t.Errorf("health items = %+v with %d files on the server, want no warning and no copy", p.b.Health.List(), len(p.heads(t)))
	}
}

func TestSyncVault_ANoteMadeOnBothComputersWithTheSameTextAtOnePathIsOneNote(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Daily/Monday.md", "tea")
	p.b.WriteNote(target, "Daily/Monday.md", "tea")
	sync(t, p.a, source)

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.b, target, "Daily/Monday.md", "tea")
	if _, ok := p.b.ReadNote(target, "Daily/Monday (conflict 2026-11-23).md"); ok || len(p.heads(t)) != 1 {
		t.Errorf("copy there: %v with %d files on the server, want one note and no copy", ok, len(p.heads(t)))
	}
}

func TestSyncVault_ANoteMadeOnBothComputersWithOtherTextAtOnePathKeepsTheHeadAndACopy(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Daily/Monday.md", "tea")
	p.b.WriteNote(target, "Daily/Monday.md", "rice")
	sync(t, p.a, source)

	sync(t, p.b, target)
	sync(t, p.a, source)

	for _, c := range []struct {
		name string
		got  func(path string) (string, bool)
	}{
		{"second computer", func(path string) (string, bool) { return p.b.ReadNote(target, path) }},
		{"first computer", func(path string) (string, bool) { return p.a.ReadNote(source, path) }},
	} {
		if text, _ := c.got("Daily/Monday.md"); text != "tea" {
			t.Errorf("note of the %s = %q, want the head tea", c.name, text)
		}
		if text, ok := c.got("Daily/Monday (conflict 2026-11-23).md"); !ok || text != "rice" {
			t.Errorf("copy of the %s = %q (there: %v), want rice", c.name, text, ok)
		}
	}
}

func TestSyncVault_ANoteThatTakesAPathBetweenThePullAndThePushIsSettledWithACopy(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.b.WriteNote(target, "Daily/Monday.md", "rice")
	p.server.AfterNext("/GetChanges", func() {
		p.a.WriteNote(source, "Daily/Monday.md", "tea")
		sync(t, p.a, source)
	})

	sync(t, p.b, target)

	wantNote(t, p.b, target, "Daily/Monday.md", "tea")
	wantNote(t, p.b, target, "Daily/Monday (conflict 2026-11-23).md", "rice")
}

func TestSyncVault_ANameThatMatchesAnotherNameInCaseOnlyIsNotSyncedAndRaisesAWarning(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Papers/Report.pdf", pdfText)
	sync(t, p.a, source)
	p.b.WriteNote(target, "Papers/report.pdf", "another report")

	sync(t, p.b, target)
	sync(t, p.a, source)

	if !p.b.HasItem(syncvault.GetCaseClashText(1)) {
		t.Errorf("health items = %+v, want the warning of 1 file whose name matches another name", p.b.Health.List())
	}
	if got := len(p.heads(t)); got != 1 {
		t.Errorf("server holds %d files, want only the first of the two names", got)
	}
	wantNote(t, p.b, target, "Papers/report.pdf", "another report")
	wantNote(t, p.b, target, "Papers/Report.pdf", pdfText)
	wantNoFile(t, p.a, source, "Papers/report.pdf")
}

func TestSyncVault_AWarningOfANameThatMatchesAnotherNameClearsOnceTheFileIsGone(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Papers/Report.pdf", pdfText)
	sync(t, p.a, source)
	p.b.WriteNote(target, "Papers/report.pdf", "another report")
	sync(t, p.b, target)
	p.b.Vaults.With(target).Remove("Papers/report.pdf")

	sync(t, p.b, target)

	if p.b.HasItemKey("case-clash") {
		t.Errorf("health items = %+v, want the warning cleared", p.b.Health.List())
	}
}

func TestSyncVault_ANoteRemovedOnBothComputersIsRemovedAndNothingIsRaised(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, list, "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove(list)
	p.b.Vaults.With(target).Remove(list)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)

	sync(t, p.b, target)
	p.afterDelay()
	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNoFile(t, p.a, source, list)
	wantNoFile(t, p.b, target, list)
	if got := len(p.getLiveHeads(t)); got != 0 {
		t.Errorf("live files on the server = %d, want none", got)
	}
	if len(p.b.Health.List()) != 0 {
		t.Errorf("health items = %+v, want none", p.b.Health.List())
	}
}

func TestSyncVault_ANoteChangedOnOneComputerAndRemovedOnTheOtherStaysWithNoCopy(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, list, "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.WriteNote(source, list, "bread")
	p.b.Vaults.With(target).Remove(list)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.afterDelay()
	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.a, source, list, "bread")
	wantNote(t, p.b, target, list, "bread")
	if _, ok := p.a.ReadNote(source, listCopy); ok || len(p.getLiveHeads(t)) != 1 {
		t.Errorf("copy there: %v with %d live files, want the note alone", ok, len(p.getLiveHeads(t)))
	}
}

func TestSyncVault_ANoteRemovedOnOneComputerAndChangedOnTheOtherStaysWithNoCopy(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, list, "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove(list)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	p.b.WriteNote(target, list, "eggs")

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.b, target, list, "eggs")
	wantNote(t, p.a, source, list, "eggs")
	if _, ok := p.b.ReadNote(target, listCopy); ok {
		t.Error("a conflict copy was made against a removal")
	}
}
