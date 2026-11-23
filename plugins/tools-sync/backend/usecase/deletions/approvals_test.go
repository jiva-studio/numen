package deletions_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
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

func TestGuardDeletions_LetsTheRemovalsGoOnceAfterThePersonAnsweredRemove(t *testing.T) {
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

func TestGuardDeletions_UsesNoAnswerWhenThereAreNoRemovalsToAsk(t *testing.T) {
	t.Parallel()
	s, answers := newAnsweredSetup()
	answers.answers[vaultID] = transfer.Approval{Direction: transfer.DirectionSend, Choice: transfer.ChoiceKeep}

	verdict := s.check(t, transfer.DirectionSend, 0, 10)

	if verdict != transfer.Go {
		t.Errorf("verdict with no removals = %d, want Go", verdict)
	}
	if _, isKept := answers.answers[vaultID]; !isKept {
		t.Error("an answer was used by a pass with nothing to remove")
	}
}
