package modelcall_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

type agentSetup struct {
	world *modelcalltest.World
	calls *modelcall.Calls
}

func newAgentSetup(t *testing.T) *agentSetup {
	t.Helper()
	w := modelcalltest.New()
	entry := w.AddAgent(t, "Claude Code")
	w.Rows.Set(consumerID, "complete", connection.Row{ConnectionID: entry.ID, Model: "sonnet", Reasoning: "high"})
	return &agentSetup{
		world: w,
		calls: modelcall.NewCalls(w.Resolve, w.ReadSecrets, w.Rows, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside),
	}
}

func (s *agentSetup) complete(ctx context.Context, text string) <-chan error {
	failed := make(chan error, 1)
	go func() {
		_, err := s.calls.Complete(ctx, consumerID, modelcalldomain.CompleteRequest{
			Messages: []modelcalldomain.Message{{Role: "user", Text: text}},
		}, modelcalldomain.PriorityBackground)
		failed <- err
	}()
	return failed
}

func TestARowOfAnAgentConnectionAnswersComplete(t *testing.T) {
	s := newAgentSetup(t)

	_, err := s.calls.Complete(t.Context(), consumerID, modelcalldomain.CompleteRequest{
		Messages: []modelcalldomain.Message{{Role: "user", Text: "fix this"}},
	}, modelcalldomain.PriorityBackground)

	if err != nil {
		t.Fatal(err)
	}
	calls := s.world.Provider.Calls()
	if len(calls) != 1 {
		t.Fatalf("the provider received %d calls, want 1", len(calls))
	}
	got := calls[0].Call
	if got.Type != "agent" || got.Model != "sonnet" || got.Reasoning != "high" {
		t.Errorf("call = %+v, want the type agent with the model and the level of the row", got)
	}
	if len(got.Secrets) != 0 || len(got.Fields) != 0 {
		t.Errorf("call = %+v, want no secret and no field: the program is the plugin", got)
	}
}

func TestAnAgentConnectionRunsOneCallAtATime(t *testing.T) {
	s := newAgentSetup(t)
	s.world.Provider.Block("")
	first := s.complete(t.Context(), "first")
	select {
	case <-s.world.Provider.Entered():
	case <-t.Context().Done():
	}

	second := s.complete(t.Context(), "second")
	s.world.Waits.AwaitCount(t, 1)

	if got := len(s.world.Provider.Calls()); got != 1 {
		t.Errorf("the provider received %d calls while the first ran, want 1", got)
	}
	s.world.Provider.Unblock("")
	if err := errors.Join(awaitDone(t, first), awaitDone(t, second)); err != nil {
		t.Fatal(err)
	}
}

func TestAnAgentConnectionDoesNotServeEmbed(t *testing.T) {
	s := newAgentSetup(t)
	entry := s.world.AddAgent(t, "Codex")
	s.world.Rows.Set(consumerID, "embed", modelcalltest.Row(entry, "m"))

	_, err := s.calls.Embed(t.Context(), consumerID, []string{"text"}, modelcalldomain.PriorityBackground)

	if err == nil || err.Error() != "Codex: this type cannot embed" {
		t.Errorf("Embed error = %v, want %q", err, "Codex: this type cannot embed")
	}
	if got := len(s.world.Provider.Calls()); got != 0 {
		t.Errorf("the provider received %d calls, want none", got)
	}
}
