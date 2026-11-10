package codexevent

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

type collector struct {
	sent []*agentrun.Response
}

func (c *collector) Send(resp *agentrun.Response) { c.sent = append(c.sent, resp) }

func TestAnAgentMessageBecomesText(t *testing.T) {
	var sink collector

	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": "The answer."}})
	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "agent_message", "text": ""}})

	if len(sink.sent) != 1 || sink.sent[0].GetText() != "The answer." {
		t.Errorf("sent %v", sink.sent)
	}
}

func TestAToolCallStartsAndFinishes(t *testing.T) {
	var sink collector

	Handle(&sink, map[string]any{"type": "item.started", "item": map[string]any{
		"type": "mcp_tool_call", "name": "read_note", "arguments": map[string]any{"path": "a.md"},
	}})
	Handle(&sink, map[string]any{"type": "item.completed", "item": map[string]any{"type": "mcp_tool_call"}})

	if len(sink.sent) != 2 {
		t.Fatalf("sent %v", sink.sent)
	}
	if call := sink.sent[0].GetToolCall(); call.Tool != "read_note" || call.About != `{"path":"a.md"}` {
		t.Errorf("tool call is %v", sink.sent[0])
	}
	if !sink.sent[1].IsToolFinished() {
		t.Errorf("expected a finished tool, got %v", sink.sent[1])
	}
}

func TestAToolCallNamesItsToolByTheToolField(t *testing.T) {
	var sink collector

	Handle(&sink, map[string]any{"type": "item.started", "item": map[string]any{
		"type": "mcp_tool_call", "server": "numen", "tool": "search_notes", "arguments": map[string]any{"q": "x"},
	}})

	if len(sink.sent) != 1 || sink.sent[0].GetToolCall().Tool != "search_notes" {
		t.Errorf("sent %v", sink.sent)
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

	if len(sink.sent) != 0 {
		t.Errorf("sent %v", sink.sent)
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
