package pull_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

type stuckRevisions struct {
	ports.Revisions
	calls int
}

func (s *stuckRevisions) GetChanges(ctx context.Context, vaultID []byte, since uint64, limit uint32) (*syncv1.GetChangesResponse, error) {
	s.calls++
	live, err := s.Revisions.GetChanges(ctx, vaultID, 0, 0)
	if err != nil {
		return nil, err
	}
	return &syncv1.GetChangesResponse{
		NextSeq: since, HeadSeq: since + 5, HasMore: true, RestoreEpochId: live.GetRestoreEpochId(),
	}, nil
}

func TestPullChanges_AServerThatSendsMoreAndNeverAdvancesTheCursorEndsThePull(t *testing.T) {
	p := newPair(t)
	deps := p.b.PullDeps()
	stuck := &stuckRevisions{Revisions: deps.Revisions}
	deps.Revisions = stuck

	_, err := pull.NewPullChanges(deps).Execute(t.Context(), ports.PullInput{VaultID: target, IsAuthor: trustAll})

	if !errors.Is(err, syncprotocol.ErrPageStuck) {
		t.Fatalf("error = %v, want ErrPageStuck", err)
	}
	if stuck.calls > 2 {
		t.Errorf("asked %d times", stuck.calls)
	}
}
