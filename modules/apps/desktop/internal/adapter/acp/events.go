package acp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// eventReader turns the lines an agent writes into the steps of its work. It
// holds the tool call a streamed block is building up.
type eventReader struct {
	agent        *Agent
	work         *work
	conversation string

	currentTool string
	currentArgs strings.Builder
}

// send reports a step.
func (r *eventReader) send(step port.Step) {
	r.work.steps <- step
}

// handle reports the steps one decoded line of the agent's output carries.
func (r *eventReader) handle(rawMap map[string]any) {
	if evType, ok := rawMap["type"].(string); ok && evType == "stream_event" {
		r.handleStreamEvent(rawMap)
		return
	}

	r.rememberSession(rawMap)

	if thinking, ok := rawMap["thinking"].(string); ok && thinking != "" {
		r.send(port.Step{Kind: domain.StepThinking})
	}
	r.handleCodexEvent(rawMap)
	r.handleToolCalls(rawMap)

	if content, ok := rawMap["content"].(string); ok && content != "" {
		r.send(port.Step{Kind: domain.StepSaying, Text: content})
	}
	r.handleStepUpdate(rawMap)
	r.handleSessionUpdate(rawMap)
	r.handleResult(rawMap)
}

// rememberSession keeps the session the line names for the conversation, so a
// later task of it carries on in the same session.
func (r *eventReader) rememberSession(rawMap map[string]any) {
	if r.conversation == "" {
		return
	}
	for _, key := range []string{"conversation_id", "session_id"} {
		if sessID, ok := rawMap[key].(string); ok && sessID != "" {
			r.agent.carried.Lock()
			if r.agent.carried.sessions == nil {
				r.agent.carried.sessions = map[string]string{}
			}
			r.agent.carried.sessions[r.conversation] = sessID
			r.agent.carried.Unlock()
		}
	}
}

// handleStreamEvent reports the steps of a streamed content block.
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
			r.send(port.Step{Kind: domain.StepToolCall, Tool: r.currentTool, About: r.currentArgs.String()})
			r.currentTool = ""
			r.currentArgs.Reset()
		}
	}
}

// startBlock opens a tool call where the block is one.
func (r *eventReader) startBlock(evObj map[string]any) {
	blk, ok := evObj["content_block"].(map[string]any)
	if !ok {
		return
	}
	if blkType, _ := blk["type"].(string); blkType == "tool_use" {
		r.currentTool, _ = blk["name"].(string)
		r.currentArgs.Reset()
		r.send(port.Step{Kind: domain.StepToolCall, Tool: r.currentTool})
	}
}

// continueBlock reports what a block has added to the words, the thinking or
// the arguments of the tool call.
func (r *eventReader) continueBlock(evObj map[string]any) {
	delta, ok := evObj["delta"].(map[string]any)
	if !ok {
		return
	}
	deltaType, _ := delta["type"].(string)
	switch deltaType {
	case "text_delta":
		if t, _ := delta["text"].(string); t != "" {
			r.send(port.Step{Kind: domain.StepSaying, Text: t})
		}
	case "thinking_delta":
		r.send(port.Step{Kind: domain.StepThinking})
	case "input_json_delta":
		if partial, _ := delta["partial_json"].(string); partial != "" {
			r.currentArgs.WriteString(partial)
			r.send(port.Step{Kind: domain.StepToolCall, Tool: r.currentTool, About: r.currentArgs.String()})
		}
	}
}

// handleCodexEvent reports the steps of a codex event: a turn, an item started,
// updated or completed, or an error.
func (r *eventReader) handleCodexEvent(rawMap map[string]any) {
	evType, ok := rawMap["type"].(string)
	if !ok {
		return
	}
	item, _ := rawMap["item"].(map[string]any)
	switch evType {
	case "turn.started":
		r.send(port.Step{Kind: domain.StepThinking})
	case "item.started":
		r.handleItemStarted(item)
	case "item.updated":
		r.handleItemUpdated(item)
	case "item.completed":
		r.handleItemCompleted(item)
	case "error", "turn.failed":
		r.handleCodexError(rawMap)
	case "GENERIC", "user":
		r.send(port.Step{Kind: domain.StepAnswered})
	}
}

// handleItemStarted reports the tool call an item opens.
func (r *eventReader) handleItemStarted(item map[string]any) {
	if item == nil {
		return
	}
	iType, _ := item["type"].(string)
	switch iType {
	case "mcp_tool_call":
		toolName, _ := item["tool"].(string)
		if toolName == "" {
			toolName, _ = item["name"].(string)
		}
		r.send(port.Step{Kind: domain.StepToolCall, Tool: toolName, About: formatToolArgs(item["arguments"])})
	case "command_execution":
		cmdStr, _ := item["command"].(string)
		r.send(port.Step{Kind: domain.StepToolCall, Tool: "bash", About: cmdStr})
	case "file_read":
		pathStr, _ := item["path"].(string)
		r.send(port.Step{Kind: domain.StepToolCall, Tool: "read", About: pathStr})
	}
}

// handleItemUpdated reports the words an agent message has added.
func (r *eventReader) handleItemUpdated(item map[string]any) {
	if item == nil {
		return
	}
	if iType, _ := item["type"].(string); iType != "agent_message" {
		return
	}
	if delta, ok := item["delta"].(string); ok && delta != "" {
		r.send(port.Step{Kind: domain.StepSaying, Text: delta})
	}
}

// handleItemCompleted reports the message an item ends with, or that its tool
// call has been answered.
func (r *eventReader) handleItemCompleted(item map[string]any) {
	if item == nil {
		return
	}
	iType, _ := item["type"].(string)
	switch iType {
	case "agent_message":
		if text, ok := item["text"].(string); ok && text != "" {
			r.send(port.Step{Kind: domain.StepSaying, Text: text})
		}
	case "mcp_tool_call", "command_execution", "file_read":
		r.send(port.Step{Kind: domain.StepAnswered})
	}
}

// handleCodexError reports the message an error event carries.
func (r *eventReader) handleCodexError(rawMap map[string]any) {
	if errObj, ok := rawMap["error"].(map[string]any); ok {
		msg, _ := errObj["message"].(string)
		r.send(port.Step{Kind: domain.StepStopped, Detail: msg})
	} else if msg, ok := rawMap["message"].(string); ok && msg != "" {
		r.send(port.Step{Kind: domain.StepStopped, Detail: msg})
	}
}

// handleToolCalls reports the tool calls of a line, which carries an array of
// them.
func (r *eventReader) handleToolCalls(rawMap map[string]any) {
	toolCalls, ok := rawMap["tool_calls"].([]any)
	if !ok {
		return
	}
	for _, tc := range toolCalls {
		tcMap, ok := tc.(map[string]any)
		if !ok {
			continue
		}
		toolName, _ := tcMap["name"].(string)
		if toolName == "" {
			toolName, _ = tcMap["tool"].(string)
		}
		if toolName != "" {
			r.send(port.Step{Kind: domain.StepToolCall, Tool: toolName, About: formatToolArgs(tcMap["args"])})
		}
	}
}

// handleStepUpdate reports the step an antigravity step update wraps.
func (r *eventReader) handleStepUpdate(rawMap map[string]any) {
	suMap, ok := rawMap["step_update"].(map[string]any)
	if !ok {
		return
	}
	stepType, _ := suMap["step_type"].(string)
	switch stepType {
	case "agent_response":
		if delta, ok := suMap["text_delta"].(string); ok && delta != "" {
			r.send(port.Step{Kind: domain.StepSaying, Text: delta})
		}
	case "thinking":
		r.send(port.Step{Kind: domain.StepThinking})
	case "tool_call":
		toolName, argsVal := getStepTool(suMap)
		r.send(port.Step{Kind: domain.StepToolCall, Tool: toolName, About: formatToolArgs(argsVal)})
	case "tool_result":
		r.send(port.Step{Kind: domain.StepAnswered})
	}
}

// getStepTool is the tool a step update calls and its arguments. A call to the
// generic MCP tool names the tool it calls among its arguments.
func getStepTool(suMap map[string]any) (string, any) {
	toolName, _ := suMap["tool_name"].(string)
	argsVal := suMap["arguments"]
	if toolName != "call_mcp_tool" {
		return toolName, argsVal
	}
	if mArgs, ok := argsVal.(map[string]any); ok {
		if tName, ok := mArgs["ToolName"].(string); ok && tName != "" {
			return tName, mArgs["Arguments"]
		}
	}
	return toolName, argsVal
}

// handleSessionUpdate reports the step an ACP session update carries.
func (r *eventReader) handleSessionUpdate(rawMap map[string]any) {
	if method, _ := rawMap["method"].(string); method != "session/update" {
		return
	}
	params, ok := rawMap["params"].(map[string]any)
	if !ok {
		return
	}
	update, ok := params["update"].(map[string]any)
	if !ok {
		return
	}
	uType, _ := update["type"].(string)
	switch uType {
	case "agent_message":
		if text, _ := update["text"].(string); text != "" {
			r.send(port.Step{Kind: domain.StepSaying, Text: text})
		}
	case "agent_thought":
		r.send(port.Step{Kind: domain.StepThinking})
	case "tool_call":
		toolName, _ := update["tool"].(string)
		r.send(port.Step{Kind: domain.StepToolCall, Tool: toolName, About: formatToolArgs(update["arguments"])})
	case "tool_result":
		r.send(port.Step{Kind: domain.StepAnswered})
	}
}

// handleResult reports a result that ends in error.
func (r *eventReader) handleResult(rawMap map[string]any) {
	resMap, ok := rawMap["result"].(map[string]any)
	if !ok {
		return
	}
	if st, _ := resMap["status"].(string); st == "ERROR" {
		errMsg, _ := resMap["error"].(string)
		r.send(port.Step{Kind: domain.StepStopped, Detail: errMsg})
	}
}

// formatToolArgs is the arguments of a tool call as the line of text the panel
// shows: the argument that says what the call is about, or else the whole.
func formatToolArgs(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]any); ok {
		for _, k := range []string{"Query", "query", "path", "file", "TargetFile", "SearchPath", "CommandLine"} {
			if val, ok := m[k].(string); ok && val != "" {
				return val
			}
		}
		if b, err := json.Marshal(m); err == nil {
			return string(b)
		}
	}
	return fmt.Sprint(v)
}
