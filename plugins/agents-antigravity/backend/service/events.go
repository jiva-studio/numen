package service

import (
	"encoding/json"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"strings"
)

type eventReader struct {
	reader *reader

	// isTextStreamed is set once the agent has said any of its answer in step updates.
	isTextStreamed bool

	currentTool string
	currentArgs strings.Builder
}

// newEventHandler is what the run calls with each event of the Antigravity output.
func newEventHandler(r *reader) func(map[string]any) {
	return (&eventReader{reader: r}).handle
}

func (r *eventReader) handle(rawMap map[string]any) {
	if kind, _ := rawMap["event"].(string); kind != "" {
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
			Step: &agentrun.AskAgentResponse_ModelStarted{
				ModelStarted: &agentrun.ModelStarted{},
			},
		})
	}
	r.handleToolCalls(rawMap)

	if content, ok := rawMap["content"].(string); ok && content != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponse_Text{
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
	subType, _ := evObj["type"].(string)
	switch subType {
	case "content_block_start":
		r.startBlock(evObj)
	case "content_block_delta":
		r.continueBlock(evObj)
	case "content_block_stop":
		if r.currentTool != "" {
			r.reader.Send(&agentrun.AskAgentResponse{
				Step: &agentrun.AskAgentResponse_ToolCall{
					ToolCall: &agentrun.ToolCall{
						Tool:  r.currentTool,
						About: r.currentArgs.String(),
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
		r.currentTool, _ = cb["name"].(string)
		r.currentArgs.Reset()
		if initInput, ok := cb["input"].(map[string]any); ok {
			raw, _ := json.Marshal(initInput)
			r.currentArgs.WriteString(string(raw))
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
			Step: &agentrun.AskAgentResponse_Text{
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
		name, _ := tc["name"].(string)
		args, _ := json.Marshal(tc["arguments"])
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponse_ToolCall{
				ToolCall: &agentrun.ToolCall{
					Tool:  name,
					About: string(args),
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
			Step: &agentrun.AskAgentResponse_Text{
				Text: text,
			},
		})
	}
}

func (r *eventReader) handleResult(rawMap map[string]any) {
	if text, ok := rawMap["result"].(string); ok && text != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponse_Text{
				Text: text,
			},
		})
	}
	if reason, ok := rawMap["stopped"].(string); ok && reason != "" {
		r.reader.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponse_Stopped{
				Stopped: reason,
			},
		})
	}
}

// handleCliEvent reads an event of the stream the CLI writes: "init", "step_update" and "result".
// What the agent says arrives as the text deltas of its response steps, and a run that ends in
// error does so in its result.
func (r *eventReader) handleCliEvent(kind string, rawMap map[string]any) {
	switch kind {
	case "step_update":
		step, _ := rawMap["step_update"].(map[string]any)
		r.handleCliStep(step)
	case "result":
		result, _ := rawMap["result"].(map[string]any)
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
	text, _ := step["text_delta"].(string)
	if kind, _ := step["step_type"].(string); kind != "agent_response" || text == "" {
		return
	}
	r.isTextStreamed = true
	r.reader.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Text{Text: text}})
}

// handleCliResult says the response of a run that streamed none, and stops a run whose status is
// not success with the error the CLI gave.
func (r *eventReader) handleCliResult(result map[string]any) {
	r.reader.isFinished = true
	if session, ok := result["conversation_id"].(string); ok {
		r.reader.SetSession(session)
	}
	if status, _ := result["status"].(string); status != "" && !strings.EqualFold(status, "success") {
		reason, _ := result["error"].(string)
		if reason == "" {
			reason = "the run ended with status " + status
		}
		r.reader.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Stopped{Stopped: reason}})
		return
	}
	if response, _ := result["response"].(string); response != "" && !r.isTextStreamed {
		r.reader.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Text{Text: response}})
	}
}
