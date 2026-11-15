package sessions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/sessions"
)

type stubLogStores struct{}

func (stubLogStores) Open(domain.Vault) (ports.LogStore, error) { return stubLogStore{}, nil }

type stubLogStore struct{}

func (stubLogStore) ListLogs(context.Context) ([]ports.Entry, error)             { return nil, nil }
func (stubLogStore) ReadLog(context.Context, string) ([]byte, error)             { return nil, nil }
func (stubLogStore) AppendLog(context.Context, string, []byte) error             { return nil }
func (stubLogStore) DeleteLogs(context.Context, []string) error                  { return nil }
func (stubLogStore) CompactLogs(context.Context, string, []byte, []string) error { return nil }

func TestGetRunReturnsErrNoRunForAnUnopenedRun(t *testing.T) {
	r := sessions.NewRegistry()
	if _, err := r.GetRun("v1", "r1"); !errors.Is(err, sessions.ErrNoRun) {
		t.Fatalf("expected ErrNoRun, got %v", err)
	}

	run, err := r.Open(t.Context(), flashcards.Log{Stores: stubLogStores{}}, domain.Vault{ID: "v1"}, time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := r.GetRun("v1", run.GetName())
	if err != nil || got != run {
		t.Fatalf("GetRun matching name = (%v, %v), want (%v, nil)", got, err, run)
	}
	if _, err := r.GetRun("v1", "different-run"); !errors.Is(err, sessions.ErrNoRun) {
		t.Fatalf("GetRun mismatched name = %v, want ErrNoRun", err)
	}
}
