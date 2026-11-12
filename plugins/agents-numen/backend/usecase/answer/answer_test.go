package answer_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
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
	if len(c.replies) > 1 {
		c.replies = c.replies[1:]
	}
	if err := emit(reply.Message.Text); err != nil {
		return conversation.Reply{}, err
	}
	return reply, nil
}

type toolCall struct {
	name, arguments string
	scope           conversation.Scope
}

// fakeTools are two tools of the host: one that reads a note and one that fails.
type fakeTools struct {
	called []toolCall
}

func (f *fakeTools) ListTools(context.Context) ([]conversation.Tool, error) {
	return []conversation.Tool{
		{Name: "note_read", Title: "Read a note", IsReadOnly: true, Instructions: []string{"Cite the note.", "Answer briefly."}},
		{Name: "note_write", Title: "Write a note", Instructions: []string{"Answer briefly.", "Ask before writing."}},
	}, nil
}

func (f *fakeTools) CallTool(_ context.Context, name, arguments string, scope conversation.Scope) conversation.ToolResult {
	f.called = append(f.called, toolCall{name: name, arguments: arguments, scope: scope})
	if name == "note_write" {
		return conversation.ToolResult{Text: "error: the note is locked", IsError: true}
	}
	return conversation.ToolResult{Text: "result of " + name}
}

func newAnswerer(t *testing.T, chat *scriptedChat, tools *fakeTools, rows string) (*answer.Answerer, *clock.Fake) {
	t.Helper()
	store := settings.NewStore()
	store.Configure(map[string]string{settings.KeyConnections: rows})
	now := clock.NewFake(time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC))
	answerer, err := answer.New(store, chat, tools, now)
	if err != nil {
		t.Fatal(err)
	}
	return answerer, now
}

const oneRow = `[{"connection":"c1","model":"m"}]`

func run(t *testing.T, answerer *answer.Answerer, task *acp.TaskRequest) ([]*acp.TaskFrame, *acp.TaskDone, error) {
	t.Helper()
	var frames []*acp.TaskFrame
	done, err := answerer.RunTask(t.Context(), connection.Call{}, task, func(frame *acp.TaskFrame) error {
		frames = append(frames, frame)
		return nil
	})
	return frames, done, err
}

func runTaskToEnd(t *testing.T, answerer *answer.Answerer, task *acp.TaskRequest) ([]*acp.TaskFrame, acp.StopReason) {
	t.Helper()
	frames, done, err := run(t, answerer, task)
	if err != nil {
		t.Fatal(err)
	}
	return frames, done.GetStopReason()
}

func callReply(id, name, arguments string) conversation.Reply {
	return conversation.Reply{Message: conversation.Message{Role: "assistant", ToolCalls: []conversation.ToolCall{{ID: id, Name: name, Arguments: arguments}}}}
}

func textReply(text string) conversation.Reply {
	return conversation.Reply{Message: conversation.Message{Role: "assistant", Text: text}}
}

func assertUnavailable(t *testing.T, err error) {
	t.Helper()
	if !connection.IsStop(err) || err.Error() != connection.ReasonUnavailable.String() {
		t.Errorf("err = %v, want a stop of reason unavailable", err)
	}
}

func TestATaskStopsAsUnavailableWhileTheSettingsNameNoConnection(t *testing.T) {
	chat := &scriptedChat{}
	answerer, _ := newAnswerer(t, chat, &fakeTools{}, `[]`)

	frames, _, err := run(t, answerer, &acp.TaskRequest{Prompt: "hi"})

	assertUnavailable(t, err)
	if len(frames) != 0 || len(chat.turns) != 0 {
		t.Errorf("frames = %v, turns = %d", frames, len(chat.turns))
	}
}

func TestAnAnswerWithoutToolCallsEndsTheTurnAndIsSentAsAMessageChunk(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{textReply("hello")}}
	answerer, _ := newAnswerer(t, chat, &fakeTools{}, oneRow)

	frames, reason := runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "hi"})

	if reason != acp.StopReason_STOP_REASON_END_TURN {
		t.Errorf("stop = %v, want end turn", reason)
	}
	if len(frames) != 1 || frames[0].GetAgentMessageChunk().GetText() != "hello" {
		t.Errorf("frames = %v", frames)
	}
}

func TestTheDistinctInstructionsOfTheToolsAreTheSystemText(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{textReply("hello")}}
	answerer, _ := newAnswerer(t, chat, &fakeTools{}, oneRow)

	runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "hi"})

	messages := chat.turns[0].Messages
	if len(messages) != 2 || messages[0].Role != "system" || messages[0].Text != "Cite the note. Answer briefly. Ask before writing." {
		t.Errorf("the messages are %+v", messages)
	}
	if names := []string{chat.turns[0].Tools[0].Name, chat.turns[0].Tools[1].Name}; !slices.Equal(names, []string{"note_read", "note_write"}) {
		t.Errorf("the model was offered %v", names)
	}
}

func TestTheNotesAttachedToThePromptGoWithIt(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{textReply("hello")}}
	answerer, _ := newAnswerer(t, chat, &fakeTools{}, oneRow)

	runTaskToEnd(t, answerer, &acp.TaskRequest{
		Prompt:       "When does the ferry leave?",
		ContextNotes: []*acp.ContextNote{{Path: "Trips/Ferry.md", Title: "Ferry", Text: "Leaves at 7:40."}},
	})

	prompt := chat.turns[0].Messages[1].Text
	if !strings.HasPrefix(prompt, "When does the ferry leave?") || !strings.Contains(prompt, "Trips/Ferry.md") || !strings.Contains(prompt, "Leaves at 7:40.") {
		t.Errorf("the prompt is %q", prompt)
	}
}

func TestAToolCallIsSentWithItsTitleKindAndNoteAndEndsCompleted(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{callReply("1", "note_read", `{"path":"Trips/Ferry.md"}`), textReply("done reading")}}
	tools := &fakeTools{}
	answerer, _ := newAnswerer(t, chat, tools, oneRow)

	frames, reason := runTaskToEnd(t, answerer, &acp.TaskRequest{
		Prompt: "read it", ToolContext: map[string]string{"md.example.cards/card": "k7"}, VaultId: "v-trips",
	})

	if len(frames) != 3 || reason != acp.StopReason_STOP_REASON_END_TURN {
		t.Fatalf("frames = %v, stop = %v", frames, reason)
	}
	call := frames[0].GetToolCall()
	if call.GetToolCallId() != "1" || call.GetTitle() != "Read a note" || call.GetKind() != acp.ToolKind_TOOL_KIND_READ ||
		call.GetContent()[0] != "Trips/Ferry.md" || call.GetLocations()[0].GetPath() != "Trips/Ferry.md" {
		t.Errorf("the call is %v", call)
	}
	if update := frames[1].GetToolCallUpdate(); update.GetToolCallId() != "1" || update.GetStatus() != acp.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED {
		t.Errorf("the update is %v", update)
	}
	if len(tools.called) != 1 || tools.called[0].scope.VaultID != "v-trips" || tools.called[0].scope.Context["md.example.cards/card"] != "k7" {
		t.Errorf("the tools were called with %+v", tools.called)
	}
	sent := chat.turns[1].Messages
	if last := sent[len(sent)-1]; last.Role != "tool" || last.ToolCallID != "1" || last.Text != "result of note_read" {
		t.Errorf("the tool result sent back = %+v", last)
	}
}

func TestAToolThatChangesTheVaultIsAnEditAndAFailedCallEndsFailed(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{callReply("1", "note_write", `{"path":"Trips/Ferry.md"}`), textReply("could not")}}
	answerer, _ := newAnswerer(t, chat, &fakeTools{}, oneRow)

	frames, _ := runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "write it"})

	if kind := frames[0].GetToolCall().GetKind(); kind != acp.ToolKind_TOOL_KIND_EDIT {
		t.Errorf("the kind is %v", kind)
	}
	if status := frames[1].GetToolCallUpdate().GetStatus(); status != acp.ToolCallStatus_TOOL_CALL_STATUS_FAILED {
		t.Errorf("the status is %v", status)
	}
	if sent := chat.turns[1].Messages; sent[len(sent)-1].Text != "error: the note is locked" {
		t.Errorf("the model read %q", sent[len(sent)-1].Text)
	}
}

func TestACallOfAToolTheHostDoesNotListFailsWithoutACall(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{callReply("1", "note_delete", " "), textReply("ok")}}
	tools := &fakeTools{}
	answerer, _ := newAnswerer(t, chat, tools, oneRow)

	frames, _ := runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "delete it"})

	if call := frames[0].GetToolCall(); call.GetTitle() != "note_delete" || call.GetKind() != acp.ToolKind_TOOL_KIND_OTHER || call.GetRawInputJson() != "{}" {
		t.Errorf("the call is %v", call)
	}
	if status := frames[1].GetToolCallUpdate().GetStatus(); status != acp.ToolCallStatus_TOOL_CALL_STATUS_FAILED || len(tools.called) != 0 {
		t.Errorf("the status is %v after the calls %v", status, tools.called)
	}
}

func TestARunThatKeepsCallingToolsStopsAtTheLimitOfSteps(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{callReply("1", "note_read", "{}")}}
	store := settings.NewStore()
	store.Configure(map[string]string{settings.KeyConnections: oneRow, settings.KeyMaxSteps: "2"})
	limited, err := answer.New(store, chat, &fakeTools{}, clock.System{})
	if err != nil {
		t.Fatal(err)
	}

	_, reason := runTaskToEnd(t, limited, &acp.TaskRequest{Prompt: "loop"})

	if reason != acp.StopReason_STOP_REASON_MAX_TURN_REQUESTS || len(chat.turns) != 2 {
		t.Errorf("stop = %v after %d turns", reason, len(chat.turns))
	}
}

func TestAnAnswerEndedByTheLengthLimitStopsAtMaxTokens(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{{Message: conversation.Message{Role: "assistant", Text: "long"}, IsCut: true}}}
	answerer, _ := newAnswerer(t, chat, &fakeTools{}, oneRow)

	if _, reason := runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "hi"}); reason != acp.StopReason_STOP_REASON_MAX_TOKENS {
		t.Errorf("stop = %v, want max tokens", reason)
	}
}

func TestAChatThatFailsStopsTheTaskAsUnavailableWithoutItsText(t *testing.T) {
	for _, failure := range []error{
		connection.ErrNoConnection,
		&connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "Ferry API", Reason: "rate limited"}}},
	} {
		chat := &scriptedChat{err: failure}
		answerer, _ := newAnswerer(t, chat, &fakeTools{}, oneRow)

		_, _, err := run(t, answerer, &acp.TaskRequest{Prompt: "hi"})

		assertUnavailable(t, err)
		if strings.Contains(err.Error(), "Ferry API") || strings.Contains(err.Error(), "connection:") || len(chat.turns) != 1 {
			t.Errorf("err = %v after %d turns", err, len(chat.turns))
		}
	}
}

func TestAConversationContinuesFromItsHistory(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{textReply("one")}}
	answerer, _ := newAnswerer(t, chat, &fakeTools{}, oneRow)
	task := &acp.TaskRequest{Prompt: "q", Conversation: "thread"}

	runTaskToEnd(t, answerer, task)
	runTaskToEnd(t, answerer, task)
	runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "q"})

	if got := len(chat.turns[1].Messages); got != 4 {
		t.Errorf("the second turn holds %d messages, want 4 (system, question, answer, question)", got)
	}
	if got := len(chat.turns[2].Messages); got != 2 {
		t.Errorf("a task of no conversation holds %d messages, want 2", got)
	}
}

func TestAConversationIdleForThirtyMinutesIsForgotten(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{textReply("one")}}
	answerer, now := newAnswerer(t, chat, &fakeTools{}, oneRow)
	task := &acp.TaskRequest{Prompt: "q", Conversation: "thread"}

	runTaskToEnd(t, answerer, task)
	now.Advance(29 * time.Minute)
	runTaskToEnd(t, answerer, task)
	now.Advance(30 * time.Minute)
	runTaskToEnd(t, answerer, task)

	if kept, forgotten := len(chat.turns[1].Messages), len(chat.turns[2].Messages); kept != 4 || forgotten != 2 {
		t.Errorf("the turns hold %d and %d messages, want 4 and 2", kept, forgotten)
	}
}

func TestANinthConversationForgetsTheOneUsedLongestAgo(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{textReply("one")}}
	answerer, now := newAnswerer(t, chat, &fakeTools{}, oneRow)
	for _, id := range []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9"} {
		runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "q", Conversation: id})
		now.Advance(time.Minute)
	}

	runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "q", Conversation: "t2"})
	runTaskToEnd(t, answerer, &acp.TaskRequest{Prompt: "q", Conversation: "t1"})

	turns := chat.turns
	if kept, forgotten := len(turns[len(turns)-2].Messages), len(turns[len(turns)-1].Messages); kept != 4 || forgotten != 2 {
		t.Errorf("t2 holds %d messages and t1 %d, want 4 and 2", kept, forgotten)
	}
}

func TestAFrameThatCannotBeSentEndsTheTaskWithItsError(t *testing.T) {
	chat := &scriptedChat{replies: []conversation.Reply{callReply("1", "note_read", "{}"), textReply("ok")}}
	tools := &fakeTools{}
	answerer, _ := newAnswerer(t, chat, tools, oneRow)
	closed := errors.New("the window closed")

	_, err := answerer.RunTask(t.Context(), connection.Call{}, &acp.TaskRequest{Prompt: "read"}, func(*acp.TaskFrame) error { return closed })

	if !errors.Is(err, closed) || len(tools.called) != 0 {
		t.Errorf("err = %v after the calls %v", err, tools.called)
	}
}

func TestAnAnswererNeedsItsParts(t *testing.T) {
	if _, err := answer.New(nil, &scriptedChat{}, &fakeTools{}, clock.System{}); err == nil {
		t.Error("an answerer was made with no settings")
	}
}

func TestAClosedAnswererRejectsNewTasks(t *testing.T) {
	answerer, _ := newAnswerer(t, &scriptedChat{replies: []conversation.Reply{textReply("hi")}}, &fakeTools{}, oneRow)
	if err := answerer.Close(); err != nil {
		t.Fatal(err)
	}

	_, _, err := run(t, answerer, &acp.TaskRequest{Prompt: "hi"})

	assertUnavailable(t, err)
}
