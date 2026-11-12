package answer_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/answer"
)

type scriptedChat struct {
	replies []conversation.Reply
	err     error
	turns   []conversation.Turn
}

func (c *scriptedChat) Stream(_ context.Context, turn conversation.Turn, emit func(string) error) (conversation.Reply, error) {
	c.turns = append(c.turns, turn)
	if c.err != nil {
		return conversation.Reply{}, c.err
	}
	reply := c.replies[0]
	c.replies = c.replies[1:]
	if err := emit(reply.Message.Text); err != nil {
		return conversation.Reply{}, err
	}
	return reply, nil
}

type fakeToolbox struct{ called []string }

func (f *fakeToolbox) Tools() []conversation.Tool { return nil }

func (f *fakeToolbox) Call(_ context.Context, name, _ string) string {
	f.called = append(f.called, name)
	return "result of " + name
}

func (f *fakeToolbox) Close() error { return nil }

type fakeOpener struct {
	toolbox *fakeToolbox
	err     error
}

func (o *fakeOpener) OpenTools(context.Context, conversation.ToolServer) (ports.Toolbox, error) {
	return o.toolbox, o.err
}

func newAnswerer(t *testing.T, chat *scriptedChat, opener *fakeOpener, rows string) *answer.Answerer {
	t.Helper()
	store := settings.NewStore()
	store.Configure(map[string]string{settings.KeyConnections: rows})
	answerer, err := answer.New(store, chat, opener)
	if err != nil {
		t.Fatal(err)
	}
	return answerer
}

const oneRow = `[{"connection":"c1","model":"m"}]`

func collect(t *testing.T, answerer *answer.Answerer, task *agentrun.Request) []agentrun.Step {
	t.Helper()
	var steps []agentrun.Step
	err := answerer.Ask(t.Context(), task, func(response *agentrun.Response) error {
		steps = append(steps, response.Step)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return steps
}

func lastStop(t *testing.T, steps []agentrun.Step) string {
	t.Helper()
	stopped, ok := steps[len(steps)-1].(*agentrun.StoppedStep)
	if !ok {
		t.Fatalf("last step is %T, want a stop", steps[len(steps)-1])
	}
	return stopped.Stopped
}

func TestATaskStopsAtOnceWhileTheSettingsNameNoConnection(t *testing.T) {
	chat := &scriptedChat{}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: &fakeToolbox{}}, `[]`)

	steps := collect(t, answerer, &agentrun.Request{Prompt: "hi"})

	if len(steps) != 1 || lastStop(t, steps) != conversation.ReasonNoConnection || len(chat.turns) != 0 {
		t.Errorf("steps = %v, turns = %d", steps, len(chat.turns))
	}
}

func TestAnAnswerWithoutToolCallsEndsTheTask(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{{Message: conversation.Message{Role: "assistant", Text: "hello"}}}}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: &fakeToolbox{}}, oneRow)

	steps := collect(t, answerer, &agentrun.Request{Prompt: "hi", Instructions: "be brief"})

	if lastStop(t, steps) != answer.ReasonDone {
		t.Errorf("stop = %q, want done", lastStop(t, steps))
	}
	if !slices.ContainsFunc(steps, func(s agentrun.Step) bool { text, ok := s.(*agentrun.TextStep); return ok && text.Text == "hello" }) {
		t.Errorf("the text of the answer was not reported: %v", steps)
	}
	roles := []string{}
	for _, message := range chat.turns[0].Messages {
		roles = append(roles, message.Role)
	}
	if !slices.Equal(roles, []string{"system", "user"}) {
		t.Errorf("roles sent = %v, want system and user", roles)
	}
}

func TestAToolTheModelCallsRunsWithoutApprovalAndItsResultGoesBackToTheModel(t *testing.T) {
	call := conversation.ToolCall{ID: "1", Name: "read_note", Arguments: " "}
	chat := &scriptedChat{replies: []conversation.Reply{
		{Message: conversation.Message{Role: "assistant", ToolCalls: []conversation.ToolCall{call}}},
		{Message: conversation.Message{Role: "assistant", Text: "done reading"}},
	}}
	toolbox := &fakeToolbox{}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: toolbox}, oneRow)

	steps := collect(t, answerer, &agentrun.Request{Prompt: "read it"})

	if !slices.Equal(toolbox.called, []string{"read_note"}) || lastStop(t, steps) != answer.ReasonDone {
		t.Fatalf("tools called = %v, stop = %q", toolbox.called, lastStop(t, steps))
	}
	sent := chat.turns[1].Messages
	if last := sent[len(sent)-1]; last.Role != "tool" || last.ToolCallID != "1" || last.Text != "result of read_note" {
		t.Errorf("the tool result sent back = %+v", last)
	}
	if sent[len(sent)-2].ToolCalls[0].Arguments != "{}" {
		t.Errorf("empty arguments were sent as %q, want {}", sent[len(sent)-2].ToolCalls[0].Arguments)
	}
}

func TestARunThatKeepsCallingToolsStopsAtTheLimitOfSteps(t *testing.T) {
	call := conversation.ToolCall{ID: "1", Name: "read_note", Arguments: "{}"}
	reply := conversation.Reply{Message: conversation.Message{Role: "assistant", ToolCalls: []conversation.ToolCall{call}}}
	chat := &scriptedChat{replies: []conversation.Reply{reply, reply}}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: &fakeToolbox{}}, oneRow)

	steps := collect(t, answerer, &agentrun.Request{Prompt: "loop", MaxSteps: 2})

	if lastStop(t, steps) != answer.ReasonTurns || len(chat.turns) != 2 {
		t.Errorf("stop = %q after %d turns", lastStop(t, steps), len(chat.turns))
	}
}

func TestAnAnswerEndedByTheLengthLimitIsReported(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{{Message: conversation.Message{Role: "assistant", Text: "long"}, IsCut: true}}}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: &fakeToolbox{}}, oneRow)

	if stop := lastStop(t, collect(t, answerer, &agentrun.Request{Prompt: "hi"})); stop != answer.ReasonCut {
		t.Errorf("stop = %q, want the cut reason", stop)
	}
}

func TestAFailedTurnStopsTheTaskWithItsError(t *testing.T) {
	chat := &scriptedChat{err: errors.New("the service is down")}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: &fakeToolbox{}}, oneRow)

	if stop := lastStop(t, collect(t, answerer, &agentrun.Request{Prompt: "hi"})); stop != "the service is down" || len(chat.turns) != 1 {
		t.Errorf("stop = %q after %d turns, want the error once", stop, len(chat.turns))
	}
}

func TestATurnWithoutAConnectionStopsAsHavingNone(t *testing.T) {
	chat := &scriptedChat{err: conversation.ErrNoConnection}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: &fakeToolbox{}}, oneRow)

	if stop := lastStop(t, collect(t, answerer, &agentrun.Request{Prompt: "hi"})); stop != conversation.ReasonNoConnection {
		t.Errorf("stop = %q, want no connection", stop)
	}
}

func TestATaskThatCannotReachTheToolsStopsWithAReason(t *testing.T) {
	answerer := newAnswerer(t, &scriptedChat{}, &fakeOpener{err: errors.New("refused")}, oneRow)

	stop := lastStop(t, collect(t, answerer, &agentrun.Request{Prompt: "hi"}))

	if stop != "the agent could not reach this vault: refused" {
		t.Errorf("stop = %q", stop)
	}
}

func TestAConversationContinuesFromItsHistoryUntilItIsFinished(t *testing.T) {
	first := conversation.Reply{Message: conversation.Message{Role: "assistant", Text: "one"}}
	chat := &scriptedChat{replies: []conversation.Reply{first, first, first}}
	answerer := newAnswerer(t, chat, &fakeOpener{toolbox: &fakeToolbox{}}, oneRow)
	task := &agentrun.Request{Prompt: "q", Conversation: "thread"}

	collect(t, answerer, task)
	collect(t, answerer, task)
	if got := len(chat.turns[1].Messages); got != 3 {
		t.Errorf("second turn holds %d messages, want 3 (question, answer, question)", got)
	}
	if err := answerer.FinishConversation(t.Context(), "thread"); err != nil {
		t.Fatal(err)
	}
	collect(t, answerer, task)
	if got := len(chat.turns[2].Messages); got != 1 {
		t.Errorf("after the conversation finished the turn holds %d messages, want 1", got)
	}
}

func TestAnAnswererNeedsItsParts(t *testing.T) {
	if _, err := answer.New(nil, &scriptedChat{}, &fakeOpener{}); err == nil {
		t.Error("an answerer was made with no settings")
	}
}

func TestAClosedAnswererRejectsNewTasks(t *testing.T) {
	answerer := newAnswerer(t, &scriptedChat{}, &fakeOpener{toolbox: &fakeToolbox{}}, oneRow)
	if err := answerer.Close(); err != nil {
		t.Fatal(err)
	}

	err := answerer.Ask(t.Context(), &agentrun.Request{Prompt: "hi"}, func(*agentrun.Response) error { return nil })

	if err == nil {
		t.Error("a task was accepted after Close")
	}
}
