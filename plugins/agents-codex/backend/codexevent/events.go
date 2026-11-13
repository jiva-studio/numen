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
	item := getMap(rawMap, "item")
	switch getString(rawMap, "type") {
	case "thread.started":
		if id := getString(rawMap, "thread_id"); id != "" {
			sink.RecordThread(id)
		}
	case "item.started":
		if item != nil && item["type"] == "mcp_tool_call" {
			return sink.SendToolCall(getString(item, "id"), getToolName(item), getToolArguments(item))
		}
	case "item.completed":
		if item != nil {
			return handleCompletedItem(sink, item)
		}
	}
	return nil
}

func handleCompletedItem(sink Sink, item map[string]any) error {
	text := getString(item, "text")
	switch {
	case item["type"] == "agent_message" && text != "":
		return sink.SendText(text)
	case item["type"] == "reasoning" && text != "":
		return sink.SendThought(text)
	case item["type"] == "mcp_tool_call":
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
