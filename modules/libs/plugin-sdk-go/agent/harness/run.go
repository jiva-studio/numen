package harness

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
// holds once the chat API has refused to stream, and the run's later requests do not stream.
type run struct {
	chat    chat
	model   string
	tools   *toolbox
	words   map[string]toolcall.Words
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
		if err := r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_ModelStarted{ModelStarted: &numenv1.ModelStarted{}},
		}); err != nil {
			return messages, "", err
		}
		var unsent error
		say := func(text string) error {
			unsent = r.send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Said{Said: text}})
			return unsent
		}
		reply, finished, err := r.ask(ctx, messages, say)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return messages, "", ctxErr
		}
		if unsent != nil {
			return messages, "", unsent
		}
		if err != nil {
			return messages, "", &chatFailure{err: err}
		}
		// A reply cut off at its length limit calls no tool, since its calls may not have been written whole.
		if finished == "length" {
			reply.ToolCalls = nil
		}
		for i := range reply.ToolCalls {
			if strings.TrimSpace(reply.ToolCalls[i].Function.Arguments) == "" {
				reply.ToolCalls[i].Function.Arguments = "{}"
			}
		}
		messages = append(messages, reply)

		if finished == "length" {
			return messages, reasonCut, nil
		}
		if len(reply.ToolCalls) == 0 {
			return messages, reasonDone, nil
		}
		for _, call := range reply.ToolCalls {
			result, err := r.callTool(ctx, call)
			if err != nil {
				return messages, "", err
			}
			messages = append(messages, message{Role: "tool", ToolCallID: call.ID, Content: newText(result)})
		}
	}
	return messages, reasonTurns, nil
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

// ask asks the model for the next message with its text streamed to say. A chat API that refuses
// to stream and answers the same request unstreamed is logged as such, and the run asks it
// unstreamed from then on, with the whole text going to say at once.
func (r *run) ask(ctx context.Context, messages []message, say func(string) error) (message, string, error) {
	var refusal error
	if !r.isPlain {
		reply, finished, err := r.chat.stream(ctx, r.model, messages, r.tools.offered, say)
		if !errors.Is(err, errStreamRefused) {
			return reply, finished, err
		}
		refusal = err
	}
	reply, finished, err := r.chat.send(ctx, r.model, messages, r.tools.offered)
	if err != nil {
		return message{}, "", err
	}
	if refusal != nil {
		r.isPlain = true
		log.Printf("harness: %v; it answered unstreamed, and this run goes on unstreamed", refusal)
	}
	if text := reply.Content; text != nil && *text != "" {
		if err := say(*text); err != nil {
			return message{}, "", err
		}
	}
	return reply, finished, nil
}
