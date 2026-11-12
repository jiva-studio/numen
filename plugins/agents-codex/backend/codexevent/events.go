// Package codexevent turns the JSON events that Codex prints during a run into the steps of an agent
// response.
package codexevent

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
)

// Sink receives the steps that the events of a run produce.
type Sink interface {
	Send(resp *agentrun.Response)
}

// Handle sends the step that an event stands for to sink. An event that stands for no step is ignored.
func Handle(sink Sink, rawMap map[string]any) {
	evType := getString(rawMap, "type")
	item := getMap(rawMap, "item")
	if item == nil {
		return
	}

	switch {
	case evType == "item.completed" && item["type"] == "agent_message":
		if text := getString(item, "text"); text != "" {
			sink.Send(&agentrun.Response{Step: &agentrun.TextStep{Text: text}})
		}
	case evType == "item.started" && item["type"] == "mcp_tool_call":
		sink.Send(&agentrun.Response{
			Step: &agentrun.ToolCallStep{
				ToolCall: &toolcall.ToolCall{
					Tool:   getToolName(item),
					About:  getToolArguments(item),
					Length: 0,
					Path:   "",
					Range:  nil,
					Call:   "",
					Kind:   "",
				},
			},
		})
	case evType == "item.completed" && item["type"] == "mcp_tool_call":
		sink.Send(&agentrun.Response{
			Step: &agentrun.ToolFinishedStep{},
		})
	}
}

func getToolName(item map[string]any) string {
	if tool := getString(item, "tool"); tool != "" {
		return tool
	}
	name := getString(item, "name")
	return name
}

func getToolArguments(item map[string]any) string {
	args, err := json.Marshal(item["arguments"])
	if err != nil {
		return ""
	}
	return string(args)
}
