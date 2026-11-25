package deletions_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/healthkeytest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
)

type approvalStore struct {
	answers map[domain.VaultID]transfer.Approval
}

func (a *approvalStore) ReadApproval(_ context.Context, id domain.VaultID) (*transfer.Approval, error) {
	answer, ok := a.answers[id]
	if !ok {
		return nil, nil
	}
	return &answer, nil
}

func (a *approvalStore) WriteApproval(_ context.Context, id domain.VaultID, approval transfer.Approval) error {
	a.answers[id] = approval
	return nil
}

func (a *approvalStore) DeleteApproval(_ context.Context, id domain.VaultID) error {
	delete(a.answers, id)
	return nil
}

func newAnsweredSetup() (*setup, *approvalStore) {
	s := &setup{
		store:  &pendingStore{states: map[domain.VaultID]transfer.PausedRemovals{}},
		health: &healthItems{items: map[string]health.Item{}},
		now:    time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC),
	}
	answers := &approvalStore{answers: map[domain.VaultID]transfer.Approval{}}
	deps := deletions.Deps{Pending: s.store, Approvals: answers, Health: s.health, Clock: fixedClock{now: s.now}}
	s.guard = deletions.NewGuardDeletions(deps)
	s.answer = deletions.NewResolveDeletions(deps)
	return s, answers
}

func TestGuardDeletions_ReleasesTheRemovalsOnceAfterThePersonAnsweredRemove(t *testing.T) {
	t.Parallel()
	s, _ := newAnsweredSetup()
	s.check(t, transfer.DirectionSend, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	first := s.check(t, transfer.DirectionSend, 6, 10)
	second := s.check(t, transfer.DirectionSend, 6, 10)

	if first != transfer.Go {
		t.Errorf("verdict after the answer = %d, want Go", first)
	}
	if second != transfer.Pause {
		t.Errorf("verdict the next time = %d, want Pause: an answer is used once", second)
	}
}

func TestGuardDeletions_DropsTheRemovalsOnceAfterThePersonAnsweredKeep(t *testing.T) {
	t.Parallel()
	s, answers := newAnsweredSetup()
	s.check(t, transfer.DirectionApply, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceKeep); err != nil {
		t.Fatal(err)
	}

	verdict := s.check(t, transfer.DirectionApply, 6, 10)

	if verdict != transfer.Keep {
		t.Errorf("verdict after the answer = %d, want Keep", verdict)
	}
	if _, isKept := answers.answers[vaultID]; isKept {
		t.Error("the answer is still kept after the guard used it")
	}
}

func TestGuardDeletions_KeepsTheAnswerForTheDirectionItWasGivenFor(t *testing.T) {
	t.Parallel()
	s, answers := newAnsweredSetup()
	s.check(t, transfer.DirectionApply, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	verdict := s.check(t, transfer.DirectionSend, 6, 10)

	if verdict != transfer.Pause {
		t.Errorf("verdict of a send after the answer to an apply = %d, want Pause: the answer is not for a send", verdict)
	}
	if answer, isKept := answers.answers[vaultID]; !isKept || answer.Direction != transfer.DirectionApply {
		t.Errorf("answers = %+v, want the answer to the apply kept", answers.answers)
	}
}

func TestGuardDeletions_DropsTheAnswerWhenThereAreNoRemovalsToAsk(t *testing.T) {
	t.Parallel()
	s, answers := newAnsweredSetup()
	s.check(t, transfer.DirectionSend, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	verdict := s.check(t, transfer.DirectionSend, 0, 10)
	later := s.check(t, transfer.DirectionSend, 6, 10)

	if verdict != transfer.Go {
		t.Errorf("verdict with no removals = %d, want Go", verdict)
	}
	if _, isKept := answers.answers[vaultID]; isKept {
		t.Error("the answer is still kept after a pass with nothing to remove")
	}
	if later != transfer.Pause {
		t.Errorf("verdict of the same removals later = %d, want Pause: the answer is gone", later)
	}
}

func TestGuardDeletions_KeepsTheAnswerOfTheOtherDirectionWhenThereAreNoRemovals(t *testing.T) {
	t.Parallel()
	s, answers := newAnsweredSetup()
	s.check(t, transfer.DirectionApply, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	s.check(t, transfer.DirectionSend, 0, 10)

	if _, isKept := answers.answers[vaultID]; !isKept {
		t.Error("a send with nothing to remove dropped the answer to an apply")
	}
}

func TestGuardDeletions_DoesNotUseAnAnswerForMoreFilesThanThePersonWasShown(t *testing.T) {
	t.Parallel()
	s, answers := newAnsweredSetup()
	s.check(t, transfer.DirectionSend, 6, 20)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	verdict := s.check(t, transfer.DirectionSend, 12, 20)

	if verdict != transfer.Pause {
		t.Errorf("verdict for 12 files after the answer for 6 = %d, want Pause", verdict)
	}
	if _, isKept := answers.answers[vaultID]; isKept {
		t.Error("the answer is still kept after it was refused")
	}
	if got := len(s.store.states[vaultID].Files); got != 12 {
		t.Errorf("pending files = %d, want the 12 the person now has to answer for", got)
	}
}

func TestGuardDeletions_DoesNotUseAnAnswerForOtherFilesOfTheSameNumber(t *testing.T) {
	t.Parallel()
	s, _ := newAnsweredSetup()
	s.check(t, transfer.DirectionSend, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}
	others := removals(6)
	for i := range others {
		others[i].BaseHash += "-edited"
	}

	verdict, err := s.guard.Execute(t.Context(), transfer.GuardInput{Vault: vaultID, Direction: transfer.DirectionSend, Candidates: others, Tracked: 10})

	if err != nil || verdict != transfer.Pause {
		t.Errorf("verdict = %d, error = %v, want Pause: the files changed after they were shown", verdict, err)
	}
}

func TestGuardDeletions_UsesTheAnswerForFewerOfTheFilesThePersonWasShown(t *testing.T) {
	t.Parallel()
	s, _ := newAnsweredSetup()
	s.check(t, transfer.DirectionSend, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	verdict := s.check(t, transfer.DirectionSend, 5, 10)

	if verdict != transfer.Go {
		t.Errorf("verdict for 5 of the 6 files shown = %d, want Go", verdict)
	}
}

func TestGuardDeletions_DoesNotUseAnAnswerAfterItsLifetime(t *testing.T) {
	t.Parallel()
	s, answers := newAnsweredSetup()
	s.check(t, transfer.DirectionSend, 6, 10)
	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}
	later := deletions.NewGuardDeletions(deletions.Deps{
		Pending: s.store, Approvals: answers, Health: s.health, Clock: fixedClock{now: s.now.Add(deletions.ApprovalLifetime + time.Second)},
	})

	verdict, err := later.Execute(t.Context(), transfer.GuardInput{Vault: vaultID, Direction: transfer.DirectionSend, Candidates: removals(6), Tracked: 10})

	if err != nil || verdict != transfer.Pause {
		t.Errorf("verdict = %d, error = %v, want Pause for an answer older than its lifetime", verdict, err)
	}
}

func TestResolveDeletions_ClearsOnlyTheHealthItemOfItsVault(t *testing.T) {
	t.Parallel()
	s := newSetup()
	s.check(t, transfer.DirectionSend, 6, 10)
	if _, err := s.guard.Execute(t.Context(), transfer.GuardInput{Vault: "vault-2", Direction: transfer.DirectionSend, Candidates: removals(6), Tracked: 10}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.answer.Execute(t.Context(), vaultID, transfer.ChoiceKeep); err != nil {
		t.Fatal(err)
	}

	if _, isShown := s.health.items[deletions.GetHealthKey("vault-2")]; !isShown {
		t.Error("answering one vault cleared the health item of another vault that is still paused")
	}
	if _, isShown := s.health.items[deletions.GetHealthKey(vaultID)]; isShown {
		t.Error("the health item of the answered vault is still shown")
	}
	if key := deletions.GetHealthKey("01M7WEFFR86SKR8EG3SV5HF4XS"); !healthkeytest.IsValid(key) {
		t.Errorf("the key %q is not valid for the host", key)
	}
}
