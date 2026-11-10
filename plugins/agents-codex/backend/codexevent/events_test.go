package codexevent

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

type collector struct {
	steps []*agentrun.Response
}

func (c *collector) Send(resp *agentrun.Response) { c.steps = append(c.steps, resp) }

func TestAnAgentMessageBecomesText(t *testing.T) {
	var sink collector

	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": "The answer."}})
	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": ""}})

	if len(sink.steps) != 1 || sink.steps[0].GetText() != "The answer." {
		t.Errorf("steps %v", sink.steps)
	}
}

func TestAToolCallStartsAndFinishes(t *testing.T) {
	var sink collector

	Handle(&sink, map[string]any{"type": "item.started", "item": map[string]any{
		"type": "mcp_tool_call", "name": "read_note", "arguments": map[string]any{"path": "a.md"},
	}})
	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "mcp_tool_call"}})

	if len(sink.steps) != 2 {
		t.Fatalf("steps %v", sink.steps)
	}
	if call := sink.steps[0].GetToolCall(); call.Tool != "read_note" || call.About != `{"path":"a.md"}` {
		t.Errorf("tool call is %v", sink.steps[0])
	}
	if !sink.steps[1].IsToolFinished() {
		t.Errorf("expected a finished tool, got %v", sink.steps[1])
	}
}

func TestAToolCallNamesItsToolByTheToolField(t *testing.T) {
	var sink collector

	Handle(&sink, map[string]any{"type": "item.started", "item": map[string]any{
		"type": "mcp_tool_call", "server": "numen", "tool": "search_notes", "arguments": map[string]any{"q": "x"},
	}})

	if len(sink.steps) != 1 || sink.steps[0].GetToolCall().Tool != "search_notes" {
		t.Errorf("steps %v", sink.steps)
	}
}

func TestOtherEventsSendNothing(t *testing.T) {
	var sink collector

	Handle(&sink, map[string]any{"type": "item.completed"})
	Handle(&sink, map[string]any{"type": "item.started", "item": map[string]any{"type": "agent_message", "text": "no"}})
	Handle(&sink, map[string]any{"type": "other", "item": map[string]any{"type": "mcp_tool_call", "tool": "t"}})
	Handle(&sink, map[string]any{"type": "item.started", "item": map[string]any{"type": "other"}})
	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "other"}})
	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "reasoning", "text": "hidden"}})

	if len(sink.steps) != 0 {
		t.Errorf("steps %v", sink.steps)
	}
}

func TestFieldReadersIgnoreValuesOfAnotherType(t *testing.T) {
	if got := getString(map[string]any{"k": 123}, "k"); got != "" {
		t.Errorf("getString non-string = %q, want empty", got)
	}
	if got := getMap(map[string]any{"k": "not-map"}, "k"); got != nil {
		t.Errorf("getMap non-map = %v, want nil", got)
	}
	if got := getToolArguments(map[string]any{"arguments": make(chan int)}); got != "" {
		t.Errorf("getToolArguments channel = %q, want empty", got)
	}
}
