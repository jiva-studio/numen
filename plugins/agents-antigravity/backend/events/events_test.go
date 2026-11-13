package events

import (
	"errors"
	"slices"
	"testing"
)

type recorder struct {
	reports    []string
	sessions   []string
	isFinished bool
	refusal    error
}

func (r *recorder) SendText(text string) error {
	r.reports = append(r.reports, "text "+text)
	return r.refusal
}

func (r *recorder) SendToolCall(tool, arguments string) error {
	r.reports = append(r.reports, "call "+tool+" "+arguments)
	return r.refusal
}

func (r *recorder) SetSession(session string) { r.sessions = append(r.sessions, session) }

func (r *recorder) Stop(reason string) { r.reports = append(r.reports, "stop "+reason) }

func (r *recorder) Finish() { r.isFinished = true }

func TestJsonFieldsOfAnotherTypeAreEmpty(t *testing.T) {
	if got := getString(map[string]any{"k": 123}, "k"); got != "" {
		t.Errorf("getString non-string = %q, want empty", got)
	}
	if got := getMap(map[string]any{"k": "not-map"}, "k"); got != nil {
		t.Errorf("getMap non-map = %v, want nil", got)
	}
}

func TestEventsAreReportedInTheOrderTheProgramWritesThem(t *testing.T) {
	r := &recorder{}
	handler := NewHandler(r)
	handle := handler.Handle

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
	// Empty and non-matching variants report nothing.
	handle(map[string]any{
		"thinking": "", "content": "", "delta": map[string]any{"text": ""}, "result": "", "stopped": "", "tool_calls": "not-a-slice",
	})
	handle(map[string]any{"delta": "not-a-map"})
	handle(map[string]any{"type": "stream_event", "event": "not-map"})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_start", "content_block": "not-map"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_start", "content_block": map[string]any{"type": "text"}}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_stop"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": "not-map"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": map[string]any{"text": ""}}})
	// A call streamed in pieces is reported once with its whole arguments.
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_start", "content_block": map[string]any{"type": "tool_use", "name": "t1", "input": "not-map"},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_start", "content_block": map[string]any{"type": "tool_use", "name": "t3", "input": map[string]any{"k": "v"}},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "delta": map[string]any{"text": "stream-text", "partial_json": "_extra"},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_stop"}})
	handle(map[string]any{"event": "init", "conversation_id": "s-init"})
	handle(map[string]any{"event": "step_update", "step_update": map[string]any{"step_type": "agent_response", "text_delta": "Raised beds.", "conversation_id": "s-step"}})
	handle(map[string]any{"event": "result", "result": map[string]any{"status": "SUCCESS", "response": "Raised beds."}})
	handle(map[string]any{"event": "result", "result": map[string]any{"status": "CANCELED", "error": ""}})

	want := []string{
		"call bad_args_tool ",
		"text main content",
		"text step delta",
		"text result text",
		"stop stopped reason",
		"text stream-text",
		`call t3 {"k":"v"}_extra`,
		"text Raised beds.",
		"stop the run ended with status CANCELED",
	}
	if !slices.Equal(r.reports, want) {
		t.Errorf("reports = %q, want %q", r.reports, want)
	}
	if !slices.Equal(r.sessions, []string{"s-top", "s-init", "s-step"}) {
		t.Errorf("sessions = %v", r.sessions)
	}
	if !r.isFinished {
		t.Error("a result event did not finish the run")
	}
}

func TestTheResponseOfTheResultIsReportedWhenNoTextWasStreamed(t *testing.T) {
	r := &recorder{}
	NewHandler(r).Handle(map[string]any{"event": "result", "result": map[string]any{"status": "success", "response": "Raised beds.", "conversation_id": "s-1"}})

	if !slices.Equal(r.reports, []string{"text Raised beds."}) || !slices.Equal(r.sessions, []string{"s-1"}) {
		t.Errorf("reports = %q, sessions = %v", r.reports, r.sessions)
	}
}

func TestASinkThatRefusedAReportIsHandedNoOther(t *testing.T) {
	refusal := errors.New("closed")
	r := &recorder{refusal: refusal}
	handler := NewHandler(r)

	handler.Handle(map[string]any{"content": "one"})
	handler.SendText("two")

	if len(r.reports) != 1 || !errors.Is(handler.Err(), refusal) {
		t.Errorf("reports = %q, err = %v", r.reports, handler.Err())
	}
}
