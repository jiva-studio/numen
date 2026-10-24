package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/toolcall"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// defaultTurns is how many requests to the model one task may make where the task names no limit.
const defaultTurns = 25

// reasonDone is why a run stops when the model has answered.
const reasonDone = "done"

// reasonTurns is why a run stops when the model is still calling tools after its last turn.
const reasonTurns = "the agent reached the limit of steps without finishing"

// reasonCut is why a run stops when the model's answer was cut off at its length limit.
const reasonCut = "the answer was cut off at the model's length limit"

// chatFailure is a run stopped by a request to the chat API that failed. It stops the run and is
// reported as why it stopped.
type chatFailure struct{ err error }

func (f *chatFailure) Error() string { return f.err.Error() }

func (f *chatFailure) Unwrap() error { return f.err }

// run is one task worked by the model: the chat API, the tools, and where its steps go. isPlain
type run struct {
	chat    chat
	model   string
	effort  effort
	tools   *toolbox
	words   map[string]toolcall.Declaration
	turns   int
	send    func(*numenv1.AskAgentResponse) error
	isPlain bool
}

// answer asks the model until it answers without calling a tool or runs out of turns, calling
// every tool it asks for and telling it what the tool returned. It returns the conversation as it
// ends and why the run stopped. The error is the context's, a chatFailure, or a step that could
// not be sent.
func (r *run) answer(ctx context.Context, messages []message) ([]message, string, error) {
	turns := r.turns
	if turns <= 0 {
		turns = defaultTurns
	}
	for range turns {
		var (
			reason string
			isDone bool
			err    error
		)
		messages, reason, isDone, err = r.turn(ctx, messages)
		if err != nil {
			return messages, "", err
		}
		if isDone {
			return messages, reason, nil
		}
	}
	return messages, reasonTurns, nil
}

func (r *run) turn(ctx context.Context, messages []message) ([]message, string, bool, error) {
	if err := r.send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ModelStarted{ModelStarted: &numenv1.ModelStarted{}},
	}); err != nil {
		return messages, "", false, err
	}
	var unsent error
	emit := func(text string) error {
		unsent = r.send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Text{Text: text}})
		return unsent
	}
	reply, finished, err := r.ask(ctx, messages, emit)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return messages, "", false, ctxErr
	}
	if unsent != nil {
		return messages, "", false, unsent
	}
	if err != nil {
		return messages, "", false, &chatFailure{err: err}
	}
	if finished == "length" {
		reply.ToolCalls = nil
		messages = append(messages, reply)
		return messages, reasonCut, true, nil
	}
	cleanToolCalls(reply.ToolCalls)
	messages = append(messages, reply)
	if len(reply.ToolCalls) == 0 {
		return messages, reasonDone, true, nil
	}
	messages, err = r.runToolCalls(ctx, messages, reply.ToolCalls)
	return messages, "", false, err
}

func cleanToolCalls(calls []toolCall) {
	for i := range calls {
		if strings.TrimSpace(calls[i].Function.Arguments) == "" {
			calls[i].Function.Arguments = "{}"
		}
	}
}

func (r *run) runToolCalls(ctx context.Context, messages []message, calls []toolCall) ([]message, error) {
	for _, call := range calls {
		result, err := r.callTool(ctx, call)
		if err != nil {
			return messages, err
		}
		messages = append(messages, message{Role: "tool", ToolCallID: call.ID, Content: newText(result)})
	}
	return messages, nil
}

// callTool tells of the call, runs it and tells that it has finished. It returns what the model is
// told the tool returned.
func (r *run) callTool(ctx context.Context, call toolCall) (string, error) {
	name, args := call.Function.Name, call.Function.Arguments
	described := toolcall.Describe(r.words, name, call.ID, args)
	if err := r.send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_ToolCall{ToolCall: described}}); err != nil {
		return "", err
	}
	result := r.tools.call(ctx, name, args)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := r.send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ToolFinished{ToolFinished: &numenv1.ToolFinished{}},
	}); err != nil {
		return "", err
	}
	return result, nil
}

// ask asks the model for the next message with its text streamed to emit. A chat API that refuses
// to stream and answers the same request unstreamed is logged as such, and the run asks it
// unstreamed from then on, with the whole text going to emit at once.
func (r *run) ask(ctx context.Context, messages []message, emit func(string) error) (message, string, error) {
	var reqErr error
	if !r.isPlain {
		reply, finished, err := r.chat.stream(ctx, r.model, r.effort, messages, r.tools.tools, emit)
		if !errors.Is(err, errStreamRejected) {
			return reply, finished, err
		}
		reqErr = err
	}
	reply, finished, err := r.chat.send(ctx, r.model, r.effort, messages, r.tools.tools)
	if err != nil {
		return message{}, "", err
	}
	if reqErr != nil {
		r.isPlain = true
		log.Printf("harness: %v; it answered unstreamed, and this run goes on unstreamed", reqErr)
	}
	if text := reply.Content; text != nil && *text != "" {
		if err := emit(*text); err != nil {
			return message{}, "", err
		}
	}
	return reply, finished, nil
}
