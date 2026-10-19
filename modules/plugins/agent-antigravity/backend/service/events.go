package service

import (
	"encoding/json"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type eventReader struct {
	reader *agentrun.Reader

	currentTool string
	currentArgs strings.Builder
}

// newEventHandler is what the run calls with each event of the Antigravity output.
func newEventHandler(r *agentrun.Reader) func(map[string]any) {
	return (&eventReader{reader: r}).handle
}

func (r *eventReader) handle(rawMap map[string]any) {
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
		r.reader.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_ModelStarted{
				ModelStarted: &numenv1.ModelStarted{},
			},
		})
	}
	r.handleToolCalls(rawMap)

	if content, ok := rawMap["content"].(string); ok && content != "" {
		r.reader.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: content,
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
			r.reader.Send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_ToolCall{
					ToolCall: &numenv1.ToolCall{
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
		r.reader.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: text,
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
		r.reader.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_ToolCall{
				ToolCall: &numenv1.ToolCall{
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
		r.reader.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: text,
			},
		})
	}
}

func (r *eventReader) handleResult(rawMap map[string]any) {
	if text, ok := rawMap["result"].(string); ok && text != "" {
		r.reader.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: text,
			},
		})
	}
	if reason, ok := rawMap["stopped"].(string); ok && reason != "" {
		r.reader.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Stopped{
				Stopped: reason,
			},
		})
	}
}
