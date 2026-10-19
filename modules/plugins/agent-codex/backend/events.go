package main

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type eventReader struct {
	work  *agentrun.Work
	limit int
	done  int
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
	r.handleCodexEvent(rawMap)
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

func (r *eventReader) reportLimit() {
	r.send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_Stopped{
			Stopped: fmt.Sprintf("reached maximum limit of %d steps", r.limit),
		},
	})
}
