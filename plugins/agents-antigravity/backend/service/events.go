package service

import (
	"encoding/json"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/jsonfield"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/jsonlrun"
)

type eventReader struct {
	reader *jsonlrun.Reader

	// isTextStreamed is set once the agent has emitted any of its answer in
	// step updates.
	isTextStreamed bool

	currentTool string
	currentArgs strings.Builder
}

// newEventHandler is what the run calls with each event of the Antigravity output.
func newEventHandler(r *jsonlrun.Reader) func(map[string]any) {
	return (&eventReader{reader: r, isTextStreamed: false, currentTool: "", currentArgs: strings.Builder{}}).handle
}

func (r *eventReader) handle(rawMap map[string]any) {
	if kind := jsonfield.GetString(rawMap, "event"); kind != "" {
		r.handleCliEvent(kind, rawMap)
		return
	}
	if evType, ok := rawMap["type"].(string); ok && evType == "stream_event" {
		r.handleStreamEvent(rawMap)
		return
	}

	for _, key := range []string{"conversation_id", "session_id"} {
		if session, ok := rawMap[key].(string); ok {
			r.reader.SetSession(session)
		}
	}

	if thinking, ok := rawMap["thinking"].(string); ok && thinking != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseModelStarted{
				ModelStarted: &agentrun.ModelStarted{},
			},
		})
	}
	r.handleToolCalls(rawMap)

	if content, ok := rawMap["content"].(string); ok && content != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseText{
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
	subType := jsonfield.GetString(evObj, "type")
	switch subType {
	case "content_block_start":
		r.startBlock(evObj)
	case "content_block_delta":
		r.continueBlock(evObj)
	case "content_block_stop":
		if r.currentTool != "" {
			r.reader.Send(&agentrun.AskAgentResponse{
				Step: &agentrun.AskAgentResponseToolCall{
					ToolCall: &agentrun.ToolCall{
						Tool:   r.currentTool,
						About:  r.currentArgs.String(),
						Length: 0, Path: "", Range: nil, Call: "", Kind: "",
					},
				},
			})
			r.reader.CountStep()
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
	if cb["type"] == "tool_use" {
		r.currentTool = jsonfield.GetString(cb, "name")
		r.currentArgs.Reset()
		if initInput, ok := cb["input"].(map[string]any); ok {
			if raw, err := json.Marshal(initInput); err == nil {
				r.currentArgs.Write(raw)
			}
		}
	}
}

func (r *eventReader) continueBlock(evObj map[string]any) {
	delta, ok := evObj["delta"].(map[string]any)
	if !ok {
		return
	}
	if text, ok := delta["text"].(string); ok && text != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseText{
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
		name := jsonfield.GetString(tc, "name")
		args, err := json.Marshal(tc["arguments"])
		if err != nil {
			args = nil
		}
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseToolCall{
				ToolCall: &agentrun.ToolCall{
					Tool:   name,
					About:  string(args),
					Length: 0, Path: "", Range: nil, Call: "", Kind: "",
				},
			},
		})
		r.reader.CountStep()
	}
}

func (r *eventReader) handleStepUpdate(rawMap map[string]any) {
	delta, ok := rawMap["delta"].(map[string]any)
	if !ok {
		return
	}
	if text, ok := delta["text"].(string); ok && text != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseText{
				Text: text,
			},
		})
	}
}

func (r *eventReader) handleResult(rawMap map[string]any) {
	if text, ok := rawMap["result"].(string); ok && text != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseText{
				Text: text,
			},
		})
	}
	if reason, ok := rawMap["stopped"].(string); ok && reason != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseStopped{
				Stopped: reason,
			},
		})
	}
}

// handleCliEvent handles an event of the stream the CLI writes: "init",
// "step_update" and "result". The agent's text arrives as the text deltas of
// its response steps, and a run that ends in error reports it in its result.
func (r *eventReader) handleCliEvent(kind string, rawMap map[string]any) {
	switch kind {
	case "step_update":
		step := jsonfield.GetMap(rawMap, "step_update")
		r.handleCliStep(step)
	case "result":
		result := jsonfield.GetMap(rawMap, "result")
		r.handleCliResult(result)
	default:
		if session, ok := rawMap["conversation_id"].(string); ok {
			r.reader.SetSession(session)
		}
	}
}

func (r *eventReader) handleCliStep(step map[string]any) {
	if session, ok := step["conversation_id"].(string); ok {
		r.reader.SetSession(session)
	}
	text := jsonfield.GetString(step, "text_delta")
	if kind := jsonfield.GetString(step, "step_type"); kind != "agent_response" || text == "" {
		return
	}
	r.isTextStreamed = true
	r.reader.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseText{Text: text}})
}

// handleCliResult emits the response of a run that streamed no text, and stops
// a run whose status is not success with the error the CLI reported.
func (r *eventReader) handleCliResult(result map[string]any) {
	r.reader.Finish()
	if session, ok := result["conversation_id"].(string); ok {
		r.reader.SetSession(session)
	}
	if status := jsonfield.GetString(result, "status"); status != "" && !strings.EqualFold(status, "success") {
		reason := jsonfield.GetString(result, "error")
		if reason == "" {
			reason = "the run ended with status " + status
		}
		r.reader.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseStopped{Stopped: reason}})
		return
	}
	if response := jsonfield.GetString(result, "response"); response != "" && !r.isTextStreamed {
		r.reader.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseText{Text: response}})
	}
}
