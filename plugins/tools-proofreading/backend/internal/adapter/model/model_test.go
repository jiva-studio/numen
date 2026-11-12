package model

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

func TestAskSendsTheInstructionAndThePromptAtTheInteractivePriority(t *testing.T) {
	var asked []*llm.CompleteRequest
	var priorities []connection.Priority
	complete := func(_ context.Context, request *llm.CompleteRequest, priority connection.Priority) (*llm.CompleteResult, error) {
		asked = append(asked, request)
		priorities = append(priorities, priority)
		return &llm.CompleteResult{Text: "the answer"}, nil
	}

	answer, err := NewAsk(complete)(t.Context(), "be careful", "the prompt")
	if err != nil {
		t.Fatal(err)
	}

	request := asked[0]
	if answer != "the answer" || len(request.Messages) != 2 ||
		request.Messages[0].Role != "system" || request.Messages[0].Text != "be careful" ||
		request.Messages[1].Role != "user" || request.Messages[1].Text != "the prompt" ||
		priorities[0] != connection.PriorityInteractive {
		t.Errorf("answer %q to request %+v", answer, request)
	}
}

func failWith(err error) Completer {
	return func(context.Context, *llm.CompleteRequest, connection.Priority) (*llm.CompleteResult, error) {
		return nil, err
	}
}

func TestAskWithNoConnectionIsOffAndPointsToTheConnectionsPage(t *testing.T) {
	_, err := NewAsk(failWith(connection.ErrNoConnection))(t.Context(), "be careful", "the prompt")

	if !errors.Is(err, ErrNoConnection) {
		t.Fatalf("got %v, want %v", err, ErrNoConnection)
	}
}

func TestEveryConnectionThatFailedIsListedInTheError(t *testing.T) {
	failed := &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "rate limited"},
		{ConnectionName: "Local", Reason: "unreachable"},
	}}

	_, err := NewAsk(failWith(failed))(t.Context(), "be careful", "the prompt")

	all, isAllFailed := errors.AsType[*connection.AllFailedError](err)
	if !isAllFailed || len(all.Failures) != 2 {
		t.Fatalf("got %v, want every connection failed", err)
	}
}

func TestARejectionKeepsTheConnectionThatFoundTheMistake(t *testing.T) {
	_, err := NewAsk(failWith(&connection.StoppedError{ConnectionName: "OpenRouter", Reason: "bad request"}))(t.Context(), "s", "p")

	var rejected *connection.StoppedError
	if !errors.As(err, &rejected) || rejected.ConnectionName != "OpenRouter" {
		t.Fatalf("got %v, want a rejection by OpenRouter", err)
	}
}

func TestTheMockAnswersTheSameQuestionTheSameWay(t *testing.T) {
	request := &llm.CompleteRequest{Messages: []*llm.Message{
		{Role: "system", Text: numbered.SpeechInstruction},
		{Role: "user", Text: "What it holds\n\n" + numbered.BuildPrompt(numbered.Batch{Lines: []numbered.Line{{Number: 4, Text: "one"}, {Number: 5, Text: "two"}}})},
	}}

	first, err := CompleteMock(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompleteMock(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}

	if want := "4|one " + MockMark + "\n5|two " + MockMark; first.GetText() != want || second.GetText() != want {
		t.Errorf("the mock answered %q and %q, want %q", first.GetText(), second.GetText(), want)
	}
}

func TestTheMockAnswersOnlyWhereNoConnectionIsSet(t *testing.T) {
	prompt := "What it holds\n\n" + numbered.BuildPrompt(numbered.Batch{Lines: []numbered.Line{{Number: 1, Text: "one"}}})
	request := &llm.CompleteRequest{Messages: []*llm.Message{{Role: "system", Text: "s"}, {Role: "user", Text: prompt}}}

	noRows, err := WithMock(failWith(connection.ErrNoConnection))(t.Context(), request, connection.PriorityInteractive)
	if err != nil || noRows.GetText() != "1|one "+MockMark {
		t.Errorf("with no connection the mock answered %q, %v", noRows.GetText(), err)
	}
	if _, err := WithMock(failWith(&connection.AllFailedError{}))(t.Context(), request, connection.PriorityInteractive); err == nil {
		t.Error("with a connection that failed the mock answered")
	}
	fromConnection := func(context.Context, *llm.CompleteRequest, connection.Priority) (*llm.CompleteResult, error) {
		return &llm.CompleteResult{Text: "from the connection"}, nil
	}
	got, err := WithMock(fromConnection)(t.Context(), request, connection.PriorityInteractive)
	if err != nil || got.GetText() != "from the connection" {
		t.Errorf("with a connection the answer was %q, %v", got.GetText(), err)
	}
}
