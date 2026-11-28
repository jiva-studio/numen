package deletions_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
)

const vaultID domain.VaultID = "vault-1"

type pendingStore struct {
	states map[domain.VaultID]transfer.PausedRemovals
}

func (p *pendingStore) ReadPending(_ context.Context, id domain.VaultID) (*transfer.PausedRemovals, error) {
	state, ok := p.states[id]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (p *pendingStore) WritePending(_ context.Context, id domain.VaultID, state transfer.PausedRemovals) error {
	p.states[id] = state
	return nil
}

func (p *pendingStore) DeletePending(_ context.Context, id domain.VaultID) error {
	delete(p.states, id)
	return nil
}

type healthItems struct{ items map[string]health.Item }

func (h *healthItems) Report(_ context.Context, item health.Item) error {
	h.items[item.Key] = item
	return nil
}

func (h *healthItems) Clear(_ context.Context, key string) error {
	delete(h.items, key)
	return nil
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type setup struct {
	store  *pendingStore
	health *healthItems
	guard  *deletions.GuardDeletions
	answer *deletions.ResolveDeletions
	now    time.Time
}

func newSetup() *setup {
	s := &setup{
		store:  &pendingStore{states: map[domain.VaultID]transfer.PausedRemovals{}},
		health: &healthItems{items: map[string]health.Item{}},
		now:    time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC),
	}
	deps := deletions.Deps{Pending: s.store, Health: s.health, Clock: fixedClock{now: s.now}}
	s.guard = deletions.NewGuardDeletions(deps)
	s.answer = deletions.NewResolveDeletions(deps)
	return s
}

func removals(count int) []transfer.Removal {
	files := make([]transfer.Removal, count)
	for i := range files {
		files[i] = transfer.Removal{FileID: []byte{byte(i + 1)}, Path: fmt.Sprintf("Papers/File %d.pdf", i+1), BaseHash: fmt.Sprintf("hash-%d", i+1)}
	}
	return files
}

func (s *setup) check(t *testing.T, direction transfer.Direction, count, tracked int) transfer.Verdict {
	t.Helper()
	verdict, err := s.guard.Execute(t.Context(), transfer.GuardInput{Vault: vaultID, Direction: direction, Candidates: removals(count), Tracked: tracked})
	if err != nil {
		t.Fatal(err)
	}
	return verdict
}

func TestGuardDeletions_PausesAtFiveFilesAndMoreThanThirtyPercent(t *testing.T) {
	t.Parallel()

	for _, one := range []struct {
		name           string
		count, tracked int
		want           transfer.Verdict
	}{
		{"4 of 10 is under the minimum", 4, 10, transfer.Go},
		{"5 of 10", 5, 10, transfer.Pause},
		{"5 of 16 is 31 percent", 5, 16, transfer.Pause},
		{"5 of 17 is 29 percent", 5, 17, transfer.Go},
		{"exactly 30 percent", 6, 20, transfer.Go},
		{"one file over 30 percent", 7, 20, transfer.Pause},
		{"3 of 3 is under the minimum", 3, 3, transfer.Go},
		{"5 of 5", 5, 5, transfer.Pause},
		{"tracked zero", 5, 0, transfer.Pause},
		{"none", 0, 100, transfer.Go},
		{"many of many", 600, 1000, transfer.Pause},
	} {
		s := newSetup()

		verdict := s.check(t, transfer.DirectionSend, one.count, one.tracked)

		if verdict != one.want {
			t.Errorf("%s: verdict = %d, want %d", one.name, verdict, one.want)
		}
		if _, isPending := s.store.states[vaultID]; isPending != (one.want == transfer.Pause) {
			t.Errorf("%s: pending = %v, want %v", one.name, isPending, one.want == transfer.Pause)
		}
	}
}

func TestGuardDeletions_KeepsTheFilesTheCountAndTheDirectionOfAPauseAndReportsThemInTheHealthItem(t *testing.T) {
	t.Parallel()
	s := newSetup()

	s.check(t, transfer.DirectionSend, 6, 10)

	state := s.store.states[vaultID]
	if state.Direction != transfer.DirectionSend || len(state.Files) != 6 || state.Tracked != 10 || !state.CreatedAt.Equal(s.now) {
		t.Errorf("state = %+v, want a send of 6 files of 10 tracked at the time of the clock", state)
	}
	item := s.health.items[deletions.GetHealthKey(vaultID)]
	if item.Severity != health.Error || item.Text != "Sync is paused: 6 of 10 files are removed" {
		t.Errorf("health item = %+v, want an error that reads %q", item, "Sync is paused: 6 of 10 files are removed")
	}
}

func TestGuardDeletions_ReportsThatFilesWereRemovedOnAnotherComputerForAnApply(t *testing.T) {
	t.Parallel()
	s := newSetup()

	verdict := s.check(t, transfer.DirectionApply, 6, 10)

	if verdict != transfer.Pause || s.store.states[vaultID].Direction != transfer.DirectionApply {
		t.Fatalf("verdict = %d, state = %+v, want a pause of an apply", verdict, s.store.states[vaultID])
	}
	if text := s.health.items[deletions.GetHealthKey(vaultID)].Text; text != "Sync is paused: 6 of 10 files were removed on another computer" {
		t.Errorf("text = %q, want the text of an apply", text)
	}
}

func TestGuardDeletions_KeepsPausingWithoutAskingAgainOrChangingThePendingList(t *testing.T) {
	t.Parallel()
	s := newSetup()
	s.check(t, transfer.DirectionSend, 6, 10)
	first := s.store.states[vaultID]

	verdict := s.check(t, transfer.DirectionSend, 2, 10)

	if verdict != transfer.Pause {
		t.Errorf("verdict = %d, want Pause while the answer is open even for fewer files", verdict)
	}
	if got := s.store.states[vaultID]; len(got.Files) != len(first.Files) || !got.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("state = %+v, want the first pending list kept", got)
	}
}

func TestGuardDeletions_PausesAVaultAndDoesNotPauseAnotherVault(t *testing.T) {
	t.Parallel()
	s := newSetup()
	s.check(t, transfer.DirectionSend, 6, 10)

	verdict, err := s.guard.Execute(t.Context(), transfer.GuardInput{Vault: "vault-2", Direction: transfer.DirectionSend, Candidates: removals(1), Tracked: 10})

	if err != nil || verdict != transfer.Go {
		t.Errorf("another vault: verdict = %d, error = %v, want Go", verdict, err)
	}
}

func TestResolveDeletions_ReturnsThePendingStateAndClearsItAndTheHealthItemForEitherChoice(t *testing.T) {
	t.Parallel()

	for _, choice := range []transfer.Choice{transfer.ChoiceRemove, transfer.ChoiceKeep} {
		s := newSetup()
		s.check(t, transfer.DirectionApply, 6, 10)

		state, err := s.answer.Execute(t.Context(), vaultID, choice)

		if err != nil || state.Direction != transfer.DirectionApply || len(state.Files) != 6 {
			t.Errorf("%s: state = %+v, error = %v, want the pending apply of 6 files", choice, state, err)
		}
		if _, isPending := s.store.states[vaultID]; isPending {
			t.Errorf("%s: the state is still pending", choice)
		}
		if _, isShown := s.health.items[deletions.GetHealthKey(vaultID)]; isShown {
			t.Errorf("%s: the health item is still shown", choice)
		}
	}
}

func TestResolveDeletions_RefusesAnAnswerWhereNothingIsPendingAndAChoiceItDoesNotKnow(t *testing.T) {
	t.Parallel()
	s := newSetup()

	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err == nil {
		t.Error("an answer with nothing pending gave no error")
	}
	s.check(t, transfer.DirectionSend, 6, 10)
	_, err := s.answer.Execute(t.Context(), vaultID, transfer.Choice("later"))
	if err == nil || !strings.Contains(err.Error(), "later") {
		t.Errorf("error = %v, want the unknown choice named", err)
	}
	if _, isPending := s.store.states[vaultID]; !isPending {
		t.Error("an unknown choice cleared the pending state")
	}
}
