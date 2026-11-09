package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

func failWith(err error) completer {
	return func(context.Context, connection.CompleteRequest) (connection.CompleteResult, error) {
		return connection.CompleteResult{}, err
	}
}

func TestProofreadWithNoConnectionIsOffAndPointsToTheConnectionsPage(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.run.ask = newAskModel(failWith(connection.ErrNoConnection))

	_, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()})

	if !errors.Is(err, errNoConnection) || !strings.Contains(err.Error(), "Connections") {
		t.Fatalf("got %v, want %v", err, errNoConnection)
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "Connections") {
		t.Errorf("tasks %+v", tasks)
	}
	if _, live := s.readLive(t, scanPath); live.PluginID != ocrID {
		t.Errorf("a proofreading that is off left the live version %+v", live)
	}
}

func TestProofreadNamesEveryConnectionThatFailedInTheTask(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	failed := &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "rate limited"},
		{ConnectionName: "Local", Reason: "unreachable"},
	}}
	s.run.ask = newAskModel(failWith(failed))

	_, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()})

	if _, isAllFailed := errors.AsType[*connection.AllFailedError](err); !isAllFailed {
		t.Fatalf("got %v, want every connection failed", err)
	}
	tasks := s.queue.ListTasks()
	if len(tasks) != 1 || !strings.Contains(tasks[0].FailureReason, "OpenRouter: rate limited; Local: unreachable") {
		t.Errorf("tasks %+v", tasks)
	}
}

func TestAMistakeInTheRequestEndsTheProofreadingWithTheConnectionThatFoundIt(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.run.ask = newAskModel(failWith(&connection.RejectedError{ConnectionName: "OpenRouter", Reason: "bad request"}))

	_, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()})

	var rejected *connection.RejectedError
	if !errors.As(err, &rejected) || rejected.ConnectionName != "OpenRouter" {
		t.Fatalf("got %v, want a rejection by OpenRouter", err)
	}
}

func TestTheMockAnswersOnlyWhereNoConnectionIsSet(t *testing.T) {
	prompt := "What it holds\n\n" + buildPrompt(Batch{Lines: []Line{{Number: 1, Text: "one"}}})
	request := connection.CompleteRequest{Messages: []connection.Message{{Role: "system", Text: "s"}, {Role: "user", Text: prompt}}}

	noRows, err := completeOrMock(failWith(connection.ErrNoConnection))(t.Context(), request)
	if err != nil || noRows.Text != "1|one "+MockMark {
		t.Errorf("with no connection the mock answered %q, %v", noRows.Text, err)
	}
	if _, err := completeOrMock(failWith(&connection.AllFailedError{}))(t.Context(), request); err == nil {
		t.Error("with a connection that failed the mock answered")
	}
	fromConnection := func(context.Context, connection.CompleteRequest) (connection.CompleteResult, error) {
		return connection.CompleteResult{Text: "from the connection"}, nil
	}
	got, err := completeOrMock(fromConnection)(t.Context(), request)
	if err != nil || got.Text != "from the connection" {
		t.Errorf("with a connection the answer was %q, %v", got.Text, err)
	}
}
