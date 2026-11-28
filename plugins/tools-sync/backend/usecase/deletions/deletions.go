// Package deletions pauses the sync before it sends or applies the removal of many files, and carries out the
// answer of the person.
package deletions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ApprovalLifetime is how long an answer to a pause stays valid for the next pass.
const ApprovalLifetime = 24 * time.Hour

const healthKeyPrefix = "deletions-paused"

// GetHealthKey returns the key of the health item of the removals of one vault that wait for an answer.
func GetHealthKey(id domain.VaultID) string { return healthkey.Create(healthKeyPrefix, string(id)) }

// ErrNothingPending is what an answer gets for a vault whose removals are not paused.
var ErrNothingPending = errors.New("no removals are paused")

// Clock reads the time.
type Clock interface {
	Now() time.Time
}

// Deps are what the use cases need. Approvals may be nil, and then an answer is not kept for the next pass.
type Deps struct {
	Pending   ports.PausedStore
	Approvals ports.Approvals
	Health    ports.Health
	Clock     Clock
}

// GuardDeletions decides whether a pass may remove files.
type GuardDeletions struct{ deps Deps }

// NewGuardDeletions returns the use case over the dependencies.
func NewGuardDeletions(deps Deps) *GuardDeletions { return &GuardDeletions{deps: deps} }

// Execute returns Pause, and keeps the state until it is answered, when the candidates are at least
// note.MassDeleteMinFiles and more than note.MassDeleteRatio percent of the tracked files. It returns Keep or Go
// once for the answer of the person to an earlier pause of the same direction, while every candidate is a file the
// person was shown and the answer is not older than ApprovalLifetime. A pass that cannot use the answer this way
// drops it.
func (u *GuardDeletions) Execute(ctx context.Context, in transfer.GuardInput) (transfer.Verdict, error) {
	pending, err := u.deps.Pending.ReadPending(ctx, in.Vault)
	if err != nil {
		return 0, fmt.Errorf("read the pending removals: %w", err)
	}
	if pending != nil {
		return transfer.Pause, nil
	}
	if len(in.Candidates) == 0 {
		return transfer.Go, u.dropApproval(ctx, in)
	}
	if verdict, isAnswered, err := u.takeAnswer(ctx, in); isAnswered || err != nil {
		return verdict, err
	}
	count := len(in.Candidates)
	if count < note.MassDeleteMinFiles || count*100 <= note.MassDeleteRatio*in.Tracked {
		return transfer.Go, nil
	}
	state := transfer.PausedRemovals{Direction: in.Direction, Files: in.Candidates, Tracked: in.Tracked, CreatedAt: u.deps.Clock.Now()}
	if err := u.deps.Pending.WritePending(ctx, in.Vault, state); err != nil {
		return 0, fmt.Errorf("keep the pending removals: %w", err)
	}
	item := health.Item{Key: GetHealthKey(in.Vault), Severity: health.Error, Text: getPauseText(state), Detail: "", Action: nil}
	if err := u.deps.Health.Report(ctx, item); err != nil {
		return 0, fmt.Errorf("report the paused removals: %w", err)
	}
	return transfer.Pause, nil
}

func getPauseText(state transfer.PausedRemovals) string {
	if state.Direction == transfer.DirectionApply {
		return fmt.Sprintf("Sync is paused: %d of %d files were removed on another computer", len(state.Files), state.Tracked)
	}
	return fmt.Sprintf("Sync is paused: %d of %d files are removed", len(state.Files), state.Tracked)
}

func (u *GuardDeletions) takeAnswer(ctx context.Context, in transfer.GuardInput) (transfer.Verdict, bool, error) {
	if u.deps.Approvals == nil {
		return 0, false, nil
	}
	approval, err := u.deps.Approvals.ReadApproval(ctx, in.Vault)
	if err != nil {
		return 0, false, fmt.Errorf("read the answer to the paused removals: %w", err)
	}
	if approval == nil || approval.Direction != in.Direction {
		return 0, false, nil
	}
	if err := u.deps.Approvals.DeleteApproval(ctx, in.Vault); err != nil {
		return 0, false, fmt.Errorf("clear the answer to the paused removals: %w", err)
	}
	if !u.isCovered(*approval, in.Candidates) {
		return 0, false, nil
	}
	if approval.Choice == transfer.ChoiceKeep {
		return transfer.Keep, true, nil
	}
	return transfer.Go, true, nil
}

func (u *GuardDeletions) dropApproval(ctx context.Context, in transfer.GuardInput) error {
	if u.deps.Approvals == nil {
		return nil
	}
	approval, err := u.deps.Approvals.ReadApproval(ctx, in.Vault)
	if err != nil {
		return fmt.Errorf("read the answer to the paused removals: %w", err)
	}
	if approval == nil || approval.Direction != in.Direction {
		return nil
	}
	if err := u.deps.Approvals.DeleteApproval(ctx, in.Vault); err != nil {
		return fmt.Errorf("clear the answer to the paused removals: %w", err)
	}
	return nil
}

func (u *GuardDeletions) isCovered(approval transfer.Approval, candidates []transfer.Removal) bool {
	if u.deps.Clock.Now().Sub(approval.GivenAt) > ApprovalLifetime {
		return false
	}
	return !slices.ContainsFunc(candidates, func(candidate transfer.Removal) bool {
		return !slices.ContainsFunc(approval.Files, func(shown transfer.Removal) bool { return isSameRemoval(shown, candidate) })
	})
}

func isSameRemoval(a, b transfer.Removal) bool {
	return bytes.Equal(a.FileID, b.FileID) && a.Path == b.Path && a.BaseHash == b.BaseHash && bytes.Equal(a.RevID, b.RevID)
}

// ResolveDeletions carries out the answer of the person for one vault.
type ResolveDeletions struct{ deps Deps }

// NewResolveDeletions returns the use case over the dependencies.
func NewResolveDeletions(deps Deps) *ResolveDeletions { return &ResolveDeletions{deps: deps} }

// Execute removes the pending state and the health item and returns the state the answer was for.
func (u *ResolveDeletions) Execute(ctx context.Context, id domain.VaultID, choice transfer.Choice) (transfer.PausedRemovals, error) {
	if choice != transfer.ChoiceRemove && choice != transfer.ChoiceKeep {
		return transfer.PausedRemovals{}, fmt.Errorf("the choice %q is not known", choice)
	}
	pending, err := u.deps.Pending.ReadPending(ctx, id)
	if err != nil {
		return transfer.PausedRemovals{}, fmt.Errorf("read the pending removals: %w", err)
	}
	if pending == nil {
		return transfer.PausedRemovals{}, ErrNothingPending
	}
	if err := u.deps.Pending.DeletePending(ctx, id); err != nil {
		return transfer.PausedRemovals{}, fmt.Errorf("clear the pending removals: %w", err)
	}
	if err := u.deps.Health.Clear(ctx, GetHealthKey(id)); err != nil {
		return transfer.PausedRemovals{}, fmt.Errorf("clear the health item: %w", err)
	}
	if u.deps.Approvals != nil {
		if err := u.deps.Approvals.WriteApproval(ctx, id, transfer.Approval{Direction: pending.Direction, Choice: choice, Files: pending.Files, GivenAt: u.deps.Clock.Now()}); err != nil {
			return transfer.PausedRemovals{}, fmt.Errorf("keep the answer to the paused removals: %w", err)
		}
	}
	return *pending, nil
}
