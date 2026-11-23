package pull_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

const (
	list     = "Shopping list.md"
	listCopy = "Shopping list (conflict 2026-11-23).md"
)

// countingResolver counts the calls to the resolver it wraps.
type countingResolver struct {
	inner ports.ConflictResolver
	calls int
}

func (c *countingResolver) Resolve(ctx context.Context, in ports.ConflictInput) (ports.ConflictOutcome, error) {
	c.calls++
	return c.inner.Resolve(ctx, in)
}

// failingResolver is a resolver that cannot settle a conflict.
type failingResolver struct{}

func (failingResolver) Resolve(context.Context, ports.ConflictInput) (ports.ConflictOutcome, error) {
	return ports.ConflictOutcome{Applied: nil, Copy: ""}, errors.New("the resolver cannot settle the conflict")
}

func (p *pair) pullWith(t *testing.T, resolver ports.ConflictResolver) transfer.PullResult {
	t.Helper()
	deps := p.b.PullDeps()
	deps.Conflicts = resolver
	result, err := pull.NewPullChanges(deps).Execute(t.Context(), transfer.PullInput{VaultID: target, IsAuthor: trustAll})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func (p *pair) count() *countingResolver {
	return &countingResolver{inner: p.b.PullDeps().Conflicts, calls: 0}
}

func (p *pair) conflictOn(t *testing.T, path, there, here string) {
	t.Helper()
	first := p.push(t, path, "milk", nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, path, here)
	p.push(t, path, there, &first)
}

func TestPullChanges_SettlesANoteChangedOnBothComputersWithTheHeadAndACopyOfTheLocalText(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, list, "bread", "eggs")

	result := p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, list); text != "bread" {
		t.Errorf("note = %q, want the head bread", text)
	}
	if text, ok := p.b.ReadNote(target, listCopy); !ok || text != "eggs" {
		t.Errorf("copy = %q (there: %v), want the local text eggs", text, ok)
	}
	if !slices.Equal(result.Resolved, []string{list}) || len(result.Conflicts) != 0 {
		t.Errorf("result = %+v, want the note resolved and none held", result)
	}
}

func TestPullChanges_MovesTheRecordOfASettledNoteToTheHeadAndDoesNotHoldThePosition(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, list, "bread", "eggs")

	p.pull(t, trustAll)

	row, ok := p.b.GetFileRow(t, target, list)
	if !ok || row.BaseHash != note.Hash([]byte("bread")) || len(row.HeadRev) == 0 {
		t.Errorf("row = %+v (there: %v), want the hash of bread as base and the head recorded", row, ok)
	}
	changes := p.b.ReadChanges(t, p.b.GetRemoteID(t, target))
	if got, want := p.b.Link(t, target).ServerSeq, changes[len(changes)-1].GetSeq(); got != want {
		t.Errorf("position = %d, want %d: the settled note does not hold it", got, want)
	}
}

func TestPullChanges_KeepsNoRecordOfTheCopySoThePassSendsItAsANewFile(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, list, "bread", "eggs")

	p.pull(t, trustAll)

	if _, ok := p.b.GetFileRow(t, target, listCopy); ok {
		t.Error("the copy has a record, want a new local file with none")
	}
}

func TestPullChanges_SettlesANoteOnceAndMakesNoSecondCopyOnTheNextPull(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, list, "bread", "eggs")
	p.pull(t, trustAll)

	again := p.pull(t, trustAll)

	if len(again.Resolved) != 0 || len(again.Conflicts) != 0 {
		t.Errorf("second pull = %+v, want nothing to settle", again)
	}
	if _, ok := p.b.ReadNote(target, "Shopping list (conflict 2026-11-23 2).md"); ok {
		t.Error("a second copy of the same text was made")
	}
}

func TestPullChanges_MakesANumberedCopyForASecondConflictAndKeepsTheFirstCopy(t *testing.T) {
	p := newPair(t)
	first := p.push(t, list, "milk", nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, list, "eggs")
	second := p.push(t, list, "bread", &first)
	p.pull(t, trustAll)
	p.b.WriteNote(target, list, "tea")
	p.push(t, list, "rice", &second)

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, listCopy); text != "eggs" {
		t.Errorf("first copy = %q, want eggs kept", text)
	}
	if text, _ := p.b.ReadNote(target, "Shopping list (conflict 2026-11-23 2).md"); text != "tea" {
		t.Errorf("second copy = %q, want tea", text)
	}
}

func TestPullChanges_CallsTheResolverOnlyWhenBothComputersChangedTheNote(t *testing.T) {
	p := newPair(t)
	resolver := p.count()
	first := p.push(t, list, "milk", nil)
	p.pullWith(t, resolver)
	second := p.push(t, list, "bread", &first)
	p.pullWith(t, resolver)
	p.b.WriteNote(target, list, "bread and eggs")
	p.push(t, list, "bread and eggs", &second)

	p.pullWith(t, resolver)

	if resolver.calls != 0 {
		t.Errorf("resolver called %d times for a download and for equal text, want none", resolver.calls)
	}
}

func TestPullChanges_CallsTheResolverOnceForANoteChangedOnBothComputers(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, list, "bread", "eggs")
	resolver := p.count()

	p.pullWith(t, resolver)

	if resolver.calls != 1 {
		t.Errorf("resolver called %d times, want once", resolver.calls)
	}
}

func TestPullChanges_AdoptsAHeadThatHasTheContentOfTheBaseAndKeepsTheLocalChange(t *testing.T) {
	p := newPair(t)
	resolver := p.count()
	first := p.push(t, list, "milk", nil)
	p.pullWith(t, resolver)
	p.b.WriteNote(target, list, "eggs")
	p.push(t, list, "milk", &first)

	result := p.pullWith(t, resolver)

	if text, _ := p.b.ReadNote(target, list); text != "eggs" {
		t.Errorf("note = %q, want the local text eggs kept", text)
	}
	row, ok := p.b.GetFileRow(t, target, list)
	if !ok || row.BaseHash != note.Hash([]byte("milk")) || len(row.HeadRev) == 0 || resolver.calls != 0 || len(result.Resolved) != 0 {
		t.Errorf("row = %+v (there: %v), %d resolver calls, resolved %v; want the new head recorded and no resolve", row, ok, resolver.calls, result.Resolved)
	}
}

func TestPullChanges_KeepsTheNoteAndWarnsWhenTheResolverFails(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, list, "bread", "eggs")

	result := p.pullWith(t, failingResolver{})
	again := p.pullWith(t, failingResolver{})

	if text, _ := p.b.ReadNote(target, list); text != "eggs" {
		t.Errorf("note = %q, want the local text eggs", text)
	}
	if !slices.Equal(result.Conflicts, []string{list}) || !slices.Equal(again.Conflicts, []string{list}) {
		t.Errorf("conflicts = %v then %v, want the note both times", result.Conflicts, again.Conflicts)
	}
	if row, _ := p.b.GetFileRow(t, target, list); row.BaseHash != note.Hash([]byte("milk")) {
		t.Errorf("base = %s, want the hash of milk, unchanged", row.BaseHash)
	}
	if text, ok := p.b.ReadNote(target, listCopy); ok {
		t.Errorf("copy = %q, want none when the resolver failed", text)
	}
}

func TestPullChanges_ReplacesAFileChangedOnBothComputersWithTheHeadThroughPlaceFileWithBaseline(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, pdf, "edited here")
	p.push(t, pdf, pdfText+" edited there", &first)

	result := p.pull(t, trustAll)

	if got := p.readNote(t, pdf); got != pdfText+" edited there" {
		t.Errorf("file = %q, want the head", got)
	}
	places := spy.GetPlaces()
	if len(places) != 2 || places[1].Baseline == nil || places[1].Baseline.Hash != note.Hash([]byte("edited here")) {
		t.Errorf("placements = %+v, want the second to replace the local file it read", places)
	}
	if len(result.Resolved) != 0 || len(result.Conflicts) != 0 {
		t.Errorf("result = %+v, want no copy and no hold for a file that is not a note", result)
	}
	if row, ok := p.b.GetFileRow(t, target, pdf); !ok || row.BaseHash != note.Hash([]byte(pdfText+" edited there")) {
		t.Errorf("row = %+v (there: %v), want the head as base", row, ok)
	}
}

func TestPullChanges_MakesNoCopyOfAFileThatIsNotANote(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, pdf, pdfText+" there", pdfText+" here")

	p.pull(t, trustAll)

	if _, ok := p.b.ReadNote(target, "Papers/Report (conflict 2026-11-23).md"); ok {
		t.Error("a copy of a file that is not a note was made")
	}
}

func TestPullChanges_SettlesAFileThatAppearedBeforeItsPlacementWithTheHead(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	isFirst := true
	spy.RunBeforePlace(func(call synctest.PlaceCall) {
		if isFirst {
			isFirst = false
			p.b.WriteNote(target, call.Path, "written meanwhile")
		}
	})
	p.push(t, pdf, pdfText, nil)

	result := p.pull(t, trustAll)

	if got := p.readNote(t, pdf); got != pdfText {
		t.Errorf("file = %q, want the head over the file that appeared", got)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %v, want none", result.Conflicts)
	}
	if row, ok := p.b.GetFileRow(t, target, pdf); !ok || row.BaseHash != note.Hash([]byte(pdfText)) {
		t.Errorf("row = %+v (there: %v), want the head as base", row, ok)
	}
	if names := p.getStagedNames(t); len(names) != 0 {
		t.Errorf("staged files left = %v, want none", names)
	}
}

func TestPullChanges_MovesANoteChangedHereToTheNewPathWithItsChangeAndMakesNoCopy(t *testing.T) {
	p := newPair(t)
	first := p.push(t, list, "milk", nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, list, "eggs")
	p.move(t, transfer.Move{Record: first.Record, To: "Groceries.md"})

	result := p.pull(t, trustAll)

	if text, ok := p.b.ReadNote(target, "Groceries.md"); !ok || text != "eggs" {
		t.Errorf("note at the new path = %q (there: %v), want the local change eggs", text, ok)
	}
	if _, ok := p.b.ReadNote(target, list); ok {
		t.Error("the note is still at its old path")
	}
	if len(result.Resolved) != 0 || len(result.Conflicts) != 0 {
		t.Errorf("result = %+v, want no copy and no hold", result)
	}
	if _, ok := p.b.ReadNote(target, "Groceries (conflict 2026-11-23).md"); ok {
		t.Error("a conflict copy of a note renamed on one computer was made")
	}
}

func TestPullChanges_MovesAFileChangedHereToTheNewPathWithItsChange(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, pdf, "edited here")
	p.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})

	result := p.pull(t, trustAll)

	if got := p.readNote(t, "Papers/Old report.pdf"); got != "edited here" {
		t.Errorf("file at the new path = %q, want the local bytes", got)
	}
	if moves := spy.GetMoves(); len(moves) != 1 || moves[0].Baseline == nil || moves[0].Baseline.Hash != note.Hash([]byte("edited here")) {
		t.Errorf("moves = %+v, want one with the stat of the edited file", moves)
	}
	if len(result.Resolved) != 0 || len(result.Conflicts) != 0 {
		t.Errorf("result = %+v, want no copy and no hold", result)
	}
}

func TestPullChanges_KeepsTheEditOfAMovedFileForThePushBySettingTheBaseToTheContentOfTheHead(t *testing.T) {
	p := newPair(t)
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, pdf, "edited here")
	p.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})

	p.pull(t, trustAll)

	row, ok := p.b.GetFileRow(t, target, "Papers/Old report.pdf")
	if !ok || row.BaseHash != note.Hash([]byte(pdfText)) {
		t.Errorf("row = %+v (there: %v), want the content of the head as base, so the edit is a local change", row, ok)
	}
}

func TestPullChanges_SettlesANoteMovedOntoAPathTheLocalNoteHasWithTheHeadAndACopy(t *testing.T) {
	p := newPair(t)
	first := p.push(t, list, "milk", nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, "Groceries.md", "eggs")
	p.move(t, transfer.Move{Record: first.Record, To: "Groceries.md"})

	result := p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Groceries.md"); text != "milk" {
		t.Errorf("note at the new path = %q, want the head milk", text)
	}
	if text, ok := p.b.ReadNote(target, "Groceries (conflict 2026-11-23).md"); !ok || text != "eggs" {
		t.Errorf("copy = %q (there: %v), want the local text eggs", text, ok)
	}
	if len(result.Conflicts) != 0 || !slices.Contains(result.Resolved, "Groceries.md") {
		t.Errorf("result = %+v, want the new path resolved", result)
	}
}

func TestPullChanges_SettlesAFileMovedOntoAPathTheLocalFileHasWithTheHeadAndNoCopy(t *testing.T) {
	p := newPair(t)
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, "Papers/Old report.pdf", "another file")
	p.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})

	result := p.pull(t, trustAll)

	if got := p.readNote(t, "Papers/Old report.pdf"); got != pdfText {
		t.Errorf("file at the new path = %q, want the head", got)
	}
	if len(result.Conflicts) != 0 || len(result.Resolved) != 0 {
		t.Errorf("result = %+v, want the file settled with no copy", result)
	}
}

func TestPullChanges_KeepsANoteChangedHereAndRemovedThereWithNoResolveAndNoCopy(t *testing.T) {
	p := newPair(t)
	resolver := p.count()
	first := p.push(t, list, "milk", nil)
	p.pullWith(t, resolver)
	p.b.WriteNote(target, list, "eggs")
	p.remove(t, first.Record)

	result := p.pullWith(t, resolver)

	if text, _ := p.b.ReadNote(target, list); text != "eggs" {
		t.Errorf("note = %q, want the local text kept", text)
	}
	if resolver.calls != 0 || len(result.Resolved) != 0 || len(result.Conflicts) != 0 {
		t.Errorf("%d resolver calls and result %+v, want none: a removal holds no text", resolver.calls, result)
	}
	if _, ok := p.b.ReadNote(target, listCopy); ok {
		t.Error("a conflict copy was made against a removal")
	}
}

func TestPullChanges_RestoresANoteRemovedHereAndChangedThereWithNoResolveAndNoCopy(t *testing.T) {
	p := newPair(t)
	resolver := p.count()
	first := p.push(t, list, "milk", nil)
	p.pullWith(t, resolver)
	p.b.Vaults.With(target).Remove(list)
	p.push(t, list, "bread", &first)

	p.pullWith(t, resolver)

	if text, ok := p.b.ReadNote(target, list); !ok || text != "bread" {
		t.Errorf("note = %q (there: %v), want the change of the other computer", text, ok)
	}
	if _, ok := p.b.ReadNote(target, listCopy); ok || resolver.calls != 0 {
		t.Errorf("copy there: %v with %d resolver calls, want none", ok, resolver.calls)
	}
}

func TestPullChanges_CreatesNoSecondCopyWhenARunFailedAfterTheCopyWasCreated(t *testing.T) {
	p := newPair(t)
	p.conflictOn(t, list, "bread", "eggs")
	p.b.WriteNote(target, listCopy, "eggs")

	p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, list); text != "bread" {
		t.Errorf("note = %q, want the head bread", text)
	}
	if _, ok := p.b.ReadNote(target, "Shopping list (conflict 2026-11-23 2).md"); ok {
		t.Error("a second copy of the same text was made")
	}
}
