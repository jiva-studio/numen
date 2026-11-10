package events

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

type recorder struct {
	steps      []*agentrun.Response
	counted    int
	sessions   []string
	isFinished bool
}

func (r *recorder) Send(resp *agentrun.Response) { r.steps = append(r.steps, resp) }
func (r *recorder) CountStep()                   { r.counted++ }
func (r *recorder) SetSession(session string)    { r.sessions = append(r.sessions, session) }
func (r *recorder) Finish()                      { r.isFinished = true }

func TestJsonFieldsOfAnotherTypeAreEmpty(t *testing.T) {
	if got := getString(map[string]any{"k": 123}, "k"); got != "" {
		t.Errorf("getString non-string = %q, want empty", got)
	}
	if got := getMap(map[string]any{"k": "not-map"}, "k"); got != nil {
		t.Errorf("getMap non-map = %v, want nil", got)
	}
}

func TestEventsBecomeSteps(t *testing.T) {
	r := &recorder{}
	handle := NewHandler(r)

	// Top-level thinking, content, delta, result, stopped, and tool_calls edge cases.
	handle(map[string]any{
		"conversation_id": "s-top",
		"thinking":        "pondering",
		"content":         "main content",
		"delta":           map[string]any{"text": "step delta"},
		"result":          "result text",
		"stopped":         "stopped reason",
		"tool_calls": []any{
			"not-a-map",
			map[string]any{"name": "bad_args_tool", "arguments": make(chan int)},
		},
	})
	if len(r.sessions) != 1 || r.sessions[0] != "s-top" {
		t.Errorf("sessions = %v, want s-top", r.sessions)
	}
	// Empty and non-matching variants emit nothing.
	countBefore := len(r.steps)
	handle(map[string]any{
		"thinking":   "",
		"content":    "",
		"delta":      map[string]any{"text": ""},
		"result":     "",
		"stopped":    "",
		"tool_calls": "not-a-slice",
	})
	handle(map[string]any{"delta": "not-a-map"})
	if len(r.steps) != countBefore {
		t.Errorf("expected no steps for empty fields, got %d new steps", len(r.steps)-countBefore)
	}

	// Stream events: non-map event, non-map content_block, non-tool_use block, delta text and partial_json, and content_block_stop.
	handle(map[string]any{"type": "stream_event", "event": "not-map"})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_start", "content_block": "not-map"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_start", "content_block": map[string]any{"type": "text"}}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_stop"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": "not-map"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": map[string]any{"text": ""}}})

	// Tool use with non-map input, then unmarshalable input, then valid input with partial_json and a stop.
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":          "content_block_start",
		"content_block": map[string]any{"type": "tool_use", "name": "t1", "input": "not-map"},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":          "content_block_start",
		"content_block": map[string]any{"type": "tool_use", "name": "t2", "input": map[string]any{"ch": make(chan int)}},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":          "content_block_start",
		"content_block": map[string]any{"type": "tool_use", "name": "t3", "input": map[string]any{"k": "v"}},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":  "content_block_delta",
		"delta": map[string]any{"text": "stream-text", "partial_json": "_extra"},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_stop"}})

	// CLI events: default without conversation_id, a result with a status other than success and no error, and a result with no response.
	handle(map[string]any{"event": "init"})
	handle(map[string]any{"event": "result", "result": map[string]any{"status": "CANCELED", "error": ""}})
	handle(map[string]any{"event": "result", "result": map[string]any{"status": "SUCCESS", "response": ""}})

	steps := r.steps
	if len(steps) < 7 {
		t.Fatalf("expected at least 7 steps, got %d: %+v", len(steps), steps)
	}
	if !steps[0].IsModelStarted() {
		t.Errorf("step[0] = %+v, want ModelStarted", steps[0])
	}
	if steps[1].GetToolCall() == nil || steps[1].GetToolCall().Tool != "bad_args_tool" || steps[1].GetToolCall().About != "" {
		t.Errorf("step[1] = %+v, want bad_args_tool with empty About", steps[1])
	}
	if steps[len(steps)-2].GetToolCall() == nil || steps[len(steps)-2].GetToolCall().Tool != "t3" || steps[len(steps)-2].GetToolCall().About != `{"k":"v"}_extra` {
		t.Errorf("streamed tool call = %+v", steps[len(steps)-2])
	}
	if steps[len(steps)-1].GetStopped() != "the run ended with status CANCELED" {
		t.Errorf("last step = %+v, want status CANCELED", steps[len(steps)-1])
	}
	if !r.isFinished {
		t.Error("a result event did not finish the run")
	}
}
