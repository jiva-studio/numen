package hostchat

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

type fakeStream struct {
	frames []*llm.ChatFrame
	finish *llm.ChatFinish
	next   int
	err    error
	closed atomic.Bool
}

func (s *fakeStream) Next() bool {
	if s.next >= len(s.frames) {
		return false
	}
	s.next++
	return true
}

func (s *fakeStream) Frame() *llm.ChatFrame { return s.frames[s.next-1] }

func (s *fakeStream) Finish() *llm.ChatFinish { return s.finish }

func (s *fakeStream) Err() error {
	if s.next < len(s.frames) {
		return nil
	}
	return s.err
}

func (s *fakeStream) Close() error {
	s.closed.Store(true)
	return nil
}

func text(delta string) *llm.ChatFrame {
	return &llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: delta}}
}

type asked struct {
	request  *llm.ChatRequest
	priority connection.Priority
}

func ask(t *testing.T, stream *fakeStream, openErr error) (conversation.Reply, []string, asked, error) {
	t.Helper()
	var got asked
	chat := New(func(_ context.Context, request *llm.ChatRequest, priority connection.Priority) (Stream, error) {
		got = asked{request: request, priority: priority}
		if openErr != nil {
			return nil, openErr
		}
		return stream, nil
	})
	var spoken []string
	reply, err := chat.Stream(t.Context(), conversation.Turn{
		Messages: []conversation.Message{
			{Role: "user", Text: "hi"},
			{Role: "assistant", ToolCalls: []conversation.ToolCall{{ID: "c1", Name: "note_read", Arguments: "{}"}}},
			{Role: "tool", ToolCallID: "c1", Text: "found"},
		},
		Tools:       []conversation.Tool{{Name: "note_read", Description: "Reads", Parameters: []byte(`{"type":"object"}`)}},
		Temperature: new(0.5), MaxTokens: 64,
	}, func(text string) error {
		spoken = append(spoken, text)
		return nil
	})
	return reply, spoken, got, err
}

func TestATurnIsAskedForInteractivelyWithTheMessagesToolsAndOptions(t *testing.T) {
	_, _, got, err := ask(t, &fakeStream{finish: &llm.ChatFinish{}}, nil)

	if err != nil {
		t.Fatal(err)
	}
	if got.priority != connection.PriorityInteractive {
		t.Errorf("priority = %v, want interactive: a person waits for the answer", got.priority)
	}
	request := got.request
	if len(request.Messages) != 3 || request.Messages[1].ToolCalls[0].Id != "c1" || request.Messages[2].ToolCallId != "c1" {
		t.Errorf("messages = %+v", request.Messages)
	}
	if len(request.Tools) != 1 || request.Tools[0].Name != "note_read" || request.Temperature == nil || *request.Temperature != 0.5 || request.MaxTokens != 64 {
		t.Errorf("tools = %+v, temperature = %v, max tokens = %d", request.Tools, request.Temperature, request.MaxTokens)
	}
}

func TestTheTextAndTheCallsOfTheStreamMakeTheReply(t *testing.T) {
	stream := &fakeStream{frames: []*llm.ChatFrame{
		text("Let me "), text("look. "),
		{Event: &llm.ChatFrame_ToolCall{ToolCall: &llm.ToolCall{Id: "c2", Name: "note_read", ArgumentsJson: `{"path":"A.md"}`}}},
	}, finish: &llm.ChatFinish{Reason: llm.FinishReason_FINISH_REASON_TOOL_CALLS}}

	reply, spoken, _, err := ask(t, stream, nil)

	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(spoken, "|") != "Let me |look. " || reply.Message.Text != "Let me look. " || reply.Message.Role != "assistant" {
		t.Errorf("spoken %v, reply %+v", spoken, reply)
	}
	if len(reply.Message.ToolCalls) != 1 || reply.Message.ToolCalls[0].Arguments != `{"path":"A.md"}` || reply.IsCut {
		t.Errorf("reply = %+v", reply)
	}
}

func TestAnAnswerEndedByTheLengthLimitIsTruncated(t *testing.T) {
	stream := &fakeStream{frames: []*llm.ChatFrame{text("The plan")}, finish: &llm.ChatFinish{Reason: llm.FinishReason_FINISH_REASON_LENGTH}}

	reply, _, _, err := ask(t, stream, nil)

	if err != nil || !reply.IsCut {
		t.Errorf("reply = %+v, %v", reply, err)
	}
}

func TestEveryStreamOfTheModelIsClosed(t *testing.T) {
	stream := &fakeStream{frames: []*llm.ChatFrame{text("hi")}, finish: &llm.ChatFinish{Reason: llm.FinishReason_FINISH_REASON_STOP}}

	if _, _, _, err := ask(t, stream, nil); err != nil {
		t.Fatal(err)
	}

	if !stream.closed.Load() {
		t.Error("the stream of the model was left open")
	}
}

// The use case decides what a failure of the rows means, so the error of the core reaches it as it is.
func TestTheErrorOfTheCoreReachesTheCaller(t *testing.T) {
	failure := &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "Ferry API", Reason: "unavailable"}}}

	_, _, _, err := ask(t, nil, failure)

	if !errors.Is(err, failure) {
		t.Errorf("err = %v, want the failure of the rows", err)
	}
}

func TestAFailureInTheMiddleOfTheStreamKeepsTheText(t *testing.T) {
	stream := &fakeStream{frames: []*llm.ChatFrame{text("The plan is ")}, err: connection.ErrNoConnection}

	reply, spoken, _, err := ask(t, stream, nil)

	if !errors.Is(err, connection.ErrNoConnection) || strings.Join(spoken, "") != "The plan is " || reply.Message.Text != "The plan is " {
		t.Errorf("err = %v, spoken %v, reply %+v", err, spoken, reply)
	}
}
