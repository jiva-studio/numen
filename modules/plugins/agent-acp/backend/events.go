package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type eventReader struct {
	registry     *agentrun.Registry
	work         *agentrun.Work
	limit        int
	done         int
	conversation string

	currentTool string
	currentArgs strings.Builder
}

func (r *eventReader) send(resp *numenv1.AskAgentResponse) {
	select {
	case r.work.Output <- resp:
	case <-r.work.Stopped():
		return
	}
	if resp.GetToolFinished() != nil {
		r.done++
	}
}

func (r *eventReader) isOverLimit() bool {
	return r.limit > 0 && r.done >= r.limit
}

func (r *eventReader) handle(rawMap map[string]any) {
	if evType, ok := rawMap["type"].(string); ok && evType == "stream_event" {
		r.handleStreamEvent(rawMap)
		return
	}

	r.rememberSession(rawMap)

	if thinking, ok := rawMap["thinking"].(string); ok && thinking != "" {
		r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_ModelStarted{
				ModelStarted: &numenv1.ModelStarted{},
			},
		})
	}
	r.handleCodexEvent(rawMap)
	r.handleToolCalls(rawMap)

	if content, ok := rawMap["content"].(string); ok && content != "" {
		r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: content,
			},
		})
	}
	r.handleStepUpdate(rawMap)
	r.handleSessionUpdate(rawMap)
	r.handleResult(rawMap)
}

func (r *eventReader) rememberSession(rawMap map[string]any) {
	if r.conversation == "" {
		return
	}
	for _, key := range []string{"conversation_id", "session_id"} {
		if sessID, ok := rawMap[key].(string); ok && sessID != "" {
			r.registry.SetSession(r.conversation, sessID)
		}
	}
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
			r.send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_ToolCall{
					ToolCall: &numenv1.ToolCall{
						Tool:  r.currentTool,
						About: r.currentArgs.String(),
					},
				},
			})
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
		r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: text,
			},
		})
	}
	if partial, ok := delta["partial_json"].(string); ok {
		r.currentArgs.WriteString(partial)
	}
}

func (r *eventReader) handleCodexEvent(rawMap map[string]any) {
	evType, _ := rawMap["type"].(string)
	item, _ := rawMap["item"].(map[string]any)

	switch evType {
	case "item.started":
		if item != nil && item["type"] == "mcp_tool_call" {
			name, _ := item["name"].(string)
			args, _ := json.Marshal(item["arguments"])
			r.send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_ToolCall{
					ToolCall: &numenv1.ToolCall{
						Tool:  name,
						About: string(args),
					},
				},
			})
		}
	case "item.completed":
		if item != nil && item["type"] == "mcp_tool_call" {
			r.send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_ToolFinished{
					ToolFinished: &numenv1.ToolFinished{},
				},
			})
		}
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
		r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_ToolCall{
				ToolCall: &numenv1.ToolCall{
					Tool:  name,
					About: string(args),
				},
			},
		})
	}
}

func (r *eventReader) handleStepUpdate(rawMap map[string]any) {
	delta, ok := rawMap["delta"].(map[string]any)
	if !ok {
		return
	}
	if text, ok := delta["text"].(string); ok && text != "" {
		r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: text,
			},
		})
	}
}

func (r *eventReader) handleSessionUpdate(rawMap map[string]any) {
	if rawMap["type"] == "session.updated" {
		if sessID, ok := rawMap["session_id"].(string); ok && r.conversation != "" {
			r.registry.SetSession(r.conversation, sessID)
		}
	}
}

func (r *eventReader) handleResult(rawMap map[string]any) {
	if text, ok := rawMap["result"].(string); ok && text != "" {
		r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: text,
			},
		})
	}
	if reason, ok := rawMap["stopped"].(string); ok && reason != "" {
		r.send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Stopped{
				Stopped: reason,
			},
		})
	}
}

func (r *eventReader) reportLimit() {
	r.send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_Stopped{
			Stopped: fmt.Sprintf("reached maximum limit of %d steps", r.limit),
		},
	})
}
