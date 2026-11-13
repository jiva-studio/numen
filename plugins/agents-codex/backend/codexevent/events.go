// Package codexevent hands what the JSON events that Codex prints during a run report to a sink: the
// thread of the run, the messages and the reasoning of the agent, and the calls of the tools of the vault.
package codexevent

import (
	"encoding/json"
)

// Sink receives what the events of a run report. A call is reported when it starts and again when
// it ends. RecordThread receives the thread the run works in, which a later run can continue.
type Sink interface {
	RecordThread(id string)
	SendText(text string) error
	SendThought(text string) error
	SendToolCall(id, tool, arguments string) error
	SendToolResult(id string, isFailed bool) error
}

// Handle hands sink what an event reports and returns the error of the sink. An event that reports
// nothing is ignored.
func Handle(sink Sink, rawMap map[string]any) error {
	evType := getString(rawMap, "type")
	if evType == "thread.started" {
		if id := getString(rawMap, "thread_id"); id != "" {
			sink.RecordThread(id)
		}
		return nil
	}
	item := getMap(rawMap, "item")
	if item == nil {
		return nil
	}
	switch {
	case evType == "item.completed" && item["type"] == "agent_message":
		if text := getString(item, "text"); text != "" {
			return sink.SendText(text)
		}
	case evType == "item.completed" && item["type"] == "reasoning":
		if text := getString(item, "text"); text != "" {
			return sink.SendThought(text)
		}
	case evType == "item.started" && item["type"] == "mcp_tool_call":
		return sink.SendToolCall(getString(item, "id"), getToolName(item), getToolArguments(item))
	case evType == "item.completed" && item["type"] == "mcp_tool_call":
		return sink.SendToolResult(getString(item, "id"), getString(item, "status") == "failed" || item["error"] != nil)
	}
	return nil
}

func getToolName(item map[string]any) string {
	if tool := getString(item, "tool"); tool != "" {
		return tool
	}
	return getString(item, "name")
}

func getToolArguments(item map[string]any) string {
	if item["arguments"] == nil {
		return ""
	}
	args, err := json.Marshal(item["arguments"])
	if err != nil {
		return ""
	}
	return string(args)
}
