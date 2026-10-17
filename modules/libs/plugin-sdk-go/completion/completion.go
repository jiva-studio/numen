// Package completion answers CompletionService.Complete for an agent plugin.
//
// A plugin supplies the function that asks its model; the package reads the question, turns every
// failure of that function into an UNAVAILABLE refusal that carries the reason, and passes a
// cancelled call back as cancelled. An answer is the model's text and nothing else.
package completion

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// ErrEmptyAnswer means the model replied with no text.
var ErrEmptyAnswer = errors.New("completion: the model sent no text")

// Request is a question read for a model: the system prompt, the prompt and the hints.
type Request struct {
	System string
	Prompt string
	Hints  *numenv1.CompletionHints
}

// Runner asks the plugin's model and returns its text.
type Runner func(ctx context.Context, request Request) (string, error)

// Answer reads the question, asks run and returns the response for Complete.
// A malformed question is an invalid-argument error and a cancelled context is a cancelled error.
// Any other failure of run is answered as UNAVAILABLE.
func Answer(
	ctx context.Context,
	question *numenv1.CompletionQuestion,
	run Runner,
) (*connect.Response[numenv1.CompleteResponse], error) {
	request, err := ReadQuestion(question)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	text, err := run(ctx, request)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if err != nil {
		return Refuse(err.Error()), nil
	}
	if strings.TrimSpace(text) == "" {
		return Refuse(ErrEmptyAnswer.Error()), nil
	}
	return connect.NewResponse(&numenv1.CompleteResponse{
		Answer: &numenv1.CompletionAnswer{Text: text},
	}), nil
}

// Refuse answers UNAVAILABLE. The response has no field for the reason, so it goes to the log.
func Refuse(reason string) *connect.Response[numenv1.CompleteResponse] {
	log.Printf("completion: unavailable: %s", reason)
	return connect.NewResponse(&numenv1.CompleteResponse{
		Error: numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE,
	})
}

// ReadQuestion splits the question into its system prompt and its prompt.
// A lone message is the prompt as written; several are written out with their roles.
func ReadQuestion(question *numenv1.CompletionQuestion) (Request, error) {
	var system []string
	var turns []*numenv1.CompletionMessage
	for _, message := range question.GetMessages() {
		switch message.GetRole() {
		case "system":
			system = append(system, message.GetText())
		case "", "user", "assistant":
			turns = append(turns, message)
		default:
			return Request{}, fmt.Errorf("completion: unknown role %q", message.GetRole())
		}
	}
	if len(turns) == 0 {
		return Request{}, errors.New("completion: the question holds no message to answer")
	}
	return Request{
		System: strings.Join(system, "\n\n"),
		Prompt: writePrompt(turns),
		Hints:  question.GetHints(),
	}, nil
}

func writePrompt(turns []*numenv1.CompletionMessage) string {
	if len(turns) == 1 {
		return turns[0].GetText()
	}
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		role := "User"
		if turn.GetRole() == "assistant" {
			role = "Assistant"
		}
		parts = append(parts, role+":\n"+turn.GetText())
	}
	return strings.Join(parts, "\n\n")
}
