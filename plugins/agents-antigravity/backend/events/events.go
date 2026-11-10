// Package events turns the JSON events the Antigravity program prints into the steps of an agent run.
package events

import (
	"encoding/json"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
)

// Sink receives what the events of a run say.
type Sink interface {
	Send(resp *agentrun.Response)
	CountStep()
	SetSession(session string)
	Finish()
}

type eventReader struct {
	sink Sink

	isTextStreamed bool

	currentTool string
	currentArgs strings.Builder
}

// NewHandler returns the function that turns one decoded event of the program's output into steps
// for sink.
func NewHandler(sink Sink) func(map[string]any) {
	return (&eventReader{sink: sink, isTextStreamed: false, currentTool: "", currentArgs: strings.Builder{}}).handle
}

func (r *eventReader) handle(rawMap map[string]any) {
	if kind := getString(rawMap, "event"); kind != "" {
		r.handleCliEvent(kind, rawMap)
		return
	}
	if evType, ok := rawMap["type"].(string); ok && evType == "stream_event" {
		r.handleStreamEvent(rawMap)
		return
	}

	for _, key := range []string{"conversation_id", "session_id"} {
		if session, ok := rawMap[key].(string); ok {
			r.sink.SetSession(session)
		}
	}

	if thinking, ok := rawMap["thinking"].(string); ok && thinking != "" {
		r.sink.Send(&agentrun.Response{
			Step: &agentrun.ModelStartedStep{},
		})
	}
	r.handleToolCalls(rawMap)

	if content, ok := rawMap["content"].(string); ok && content != "" {
		r.sink.Send(&agentrun.Response{
			Step: &agentrun.TextStep{
				Text: content,
			},
		})
	}
	r.handleStepUpdate(rawMap)
	r.handleResult(rawMap)
}

func (r *eventReader) handleStreamEvent(rawMap map[string]any) {
	evObj, ok := rawMap["event"].(map[string]any)
	if !ok {
		return
	}
	subType := getString(evObj, "type")
	switch subType {
	case "content_block_start":
		r.startBlock(evObj)
	case "content_block_delta":
		r.continueBlock(evObj)
	case "content_block_stop":
		if r.currentTool != "" {
			r.sink.Send(&agentrun.Response{
				Step: &agentrun.ToolCallStep{
					ToolCall: &toolcall.ToolCall{
						Tool:   r.currentTool,
						About:  r.currentArgs.String(),
						Length: 0, Path: "", Range: nil, Call: "", Kind: "",
					},
				},
			})
			r.sink.CountStep()
			r.currentTool = ""
			r.currentArgs.Reset()
		}
	}
}

func (r *eventReader) startBlock(evObj map[string]any) {
	cb, ok := evObj["content_block"].(map[string]any)
	if !ok {
		return
	}
	if cb["type"] != "tool_use" {
		return
	}
	r.currentTool = getString(cb, "name")
	r.currentArgs.Reset()
	initInput, ok := cb["input"].(map[string]any)
	if !ok {
		return
	}
	if raw, err := json.Marshal(initInput); err == nil {
		r.currentArgs.Write(raw)
	}
}

func (r *eventReader) continueBlock(evObj map[string]any) {
	delta, ok := evObj["delta"].(map[string]any)
	if !ok {
		return
	}
	if text, ok := delta["text"].(string); ok && text != "" {
		r.sink.Send(&agentrun.Response{
			Step: &agentrun.TextStep{
				Text: text,
			},
		})
	}
	if partial, ok := delta["partial_json"].(string); ok {
		r.currentArgs.WriteString(partial)
	}
}

func (r *eventReader) handleToolCalls(rawMap map[string]any) {
	toolCalls, ok := rawMap["tool_calls"].([]any)
	if !ok {
		return
	}
	for _, tcRaw := range toolCalls {
		tc, ok := tcRaw.(map[string]any)
		if !ok {
			continue
		}
		name := getString(tc, "name")
		args, err := json.Marshal(tc["arguments"])
		if err != nil {
			args = nil
		}
		r.sink.Send(&agentrun.Response{
			Step: &agentrun.ToolCallStep{
				ToolCall: &toolcall.ToolCall{
					Tool:   name,
					About:  string(args),
					Length: 0, Path: "", Range: nil, Call: "", Kind: "",
				},
			},
		})
		r.sink.CountStep()
	}
}

func (r *eventReader) handleStepUpdate(rawMap map[string]any) {
	delta, ok := rawMap["delta"].(map[string]any)
	if !ok {
		return
	}
	if text, ok := delta["text"].(string); ok && text != "" {
		r.sink.Send(&agentrun.Response{
			Step: &agentrun.TextStep{
				Text: text,
			},
		})
	}
}

func (r *eventReader) handleResult(rawMap map[string]any) {
	if text, ok := rawMap["result"].(string); ok && text != "" {
		r.sink.Send(&agentrun.Response{
			Step: &agentrun.TextStep{
				Text: text,
			},
		})
	}
	if reason, ok := rawMap["stopped"].(string); ok && reason != "" {
		r.sink.Send(&agentrun.Response{
			Step: &agentrun.StoppedStep{
				Stopped: reason,
			},
		})
	}
}

func (r *eventReader) handleCliEvent(kind string, rawMap map[string]any) {
	switch kind {
	case "step_update":
		step := getMap(rawMap, "step_update")
		r.handleCliStep(step)
	case "result":
		result := getMap(rawMap, "result")
		r.handleCliResult(result)
	default:
		if session, ok := rawMap["conversation_id"].(string); ok {
			r.sink.SetSession(session)
		}
	}
}

func (r *eventReader) handleCliStep(step map[string]any) {
	if session, ok := step["conversation_id"].(string); ok {
		r.sink.SetSession(session)
	}
	text := getString(step, "text_delta")
	if kind := getString(step, "step_type"); kind != "agent_response" || text == "" {
		return
	}
	r.isTextStreamed = true
	r.sink.Send(&agentrun.Response{Step: &agentrun.TextStep{Text: text}})
}

func (r *eventReader) handleCliResult(result map[string]any) {
	r.sink.Finish()
	if session, ok := result["conversation_id"].(string); ok {
		r.sink.SetSession(session)
	}
	if status := getString(result, "status"); status != "" && !strings.EqualFold(status, "success") {
		reason := getString(result, "error")
		if reason == "" {
			reason = "the run ended with status " + status
		}
		r.sink.Send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
		return
	}
	if response := getString(result, "response"); response != "" && !r.isTextStreamed {
		r.sink.Send(&agentrun.Response{Step: &agentrun.TextStep{Text: response}})
	}
}
