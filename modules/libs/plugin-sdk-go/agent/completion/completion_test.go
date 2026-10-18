package completion

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func ask(messages ...*numenv1.CompletionMessage) *numenv1.CompletionQuestion {
	return &numenv1.CompletionQuestion{Messages: messages}
}

func message(role, text string) *numenv1.CompletionMessage {
	return &numenv1.CompletionMessage{Role: role, Text: text}
}

func TestReadQuestionKeepsALoneMessageAsWritten(t *testing.T) {
	got, err := ReadQuestion(ask(message("system", "be brief"), message("", "why?")))
	if err != nil {
		t.Fatal(err)
	}
	if got.System != "be brief" || got.Prompt != "why?" {
		t.Errorf("got %+v", got)
	}
}

func TestReadQuestionWritesSeveralMessagesWithTheirRoles(t *testing.T) {
	got, err := ReadQuestion(ask(message("user", "a"), message("assistant", "b"), message("user", "c")))
	if err != nil {
		t.Fatal(err)
	}
	if want := "User:\na\n\nAssistant:\nb\n\nUser:\nc"; got.Prompt != want {
		t.Errorf("got %q, want %q", got.Prompt, want)
	}
}

func TestReadQuestionRefusesAQuestionWithNothingToAnswer(t *testing.T) {
	if _, err := ReadQuestion(ask(message("system", "only"))); err == nil {
		t.Error("expected an error for a question with no turn")
	}
}

func TestReadQuestionRefusesAnUnknownRole(t *testing.T) {
	if _, err := ReadQuestion(ask(message("tool", "x"))); err == nil {
		t.Error("expected an error for an unknown role")
	}
}

func TestAnswerReturnsTheTextOfTheModel(t *testing.T) {
	res, err := Answer(context.Background(), ask(message("user", "q")), func(_ context.Context, r Request) (string, error) {
		return "re: " + r.Prompt, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetAnswer().GetText() != "re: q" || res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED {
		t.Errorf("got %+v", res.Msg)
	}
}

func TestAnswerRefusesWhenTheModelFails(t *testing.T) {
	res, err := Answer(context.Background(), ask(message("user", "q")), func(context.Context, Request) (string, error) {
		return "", errors.New("no key")
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE || res.Msg.GetAnswer() != nil {
		t.Errorf("got %+v", res.Msg)
	}
}

func TestAnswerRefusesAnEmptyText(t *testing.T) {
	res, err := Answer(context.Background(), ask(message("user", "q")), func(context.Context, Request) (string, error) {
		return " \n", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE {
		t.Errorf("got %+v", res.Msg)
	}
}

func TestAnswerReportsACancelledCallAsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, err := Answer(ctx, ask(message("user", "q")), func(ctx context.Context, _ Request) (string, error) {
		cancel()
		return "late", nil
	})
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v", err)
	}
}

func TestAnswerRejectsAMalformedQuestion(t *testing.T) {
	_, err := Answer(context.Background(), ask(), func(context.Context, Request) (string, error) { return "x", nil })
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("got %v", err)
	}
}
