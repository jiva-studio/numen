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
	w.Rows.Set(consumerID, modelcalltest.CompleteOperation, connection.Row{ConnectionID: entry.ID, Options: map[string]string{"model": "sonnet", "reasoning": "high"}})
	return &agentSetup{
		world: w,
		calls: modelcall.NewCalls(modelcall.Connections{Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows.ListRows}, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside),
	}
}

func (s *agentSetup) complete(ctx context.Context, text string) <-chan error {
	failed := make(chan error, 1)
	go func() {
		_, err := s.calls.Call(ctx, consumerID, modelcalldomain.Request{Operation: modelcalltest.CompleteOperation, Payload: []byte(text)},
			func([]byte) error { return nil })
		failed <- err
	}()
	return failed
}

func TestARowOfAnAgentConnectionAnswersComplete(t *testing.T) {
	s := newAgentSetup(t)

	_, err := s.calls.Call(t.Context(), consumerID, modelcalldomain.Request{Operation: modelcalltest.CompleteOperation, Payload: []byte("fix this")},
		func([]byte) error { return nil })

	if err != nil {
		t.Fatal(err)
	}
	calls := s.world.Provider.Calls()
	if len(calls) != 1 {
		t.Fatalf("the provider received %d calls, want 1", len(calls))
	}
	got := calls[0].Call
	if got.Type != "agent" || got.Options["model"] != "sonnet" || got.Options["reasoning"] != "high" {
		t.Errorf("call = %+v, want the type agent with the options of the row", got)
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
	s.world.Rows.Set(consumerID, modelcalltest.EmbedOperation, modelcalltest.Row(entry, "m"))

	_, err := s.calls.Call(t.Context(), consumerID, modelcalldomain.Request{Operation: modelcalltest.EmbedOperation}, func([]byte) error { return nil })

	if err == nil || err.Error() != "Codex: the type does not serve llm.embed" {
		t.Errorf("error = %v, want %q", err, "Codex: the type does not serve llm.embed")
	}
	if got := len(s.world.Provider.Calls()); got != 0 {
		t.Errorf("the provider received %d calls, want none", got)
	}
}
