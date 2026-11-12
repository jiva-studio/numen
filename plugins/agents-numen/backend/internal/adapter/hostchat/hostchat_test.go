package hostchat

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

type fakeStream struct {
	events []connection.ChatEvent
	next   int
	err    error
	closed atomic.Bool
}

func (s *fakeStream) Next() (connection.ChatEvent, bool) {
	if s.next >= len(s.events) {
		return connection.ChatEvent{}, false
	}
	s.next++
	return s.events[s.next-1], true
}

func (s *fakeStream) Err() error {
	if s.next < len(s.events) {
		return nil
	}
	return s.err
}

func (s *fakeStream) Close() error {
	s.closed.Store(true)
	return nil
}

func ask(t *testing.T, stream *fakeStream, openErr error) (conversation.Reply, []string, connection.ChatRequest, error) {
	t.Helper()
	var request connection.ChatRequest
	chat := New(func(_ context.Context, asked connection.ChatRequest) (Stream, error) {
		request = asked
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
	return reply, spoken, request, err
}

func TestATurnIsAskedForInteractivelyWithTheMessagesToolsAndOptions(t *testing.T) {
	_, _, request, err := ask(t, &fakeStream{}, nil)

	if err != nil {
		t.Fatal(err)
	}
	if request.Priority != connection.PriorityInteractive {
		t.Errorf("priority = %v, want interactive: a person waits for the answer", request.Priority)
	}
	if len(request.Messages) != 3 || request.Messages[1].ToolCalls[0].ID != "c1" || request.Messages[2].ToolCallID != "c1" {
		t.Errorf("messages = %+v", request.Messages)
	}
	if len(request.Tools) != 1 || request.Tools[0].Name != "note_read" || request.Temperature == nil || *request.Temperature != 0.5 || request.MaxTokens != 64 {
		t.Errorf("tools = %+v, temperature = %v, max tokens = %d", request.Tools, request.Temperature, request.MaxTokens)
	}
}

func TestTheTextAndTheCallsOfTheStreamMakeTheReply(t *testing.T) {
	stream := &fakeStream{events: []connection.ChatEvent{
		{TextDelta: "Let me "}, {TextDelta: "look. "},
		{ToolCall: &connection.ToolCall{ID: "c2", Name: "note_read", Arguments: `{"path":"A.md"}`}},
		{Finish: &connection.Finish{Reason: connection.FinishToolCalls}},
	}}

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
	stream := &fakeStream{events: []connection.ChatEvent{{TextDelta: "The plan"}, {Finish: &connection.Finish{Reason: connection.FinishLength}}}}

	reply, _, _, err := ask(t, stream, nil)

	if err != nil || !reply.IsCut {
		t.Errorf("reply = %+v, %v", reply, err)
	}
}

func TestEveryStreamOfTheModelIsClosed(t *testing.T) {
	stream := &fakeStream{events: []connection.ChatEvent{{TextDelta: "hi"}}}

	if _, _, _, err := ask(t, stream, nil); err != nil {
		t.Fatal(err)
	}

	if !stream.closed.Load() {
		t.Error("the stream of the model was left open")
	}
}

func TestEveryRowFailingNamesEachConnectionAndReason(t *testing.T) {
	failure := &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unreachable"},
	}}

	_, _, _, err := ask(t, nil, failure)

	if err == nil || err.Error() != "OpenRouter: payment required; Ollama: unreachable" {
		t.Errorf("err = %v", err)
	}
}

func TestARejectionNamesTheConnectionAndTheReason(t *testing.T) {
	_, _, _, err := ask(t, nil, &connection.RejectedError{ConnectionName: "OpenRouter", Reason: "bad request"})

	if err == nil || err.Error() != "OpenRouter: bad request" {
		t.Errorf("err = %v", err)
	}
}

func TestACoreWithNoConnectionForTheAgentIsReportedSo(t *testing.T) {
	_, _, _, err := ask(t, nil, connection.ErrNoConnection)

	if !errors.Is(err, conversation.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
}

func TestAFailureInTheMiddleOfTheStreamKeepsTheTextAndNamesTheConnection(t *testing.T) {
	stream := &fakeStream{
		events: []connection.ChatEvent{{TextDelta: "The plan is "}},
		err:    &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "OpenRouter", Reason: "unreachable"}}},
	}

	reply, spoken, _, err := ask(t, stream, nil)

	if err == nil || err.Error() != "OpenRouter: unreachable" || strings.Join(spoken, "") != "The plan is " || reply.Message.Text != "The plan is " {
		t.Errorf("err = %v, spoken %v, reply %+v", err, spoken, reply)
	}
}
